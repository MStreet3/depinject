package v3

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/mstreet3/depinject/heartbeat"
)

type randIntStream struct {
	hb <-chan heartbeat.Beat
}

// NewRandIntStream is the public constructor that accepts a heartbeat channel that signals
// when to do work
func NewRandIntStream(hb <-chan heartbeat.Beat) *randIntStream {
	return &randIntStream{
		hb: hb,
	}
}

// Start is the public means of accessing the stream of results from doing work
func (r *randIntStream) Start(ctx context.Context) <-chan int {
	return r.worker(ctx.Done())
}

// worker now calls getHeartbeat to return a heartbeat channel "just in time" to start
// doing work
func (r *randIntStream) worker(stop <-chan struct{}) <-chan int {
	var (
		values = make(chan int)
		hb     = r.getHeartbeat(stop) // "just in time" call to get heartbeat channel
		doWork = func() int {
			return rand.Int()
		}
	)

	go func() {
		defer close(values)
		for {
			select {
			case <-stop:
				fmt.Println("done working!")
				return
			case <-hb:
				fmt.Println("doing work...")
				values <- doWork()
			}
		}
	}()

	return values
}

// getHeartbeat assigns a default heartbeat for the stream if one has not already been provided
func (r *randIntStream) getHeartbeat(stop <-chan struct{}) <-chan heartbeat.Beat {
	if r.hb == nil {
		r.hb = heartbeat.BeatUntil(stop, 1*time.Second)
	}

	return r.hb
}
