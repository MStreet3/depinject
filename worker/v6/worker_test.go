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
		gotCount  = 0
		d, _      = t.Deadline()
		// timeout at 95% of deadline to avoid test panic
		timeout       = time.Duration(int64(95) * int64(time.Until(d)) / int64(100))
		ctxwt, cancel = context.WithTimeout(context.Background(), timeout)
		hb, isBeating = heartbeat.Beatn(wantCount)
		ris, err      = NewRandIntStreamf(hb)
	)

	t.Cleanup(func() {
		cancel()
	})

	if err != nil {
		t.Fatalf("unexpected constructor error: %s", err.Error())
	}

	// go count values read while heart is beating
	go func() {
		for range ris.Start(ctxwt) {
			gotCount++
		}
	}()

	// loop until counts match or timeout (condition 1)
	for gotCount != wantCount {
		select {
		case <-ctxwt.Done():
			t.Fatalf("unexpected timeout: %s", ctxwt.Err())
		default:
		}
	}

	// require that the beating is stopped (condition 2)
	_, open := <-isBeating
	if open {
		t.Fatalf("expected heartbeat to be stopped")
	}
}
