// worker pool implementation

package jobs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
	"github.com/Suthar345Piyush/videoclippingpipeline/internal/logger"
)

// polling constants

const (
	pollInitial    = 100 * time.Millisecond
	pollMax        = 30 * time.Second
	pollMultiplier = 2
)

// general purpose worker pool according to the db jobs table

/*

 -> one poller goroutine will call dequeue next job - with their are N pollers, all N fight for the same sqlite write lock simultaneously, one poller serialises dequeue writes, and eliminates the lock contention, and that will costs exactly only one extra goroutine



 -> we have to perform exponential backoff when the queue is empty, implements a fixed ticker, a genuinely idle pool uses around 2 sqlite reads / minute instead of 600


 -> channel for work have the exact size of pool size

 -> context cancellation as the shutdown signal using SIGTERM, it will close the work channel, and their will zero goroutine leaks

 -> performing retry on application level, not on database level, after a failure we also check that attempts >= max_attempts before requeue or permanently failure


*/

// struct for pool

type Pool struct {
	queries  *database.Queries
	registry *Registry
	poolsize int // pool size will be 2
	workCh   chan database.Job
	wg       sync.WaitGroup
}

// making new pool, and poolsize are basically number of concurrent handler goroutines
// will create work channel inside it
// it will return a new pool
// size of work channel will be same as size of poolsize

func NewPool(queries *database.Queries, registry *Registry, poolsize int) *Pool {
	if poolsize <= 0 {
		poolsize = 2
	}

	return &Pool{
		queries:  queries,
		registry: registry,
		poolsize: poolsize,

		workCh: make(chan database.Job, poolsize),
	}

}

// run function will start the pool and blocks/stops until ctx is cancelled or any error occurs, it will launches one worker goroutine and one poll goroutine

func (p *Pool) Run(ctx context.Context) error {

	// essential logging

	log := logger.FromContext(ctx).With(slog.String("component", "job_pool"))

	log.Info("starting job pool", slog.Int("pool_size", p.poolsize), slog.Any("registered_job_types", p.registry.Registered()))

	// launching the worker go routines first, worker will read from worker channel (workCh) until it is closed

	for range p.poolsize {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.workerLoop(ctx)
		}()
	}

	// the poller will runs in the caller's goroutines

	err := p.pollerLoop(ctx)
	close(p.workCh)
	p.wg.Wait()

	log.Info("job pool stopped")

	return err

}

// poller Loop - this is single go routine that will read from the database queue, it will use exponential backoff when queue is empty to avoid wasting resources

func (p *Pool) pollerLoop(ctx context.Context) error {

	// logging

	log := logger.FromContext(ctx).With(slog.String("component", "job_poller"))
	backoff := pollInitial

	for {
		// shutdown: the context will be cancelled
		select {
		case <-ctx.Done():
			log.Info("poller shutting down")
			return nil
		default:
		}

		job, err := p.queries.DequeueNextJob(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {

				// this means queue is empty, so do backoff and wait

				log.Debug("queue empty, back off", slog.Duration("backoff", backoff))

				select {
				case <-ctx.Done():
					log.Info("poller shuting down during backoff")
					return nil

				case <-time.After(backoff):
				}

				// actually doing the backoff
				backoff = min(backoff*pollMultiplier, pollMax)
				continue

			}

			// if any error occurs in database (like disk is full, or corrupt), well this will not stop the pool

			log.Error("DequeueNextJob failed", slog.String("error", err.Error()))

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(pollInitial):
			}
			continue

		}

		// if we got the job - reset the backoff

		backoff = pollInitial
		log.Info("dequeued job", slog.String("job_id", job.ID), slog.String("job_type", job.JobType), slog.String("attempts", job.Attempts))

		// send to workers, this will be block when all workers are busy, which cause back-pressure, so not dequeue more than we can run

		select {
		case p.workCh <- job:
		case <-ctx.Done():

			log.Warn("context cancelled in mid, requeuing job", slog.String("job_id", job.ID))

			if _, err := p.queries.RequeueJob(ctx, job.ID); err != nil {
				log.Error("failed to requeue job on shutdown", slog.String("job_id", job.ID), slog.String("error", err.Error()))
			}

			return nil

		}

	}

}

// submit function will enqueue the job into job table and return the created row , this is the entry point of workflow layer

// first priority is integer, and payload must be value that json.Marshal can encode them

