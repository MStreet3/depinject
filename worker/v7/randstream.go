package v7

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mstreet3/depinject/heartbeat"
)

type config interface {
	PulseWidth() time.Duration
}

type resultStream[T int] struct {
	config
	hb <-chan heartbeat.Beat
	worker[T]
}

// NewResultStreamf returns an int stream formatted via the provided heartbeat channel
func NewResultStreamf(hb <-chan heartbeat.Beat) (*resultStream[int], error) {
	if hb == nil {
		return nil, errors.New("cannot provide a nil channel to constructor")
	}
	return &resultStream[int]{
		hb:     hb,
		worker: randInt{},
	}, nil
}

func NewResultStream(cfg config) (*resultStream[int], error) {
	if cfg == nil {
		return nil, errors.New("cannot provide a nil config to constructor")
	}
	return &resultStream[int]{
		config: cfg,
		worker: randInt{},
	}, nil
}

// Start returns the stream of random integers, the stream has been shutdown if the returned channel is closed
func (r *resultStream[int]) Start(ctx context.Context) <-chan int {
	return r.serve(ctx.Done())
}

// worker calls does some arbitrary work with each heartbeat until it is told to stop
func (r *resultStream[int]) serve(stop <-chan struct{}) <-chan int {
	var (
		values    = make(chan int)
		heartbeat = r.getHeartbeat(stop)
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
				val, _ := r.work()
				select {
				case values <- val:
				case <-stop:
					return
				}
			}
		}
	}()

	return values
}

// getHeartbeat assigns a default heartbeat for the stream if one has not already been provided
func (r *resultStream[int]) getHeartbeat(stop <-chan struct{}) <-chan heartbeat.Beat {
	if r.hb == nil {
		r.hb = heartbeat.BeatUntil(stop, r.PulseWidth())
	}

	return r.hb
}
