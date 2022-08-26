package v2

import (
	"context"
	"fmt"
	"math/rand"
)

type beat struct{}

type randIntStream struct {
	hb <-chan beat
}

// NewRandIntStream is the public constructor that accepts a heartbeat channel that signals
// when to do work
func NewRandIntStream(hb <-chan beat) *randIntStream {
	return &randIntStream{
		hb: hb,
	}
}

// Start is the public means of accessing the stream of results from doing work
func (r *randIntStream) Start(ctx context.Context) <-chan int {
	return r.worker(ctx.Done())
}

func (r *randIntStream) worker(stop <-chan struct{}) <-chan int {
	var (
		values = make(chan int)
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
			// worker now uses the ticker injected via the constructor
			case <-r.hb:
				fmt.Println("doing work...")
				values <- doWork()
			}
		}
	}()

	return values
}
