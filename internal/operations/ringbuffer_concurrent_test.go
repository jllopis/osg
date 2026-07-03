package operations

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// TestRingBuffer_WriteDuringUnsubscribe stresses the race between ringBuffer
// writers and subscribers that disconnect: a writer must never send on a
// channel a subscriber has already closed. Run with -race; before the fix this
// panicked with "send on closed channel".
func TestRingBuffer_WriteDuringUnsubscribe(t *testing.T) {
	t.Parallel()
	rb := newRingBuffer(128)

	const writers = 4
	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Continuous writers.
	for w := range writers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
					_, _ = fmt.Fprintf(rb, "w%d line %d\n", id, i)
				}
			}
		}(w)
	}

	// Churn of short-lived subscribers that cancel (and trigger close).
	for range 200 {
		ctx, cancel := context.WithCancel(context.Background())
		ch := rb.Subscribe(ctx)
		// Drain a little so the writer's non-blocking send can hit the channel.
		go func() {
			for range ch {
			}
		}()
		cancel()
	}

	close(stop)
	wg.Wait()
}