func (p *Pool) Submit(ctx context.Context, jobType JobType, payload any, priority int) (database.Job, error) {

	raw, err := MarshalPayload(payload)
	if err != nil {
		return database.Job{}, fmt.Errorf("workerpool.Submit: %w", err)
	}

	id, err := generateID()
	if err != nil {
		return database.Job{}, fmt.Errorf("workerpool.Submit: failed to generate job ID: %w ", err)
	}

	// enqeueued the job

	job, err := p.queries.EnqueueJob(ctx, database.EnqueueJobParams{
		ID:       id,
		JobType:  string(jobType),
		Payload:  raw,
		Priority: fmt.Sprintf("%d", priority),
	})

	if err != nil {
		return database.Job{}, fmt.Errorf("workerpool.Submit: failed to enqueue %q job: %w", jobType, err)
	}

	return job, nil

}

// worker loop will run in each of the worker go routines, it will read jobs from the worker channel, give them to register handler, and update the database with the result

func (p *Pool) workerLoop(ctx context.Context) {

	// firstly logging is essential
	log := logger.FromContext(ctx).With(slog.String("component", "job_worker"))

	// iterate on the work channel

	for job := range p.workCh {

		jobLog := log.With(
			slog.String("job_id", job.ID),
			slog.String("job_type", job.JobType),
			slog.String("attempts", job.Attempts),
		)

		// handing off to the handler registry

		handler, ok := p.registry.Get(JobType(job.JobType))

		// if no handler was registered, then mark it as failed , don't retry
		if !ok {
			jobLog.Error("no handler registered for job type, marking failed")
			p.markFailed(ctx, job, fmt.Errorf("no handler is registered for job type %q", job.JobType))
			continue
		}

		jobLog.Info("handling job")
		start := time.Now()

		err := handler.Handle(ctx, job)

		elapsed := time.Since(start)

		if err != nil {
			jobLog.Error("job handler returned error", slog.String("error", err.Error()), slog.Duration("elapsed", elapsed))

			p.handleFailure(ctx, job, err)
			continue
		}

		jobLog.Info("job completed", slog.Duration("elapsed", elapsed))

		if _, dbErr := p.queries.CompleteJob(ctx, job.ID); dbErr != nil {
			jobLog.Error("failed to mark job done", slog.String("error", dbErr.Error()))
		}

	}

}

// markFailed helper function, to mark failed the job permanently

func (p *Pool) markFailed(ctx context.Context, job database.Job, err error) {
	log := logger.FromContext(ctx)

	if _, dbErr := p.queries.FailJob(ctx, database.FailJobParams{
		LastError: sql.NullString{String: err.Error(), Valid: true},
		ID:        job.ID,
	}); dbErr != nil {
		log.Error("failed to mark job as failed in DB",
			slog.String("job_id", job.ID),
			slog.String("error", dbErr.Error()),
		)
	}
}

// function for handleFailure - it will decide to requeue the job or fail it permanently, it wil compare for the attempts should be less than max attempts

func (p *Pool) handleFailure(ctx context.Context, job database.Job, err error) {

	log := logger.FromContext(ctx)

	// get the attempts and max attempts that you have

	attempts := parseIntField(job.Attempts)
	maxAttempts := parseIntField(job.MaxAttempts)

	// if attempts >= max attempts, then retries are done, failed that job permanently

	if attempts >= maxAttempts {
		log.Warn("job exceeded max attempts, marking it failed",
			slog.String("job_id", job.ID),
			slog.String("job_type", job.JobType),
			slog.Int("attempts", attempts),
			slog.Int("max_attempts", maxAttempts),
		)
		p.markFailed(ctx, job, err)
		return
	}

	// if attempts are still remaining than retry

	log.Info("requeuing job for retry",
		slog.String("job_id", job.ID),
		slog.String("job_type", job.JobType),
		slog.Int("attempts", attempts),
		slog.Int("max_attempts", maxAttempts),
	)

	// requeue job sqlc generated functions

	if _, dbErr := p.queries.RequeueJob(ctx, job.ID); dbErr != nil {
		log.Error("failed tp requeue job",
			slog.String("job_id", job.ID),
			slog.String("error", dbErr.Error()),
		)
	}
}

// helper function for parsing the string json field like in our case (attempts, max_attempts) to integer

func parseIntField(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// generating the id - jobID creation using crypto/rand standard library

func generateID() (string, error) {

	// bytes slice
	b := make([]byte, 8)

	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}

	return "job-" + hex.EncodeToString(b), nil

}
