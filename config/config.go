package config

import (
	"time"
)

type config struct {
	d time.Duration
}

// newConfig creates a new instance of the default config
func newConfig() *config {
	return &config{
		d: 250 * time.Millisecond,
	}
}

func WithPulseWidth(d time.Duration) func(*config) {
	return func(cfg *config) {
		cfg.d = d
	}
}

func NewConfig(opts ...func(*config)) *config {
	cfg := newConfig()
	for _, fn := range opts {
		fn(cfg)
	}
	return &config{
		d: cfg.d,
	}
}

func (c *config) PulseWidth() time.Duration {
	return c.d
}
