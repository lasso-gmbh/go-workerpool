package workerpool

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	workerIdleTimeSeconds int = 2
)

// Task simulates some task/request to handle
type Task func()

// Pool is a concurrent task execution engine.
// It can limit the number of concurrent processes and
// will not block when a request comes in. All incoming
// requests that cannot be processed imediately are stored
// in an unbuffered channel.
type Pool struct {
	// semaphore
	// limits the number of concurrent go routines in the pool
	//
	// I liked how Peter limited concurrency in his example...
	// http://www.goldsborough.me/go/2020/12/06/12-24-24-non-blocking_parallelism_for_services_in_go/
	sema chan int

	// queue is where all incoming requests are stored.
	//
	// The channel is unbuffered allowing for a lot of
	// requests to come in, without blocking the client.
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
}

// runTask executes the task.
func runTask(t Task) {
	t()
}

// Wait will block until all tasks in the queue have
// finished if requested to wait. Otherwise just wait for the workers
// to finish.
func (p *Pool) stop(wait bool) {
	if !wait {
		// wait for workers to finish
		p.wg.Wait()
		return
	}

	// dont know how to handle this...
	// if stop is called imediately after a task is enqueued
	// it may take longer for the task to be assigned (and the wg to be incremented)
	// then the time it takes the main thread to reach this function
	// in which case wg.Wait() will not block.
	// so I added a small sleep time, but this is hacky solution
	time.Sleep(100 * time.Millisecond)
	p.wg.Wait()

	// other solution ...
	// create a task for all slots in which the done channel
	// recieves an empty struct
	// block until we have drained the done channel
	// works but I dont like it because it wouldnt work
	// if our requests were not functions

	// n := cap(p.sema)
	// doneChan := make(chan struct{}, n)
	// for i := 0; i < n; i++ {
	// 	f := func() {
	// 		doneChan <- struct{}{}
	// 	}
	// 	p.EnqueueTask(Task(f))
	// }
	// for i := 0; i < n; i++ {
	// 	<-doneChan
	// }
	// close(doneChan)
}

// StopWait qwill exit the main event loop imediately.
func (p *Pool) StopWait() {
	p.stop(true)
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
				log.Printf("worker closing : belt closed\n")
				break loop
			}
			log.Println("worker recieved work")

			// execute the task
			runTask(t)

			// drain the timer if it expired during processing and reset it
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(time.Duration(workerIdleTimeSeconds) * time.Second)
		case <-timer.C:
			log.Printf("worker closing : no work\n")
			break loop
		}
	}
	// replenish semaphore
	// notify pool that a worker is closing
	<-p.sema
	p.wg.Done()
}

// assign assigns a task to a worker.
//
// If all workers are busy, the task is placed in the queue for task execution.
// if there are slots available, a worker is created and assigend the task.
//
// The routine blocks if all workers are busy.
func (p *Pool) assign(t Task) {
	select {
	case p.sema <- 1:
		log.Printf("creating new worker\n")
		p.wg.Add(1)
		go p.worker()
	default:
	}
	// place task in queue for execution
	p.nextTask <- t
}

// loop takes tasks from the queue and assigns them to workers.
//
// The loop will exit if the queue channel is closed or if the context is cancelled
func (p *Pool) loop(ctx context.Context) {
	for {
		select {
		case t, ok := <-p.queue:
			if !ok {
				log.Println("queue has been closed")
				return
			}
			// need to catch the exit signal here

			// blocks here
			p.assign(t)
		case <-ctx.Done():
			return
		}
	}
}

// EnqueueTask adds a task to the queue.
//
// This call should never block.
func (p *Pool) EnqueueTask(t Task) {
	p.queue <- t

}

// NewPool returns a running worker pool.
//
// Parameter limit sets the number of concrrent jobs that can
// be run at once.
func NewPool(ctx context.Context, limit int) *Pool {
	p := &Pool{
		queue:    make(chan Task),
		sema:     make(chan int, limit),
		nextTask: make(chan Task, 1),
	}

	go p.loop(ctx)

	return p
}
