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
	pulse <-chan heartbeat.Beat
}

func NewRandIntStream(cfg config, opts ...func(*option)) (*randIntStream, error) {
	o := newOption()

	// apply each function to the option
	for _, fn := range opts {
		fn(o)
	}

	ris := &randIntStream{
		config: cfg,
		pulse:  o.ticker,
	}

	if ris.config == nil && ris.pulse == nil {
		return nil, errors.New("must provide either a config or a ticker")
	}

	return ris, nil
}

func (r *randIntStream) Start(ctx context.Context) <-chan int {
	return r.worker(ctx.Done())
}

// worker calls does some arbitrary work each time a pulse is received until it is told to stop
func (r *randIntStream) worker(stop <-chan struct{}) <-chan int {
	var (
		values = make(chan int)
		pulse  = r.getHeartbeat(stop)
		doWork = func() int {
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
			case <-pulse:
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

func (r *randIntStream) getHeartbeat(stop <-chan struct{}) <-chan heartbeat.Beat {
	if r.pulse == nil {
		return heartbeat.BeatUntil(stop, r.PulseWidth())
	}

	return r.pulse
}
