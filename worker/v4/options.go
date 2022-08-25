package v4

import (
	"github.com/mstreet3/depinject/heartbeat"
)

// option is a private struct of the package that holds the default ticker value
type option struct {
	hb <-chan heartbeat.Beat
}

// by default the ticker value is nil, which will get replaced via JIT DI
func newOption() *option {
	return &option{}
}

// WithHeartbeat is a public function for accessing option to set the ticker value
func WithHeartbeat(hb <-chan heartbeat.Beat) func(*option) {
	return func(o *option) {
		o.hb = hb
	}
}
