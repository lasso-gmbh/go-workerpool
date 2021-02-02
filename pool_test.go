package workerpool

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func Benchmark4Workers(b *testing.B) {
	bench(4, b)
}
func Benchmark8Workers(b *testing.B) {
	bench(8, b)
}

func Benchmark16Workers(b *testing.B) {
	bench(16, b)
}
func Benchmark64Workers(b *testing.B) {
	bench(64, b)
}

func bench(n int, b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := NewPool(ctx, n)

	var wg sync.WaitGroup
	wg.Add(b.N * n)

	b.ResetTimer()

	for j := 0; j < n; j++ {
		for i := 0; i < b.N; i++ {
			p.EnqueueTask(Task(func() {
				wg.Done()
			}))
		}
	}
	wg.Wait()
}

func TestReuseWorker(t *testing.T) {}

func TestMaxWorkers(t *testing.T) {}

func TestStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPool(ctx, 3)

	requests := []string{"greta", "loves", "cheese"}

	// only one task should be processed
	doneChan := make(chan string, 1)

	for _, r := range requests {
		r := r
		var f Task = func() {
			fmt.Println(r)
			doneChan <- r
		}
		p.EnqueueTask(f)
	}
	p.Stop()

	close(doneChan)
	req := <-doneChan
	if req != "greta" {
		t.Errorf("lost greta, found '%v' instead", req)
	}
}

func TestWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPool(ctx, 2)

	requests := []string{"greta", "loves", "cheese"}

	// all tasks should be processed
	doneChan := make(chan string, len(requests))

	for _, r := range requests {
		r := r
		var f Task = func() {
			fmt.Println(r)
			time.Sleep(1 * time.Second)
			doneChan <- r
		}
		p.EnqueueTask(f)
	}
	p.StopWait()

	close(doneChan)
	resps := map[string]struct{}{}

	for r := range doneChan {
		resps[r] = struct{}{}
	}

	if len(resps) != len(requests) {
		t.Error("did not receive all responses")
	}

	for _, req := range requests {
		if _, ok := resps[req]; !ok {
			t.Errorf("value not found in responses: %v", req)
		}
	}
}
