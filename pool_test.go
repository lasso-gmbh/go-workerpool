package workerpool

import (
	"context"
	"fmt"
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
func TestNWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPool(ctx, 4)

	requests := []string{"greta", "loves", "cheese", "yo"}
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

	// wait gor workers to start up
	time.Sleep(100 * time.Millisecond)

	if p.slotsInUse() != 4 {
		t.Errorf("incorrect number of slots in use. got :%d want %d", p.slotsInUse(), 4)
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

func TestReueseWorker(t *testing.T) {
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

	p.StopWait()

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

func TestStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPool(ctx, 1)

	nRequests := 5
	doneChan := make(chan struct{}, nRequests)

	for i := 0; i < nRequests; i++ {
		r := i
		var f Task = func() {
			<-time.After(150 * time.Millisecond)
			fmt.Println(r)
			doneChan <- struct{}{}
		}
		p.EnqueueTask(f)
	}
	p.stop(false)
}
