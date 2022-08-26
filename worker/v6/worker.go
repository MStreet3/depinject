package v6

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

// NewRandIntStreamf returns a rand int stream formatted via the provided heartbeat channel
func NewRandIntStreamf(hb <-chan heartbeat.Beat) (*randIntStream, error) {
	if hb == nil {
		return nil, errors.New("cannot provide a nil channel to constructor")
	}
	return &randIntStream{
		hb: hb,
	}, nil
}

func NewRandIntStream(cfg config) (*randIntStream, error) {
	if cfg == nil {
		return nil, errors.New("cannot provide a nil config to constructor")
	}
	return &randIntStream{
		config: cfg,
	}, nil
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
