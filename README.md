# Go-Workerpool

Inspired by the following sources:
- [workerpool](https://github.com/gammazero/workerpool)
- [tickler pattern](http://www.goldsborough.me/go/2020/12/06/12-24-24-non-blocking_parallelism_for_services_in_go/)

This is another workerpool implementation with many ideas taken from [workerpool](https://github.com/gammazero/workerpool), but with a few differences, and not as functional.

Instead of using a mutex and worker counter to manage the number of workers concurrently working through tasks, I used a semaphore in the form a buffered channel thanks to [peter](http://www.goldsborough.me/go/2020/12/06/12-24-24-non-blocking_parallelism_for_services_in_go/).

### Conveyor belt worker pattern

While many tasks are being enqueued, the pool will continue to spawn new workers until the concurrency limit is reached. The workers all listen on a buffered channel which acts as a conveyor belt and grab a task as soon as they have finished the previous task. I like the idea of the workers managing their own lifetime. That is if a worker has been idle for too long, a timeout will trigger and the worker exits. The worker pool isnt interested in how long a worker has been idle.


Usage example:
```go    
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
p := NewPool(ctx, 3)

requests := []string{"greta", "loves", "cheese"}


for _, r := range requests {
    r := r
    var f Task = func() {
        fmt.Println(r)
    }
    p.EnqueueTask(f)
}
p.StopWait()
```
