package workerpool

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNormalUse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPool(ctx, 4)

	requests := []string{"greta", "loves", "cheese"}
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
	time.Sleep(100 * time.Millisecond)

	if p.slotsInUse() != 3 {
		t.Errorf("incorrect number of slots in use. want %d got %d", 3, p.slotsInUse())
	}

	p.Wait()

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

func TestReuseWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPool(ctx, 1)

	requests := []string{"greta", "loves", "cheese", "fat cats"}
	nRequests := len(requests)
	doneChan := make(chan string, nRequests)

	for _, r := range requests[:3] {
		r := r
		var f Task = func() {
			fmt.Println(r)
			doneChan <- r
		}
		p.EnqueueTask(f)
	}

	time.Sleep(3 * time.Second)

	for _, r := range requests[3:] {
		r := r
		var f Task = func() {
			fmt.Println(r)
			doneChan <- r
		}
		p.EnqueueTask(f)
	}

	p.Wait()

	close(doneChan)
	resps := map[string]struct{}{}

	for r := range doneChan {
		resps[r] = struct{}{}
	}

	if len(resps) != nRequests {
		t.Error("did not receive all responses")
	}

	for _, req := range requests {
		if _, ok := resps[req]; !ok {
			t.Errorf("value not found in responses: %v", req)
		}
	}

}

func Benchmark4Workers(b *testing.B) {
	bench(4, b)
}
func Benchmark8Workers(b *testing.B) {
	bench(8, b)
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

	p.Wait()
}

func TestStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPool(ctx, 1)
	defer p.Stop()

	requests := []string{"greta", "loves", "cheese"}
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
}
