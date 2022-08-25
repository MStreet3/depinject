package config

import (
	"time"
)

type config struct {
	d time.Duration
}

func newDefaultConfig() *config {
	return &config{
		d: 250 * time.Millisecond,
	}
}

func WithPulseInterval(d time.Duration) func(*config) {
	return func(cfg *config) {
		cfg.d = d
	}
}

func NewConfig(opts ...func(*config)) *config {
	cfg := newDefaultConfig()
	for _, fn := range opts {
		fn(cfg)
	}
	return &config{
		d: cfg.d,
	}
}

func (c *config) PulseInterval() time.Duration {
	return c.d
}
