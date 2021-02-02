package workerpool

import (
	"context"
	"sync"
	"time"
)

const (
	// number of seconds a worker will remain idle before it terminates itself
	workerIdleTimeSeconds int = 2

	// size of the queue channel
	bufferSize int = 10000
)

// Task simulates some task/request to handle
type Task func()

// Pool is a concurrent task execution engine.
// It can limit the number of concurrent processes and
// will not block when a request comes in. All incoming
// requests that cannot be processed imediately are stored
// in an buffered channel.
type Pool struct {
	// semaphore
	// limits the number of concurrent go routines in the pool
	//
	// I liked how Peter limited concurrency in his example...
	// http://www.goldsborough.me/go/2020/12/06/12-24-24-non-blocking_parallelism_for_services_in_go/
	sema chan int

	// queue is where all incoming requests are stored.
	//
	// The channel is buffered, this can be an issue if a lot of requests start coming in
	queue chan Task

	// nextTask is a buffered channel of length one.
	//
	// It acts as a kind of conveyor belt along which
	// tasks are passed to workers listening on this channel.
	//
	// If all workers are busy, this channel will block the main
	// loop until a worker can continue on the next task.
	nextTask chan Task

	// wg enables the pool to wait for all workers to finish.
	wg sync.WaitGroup

	// for settign stopLoop
	mu sync.Mutex

	// stopLoop is a hard stop that will terminate the main loop
	// preventing the pool from processing any remaining tasks in the queue
	stopLoop bool
}

// runTask executes the task.
func runTask(t Task) {
	t()
}

// Stop prevents workers from receiving anymore work and waits on
// those workers still working to finish their tasks.
func (p *Pool) Stop() {
	p.mu.Lock()
	p.stopLoop = true
	p.mu.Unlock()

	close(p.nextTask)
	p.wg.Wait()
}

// StopWait will wait for all workers to finish their task and the queue to be
// depleted.
func (p *Pool) StopWait() {
	close(p.queue)
	p.wg.Wait()
	// close(p.nextTask)
}

// slotsInUse should return the number of slots used by the pool.
//
// Returns the length of the semaphore.
func (p *Pool) slotsInUse() int {
	return len(p.sema)

}

// worker processes incoming tasks.
//
// Once it has completed a task, it will wait for more tasks to come in.
// If the worker has been idle for too long, the worker will exit.
//
// At the end the semaphore is replenished and the wait group notified.
func (p *Pool) worker() {
	var timer *time.Timer
	timer = time.NewTimer(1 * time.Second)

	// the main loop that makes the worker so cool
	// I tried to keep the pool implentation as simple as possible,
	// unfortunately a bit of complexity is needed somewhere...
loop:
	for {
		select {
		// the conveyor belt
		// all workers listen on this channel
		case t, ok := <-p.nextTask:
			if !ok {
				break loop
			}

			if t == nil {
				break loop
			}

			// execute the task
			runTask(t)

			// drain the timer if it expired during processing and reset it
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(time.Duration(workerIdleTimeSeconds) * time.Second)
		case <-timer.C:
			break loop
		}
	}
	// replenish semaphore
	// notify pool that a worker is closing
	<-p.sema
	p.wg.Done()
}

// tryNewWorker will try to create another worker if the pool has any free slots,
// and otherwise terminate.
func (p *Pool) tryNewWorker() {
	select {
	case p.sema <- 1:
		p.wg.Add(1)
		go p.worker()
	default:
	}
}

// loop takes tasks from the queue and pushes them into the nextTask channel where
// where workers grab their tasks.
//
// The loop will exit if a hard stop is set, regardless of whether the queue is empty or not.
// If the queue channel is closed, the loop will still drain the channel before closing.
//
// A context with timeout can also be used to run the loop for a limited amount of time.
func (p *Pool) loop(ctx context.Context) {
	for {
		select {
		case t, ok := <-p.queue:
			if !ok {
				return
			}
			if p.stopLoop {
				return
			}
			p.tryNewWorker()
			p.nextTask <- t
		case <-ctx.Done():
			return
		}
	}
}

// EnqueueTask adds a task to the queue.
//
// This call should never block.
//
// A new task is first pushed to the next task channel and the pool tries to
// create a worker.
// If the nextTask channel is full, the task is added to the queue.
func (p *Pool) EnqueueTask(t Task) {
	select {
	case p.nextTask <- t:
		p.tryNewWorker()
	default:
		p.queue <- t
	}

}

// NewPool returns a running worker pool.
//
// Parameter limit sets the number of concrrent jobs that can
// be run at once.
func NewPool(ctx context.Context, limit int) *Pool {
	if limit <= 0 {
		limit = 1
	}

	p := &Pool{
		queue:    make(chan Task, bufferSize),
		sema:     make(chan int, limit),
		nextTask: make(chan Task, 1),
	}

	go p.loop(ctx)

	return p
}
