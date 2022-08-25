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
	PulseInterval() time.Duration
}

type randIntStream struct {
	config
	ticker <-chan heartbeat.Beat
}

func NewRandIntStream(cfg config, opts ...func(*option)) (*randIntStream, error) {
	o := newOption()

	// apply each function to the option
	for _, fn := range opts {
		fn(o)
	}

	ris := &randIntStream{
		config: cfg,
		ticker: o.ticker,
	}

	if ris.config == nil && ris.ticker == nil {
		return nil, errors.New("must provide either a config or a ticker")
	}

	return ris, nil
}

func (r *randIntStream) Start(ctx context.Context) <-chan int {
	return r.worker(ctx.Done())
}

func (r *randIntStream) worker(stop <-chan struct{}) <-chan int {
	values := make(chan int)
	go func() {
		defer fmt.Println("done working!")
		defer close(values)
		ticker := r.getTicker(stop)
		for {
			select {
			case <-stop:
				return
			case <-ticker:
				fmt.Println("doing work...")
				values <- rand.Int()
			}
		}
	}()
	return values
}

func (r *randIntStream) getTicker(stop <-chan struct{}) <-chan heartbeat.Beat {
	if r.ticker == nil {
		return heartbeat.BeatUntil(stop, r.getDuration())
	}

	return r.ticker
}

func (r *randIntStream) getDuration() time.Duration {
	return r.PulseInterval()
}
