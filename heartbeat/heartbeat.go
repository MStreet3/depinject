/*
Package heartbeat implements various heartbeat generators
*/
package heartbeat

import (
	"time"
)

type Beat struct{}

// BeatUntil returns a channel that beats with a pulse width of d until stop is closed
func BeatUntil(stop <-chan struct{}, d time.Duration) <-chan Beat {
	hb := make(chan Beat)
	go func() {
		defer close(hb)
		for {
			select {
			case <-stop:
				return
			case <-time.After(d):
				hb <- Beat{}
			}
		}
	}()
	return hb
}

// Beatn returns both a channel that beats n times and a channel that only signals when done
// beating.  Both channels are closed after n beats.
func Beatn(n int) (<-chan Beat, <-chan struct{}) {
	hb := make(chan Beat)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(hb)
		for i := 0; i < n; i++ {
			hb <- Beat{}
		}
	}()
	return hb, done
}
