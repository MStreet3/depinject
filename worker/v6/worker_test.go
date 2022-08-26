package v6

import (
	"context"
	"testing"
	"time"

	"github.com/mstreet3/depinject/heartbeat"
)

func TestBeatingRandIntStream(t *testing.T) {
	var (
		wantCount = 100
		d, _      = t.Deadline()
		// timeout at 95% of deadline to avoid test panic
		timeout       = time.Duration(int64(95) * int64(time.Until(d)) / int64(100))
		ctxwt, cancel = context.WithTimeout(context.Background(), timeout)
		hb, isTicking = heartbeat.Beatn(wantCount)
		gotCount      = 0
		stopped       = make(chan struct{})
		ris, err      = NewRandIntStreamf(hb)
	)

	t.Cleanup(func() {
		cancel()
	})

	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	// read while ticking, signal when stopped
	go func() {
		defer close(stopped)
		for range ris.worker(isTicking) {
			gotCount++
		}
	}()

	// loop until counts match or timeout
	for gotCount != wantCount {
		select {
		case <-ctxwt.Done():
			t.Fatalf("test timedout: %s", ctxwt.Err())
		default:
		}
	}

	// require that the beating is stopped
	_, open := <-stopped
	if open {
		t.Fatalf("expected worker to be stopped")
	}
}
