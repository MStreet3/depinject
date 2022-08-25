package v4

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/mstreet3/depinject/entities"
)

type Config interface {
	PulseInterval() time.Duration
}

type RandIntStream struct {
	cfg    Config
	ticker <-chan entities.Beat
}

func NewRandIntStream(cfg Config, opts ...func(*option)) (*RandIntStream, error) {
	o := newOption()

	// apply each function to the option
	for _, fn := range opts {
		fn(o)
	}

	ris := &RandIntStream{
		cfg:    cfg,
		ticker: o.ticker,
	}

	if ris.cfg == nil && ris.ticker == nil {
		return nil, errors.New("must provide either a config or a ticker")
	}

	return ris, nil
}

func (r *RandIntStream) Start(ctx context.Context) <-chan int {
	return r.worker(ctx.Done())
}

func (r *RandIntStream) worker(stop <-chan struct{}) <-chan int {
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

func (r *RandIntStream) getTicker(stop <-chan struct{}) <-chan entities.Beat {
	if r.ticker == nil {
		return getHeartbeat(stop, r.getDuration())
	}

	return r.ticker
}

func (r *RandIntStream) getDuration() time.Duration {
	if r.cfg.PulseInterval() == 0 {
		return 100 * time.Millisecond
	}

	return r.cfg.PulseInterval()
}

func getHeartbeat(stop <-chan struct{}, d time.Duration) <-chan entities.Beat {
	hb := make(chan entities.Beat)
	go func() {
		defer close(hb)
		for {
			select {
			case <-stop:
				return
			case <-time.After(d):
				hb <- entities.Beat{}
			}
		}
	}()
	return hb
}

func tickNTimes(n int) (<-chan entities.Beat, <-chan struct{}) {
	hb := make(chan entities.Beat)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(hb)
		for i := 0; i < n; i++ {
			hb <- entities.Beat{}
		}
	}()
	return hb, done
}

func main() {
	stop := make(chan struct{})
	time.AfterFunc(3*time.Second, func() { close(stop) })
	ris := &RandIntStream{}
	for val := range ris.worker(stop) {
		fmt.Printf("%d\n", val)
	}
}
