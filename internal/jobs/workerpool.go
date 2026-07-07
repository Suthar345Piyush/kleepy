// worker pool implementation

package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/Suthar345Piyush/videoclippingpipeline/internal/database"
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

// run function will start the pool and blocks/stops until ctx is cancelled or any fatal error occurs, it will launches one worker goroutine and one poll goroutine

func (p *Pool) Run(ctx context.Context) error {

	return err
}
