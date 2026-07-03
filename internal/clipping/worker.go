/*

the worker pool is goroutine pool dedicated for cutting clips from source video, it is not a general job queue, the general queue will dispatch the clip job payload, and this worker will execute it

-> workflow will be something like

1. worker with service and worker pool size (limited, to stay away from goroutine leaks)

2. then we will launch the pool size goroutine

3. will use till the buffer capacity, not more than that

4. at last we will drain (will take each and every stuff out of channel and clean the channel)

-> will use simple for range iteration to implement draining

*/

package clipping

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/logger"
)

// worker struct consist of service, poolsize, channels (clip and err), and waitgroup

// clipChan will have clip id's to process

// errChan contains one entry per failed clip

type Worker struct {
	service  *Service
	poolsize int
	clipChan chan string
	errChan  chan error
	wg       sync.WaitGroup
}

// new worker function will have worker struct with the buffered channel double of actual pool size

func NewWorker(service *Service, poolsize int) *Worker {
	if poolsize <= 0 {
		poolsize = 2
	}

	return &Worker{
		service:  service,
		poolsize: poolsize,
		clipChan: make(chan string, poolsize*2),
		errChan:  make(chan error, poolsize*2),
	}
}

// launching the goroutine pool, context cancellation will stop the worker after clip cutting completed

func (w *Worker) Start(ctx context.Context) {

	log := logger.FromContext(ctx)
	log.Info("starting clipping worker pool", slog.Int("pool_size", w.poolsize))

	// goroutine pool spawn

	for range w.poolsize {
		w.wg.Add(1)

		go func() {
			defer w.wg.Done()
			w.goProc(ctx)
		}()
	}
}

// submit will put the clip id into queue for processing, it can be blocked if the buffer is full

func (w *Worker) Submit(clipID string) {
	w.clipChan <- clipID
}

// drain function will close the input channel and clean the buffered channel, and wait for all in-flight clips to finish

func (w *Worker) Drain() {
	close(w.clipChan)
	w.wg.Wait()
	close(w.errChan)
}

// errors function to return read only errors and each failed clip process will make one error entry

func (w *Worker) Errors() <-chan error {
	return w.errChan
}

/* goProc is an per goroutine processing function, it will read the clip id from clip channel and process each one of them using process clip from service and if any error occurs then it send that error to the error channel */

func (w *Worker) goProc(ctx context.Context) {

	log := logger.FromContext(ctx)

	var clipID string

	for clipID := range w.clipChan {

		// checking for cancellation before starting each new clip
		// this will prevent new clip cuts from starting during graceful shutdown, while still letting the current goroutine exit cleanly

		if ctx.Err() != nil {
			log.Warn("clipping worker stopping: context cancelled", slog.String("skipped_clip", clipID))
		}

		for range w.clipChan {

		}

		return

	}

	// processing the clip

	_, err := w.service.ProcessClip(ctx, clipID)

	if err != nil {
		log.Error("clip processing failed", slog.String("clip_id", clipID), slog.String("error", err.Error()))
	}

	//Non Blocking Channel Operation - if the error channel is full, we will log the error instead of deadlocking here

	select {
	case w.errChan <- err:
	default:
		log.Warn("err channel is full, dropping clip error", slog.String("clip_id", clipID))
	}

}
