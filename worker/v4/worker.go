/*
Package worker/v4 implements a stream of random integers whos frequency can be configured by providing
a heartbeat directly or a pulse width.
*/
package v4

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/mstreet3/depinject/heartbeat"
)

type config interface {
	PulseWidth() time.Duration
}

type randIntStream struct {
	config
	hb <-chan heartbeat.Beat
}

func NewRandIntStream(cfg config, opts ...func(*option)) (*randIntStream, error) {
	// load default options
	o := newOption()

	// apply each function to the option
	for _, fn := range opts {
		fn(o)
	}

	ris := &randIntStream{
		config: cfg,
		hb:     o.hb,
	}

	if ris.config == nil && ris.hb == nil {
		return nil, errors.New("must provide either a config or a ticker")
	}

	return ris, nil
}

// Start returns the stream of random integers, the stream has been shutdown if the returned channel is closed
func (r *randIntStream) Start(ctx context.Context) <-chan int {
	return r.worker(ctx.Done())
}

// worker calls does some arbitrary work with each heartbeat until it is told to stop
func (r *randIntStream) worker(stop <-chan struct{}) <-chan int {
	var (
		values    = make(chan int)
		heartbeat = r.getHeartbeat(stop)
		doWork    = func() int {
			return rand.Int()
		}
	)

	go func() {
		defer fmt.Println("done working!")
		defer close(values)
		for {
			select {
			case <-stop:
				return
			case <-heartbeat:
				fmt.Println("doing work...")
				select {
				case values <- doWork():
				case <-stop:
					return
				}
			}
		}
	}()

	return values
}

// getHeartbeat assigns a default heartbeat for the stream if one has not already been provided
func (r *randIntStream) getHeartbeat(stop <-chan struct{}) <-chan heartbeat.Beat {
	if r.hb == nil {
		r.hb = heartbeat.BeatUntil(stop, r.PulseWidth())
	}

	return r.hb
}
