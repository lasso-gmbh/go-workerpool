# Go-Workerpool

Inspired by the following sources:
- [concurrency limiting workerpool](https://github.com/gammazero/workerpool)
- [writing non-blocking parallel services](http://www.goldsborough.me/go/2020/12/06/12-24-24-non-blocking_parallelism_for_services_in_go/)


This is a workerpool implementation that is concurrency limiting and never blocks any requests. The aim was to write a worker pool that is simpler and that functions slightly differently in alocating and dismantling workers.


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
