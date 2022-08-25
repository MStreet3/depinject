package v4

import (
	"github.com/mstreet3/depinject/entities"
)

// option is a private struct of the package that holds the default ticker value
type option struct {
	ticker <-chan entities.Beat
}

// by default the ticker value is nil, which will get replaced via JIT DI
func newOption() *option {
	return &option{}
}

// WithTicker is a public function for accessing option to set the ticker value
func WithTicker(ticker <-chan entities.Beat) func(*option) {
	return func(o *option) {
		o.ticker = ticker
	}
}
