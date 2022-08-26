package v5

import (
	"github.com/mstreet3/depinject/heartbeat"
)

// option is a private struct of the package that holds the default heartbeat channel
type option struct {
	hb <-chan heartbeat.Beat
}

// by default the heartbeat channel is nil, which will get replaced via JIT DI
func newOption() *option {
	return &option{}
}

// WithHeartbeat is a public function for accessing an option via its pointer to set the heartbeat channel value
func WithHeartbeat(hb <-chan heartbeat.Beat) func(*option) {
	return func(o *option) {
		o.hb = hb
	}
}
