# Dependency Injection in Go

A common use case for a goroutine is to periodically do a task on a given interval. The standard
library exports the `After` function from the `time` package for this exact purpose:

```go
func worker(stop <-chan struct{}) <-chan int {
	values := make(chan int)
	go func() {
		defer close(values)
		for {
			select {
			case <-stop:
				fmt.Println("done working!")
				return
			case <-time.After(1 * time.Second):
				fmt.Println("doing work...")
				values <- rand.Int()
			}
		}
	}()
	return values
}
```

Now that we have the function written we want to verify its functionality, but what happens when we try to add unit tests?

We run into at least one issue: our tests are tightly coupled to the implementation of `worker` via the internal time delay. This reliance on a
specific time duration will make tests slow and possibly unpredictable. Our goal is to use dependency injection (DI) to:

1. decouple `worker` from explicit use of time
2. allow a scalable means for setting the pulse width (i.e., time between heartbeats or work calls) for the `worker`

## Constructor Injection

A common way to break the tight coupling to the internal time setting is to introduce a heartbeat channel
using a technique known as _Constructor Injection_. We will first make the `worker` function a method
of a `RandIntStream` struct and allow the ticker to be injected at the time of instantiation.

```go
type beat struct{}

type RandIntStream struct {
    ticker <-chan beat
}

// inject the ticker via the constructor
func NewRandIntStream(ticker <-chan beat) *RandIntStream {
	return &RandIntStream{
		ticker: ticker,
	}
}

func (r *RandIntStream) worker(stop <-chan struct{}) <-chan int {
    values := make(chan int)
	go func() {
		defer close(values)
		for {
			select {
			case <-stop:
				fmt.Println("done working!")
				return
			case <-r.ticker: // worker now uses the ticker injected via the constructor
				fmt.Println("doing work...")
				values <- rand.Int()
			}
		}
	}()
	return values
}
```

We have now broken the coupling at the expense of making the future owners or testers of the `RandIntStream`
responsible for more setup work (i.e., creating and providing a heartbeat channel instead of just a time duration).
Another issue we have introduced is that a `nil` channel blocks forever and we should therefore supply
a reasonable default value for the ticker.

## Just-In-Time (JIT) Injection

_Just-in-Time_ or JIT dependency injection is a complementary technique to _Constructor Injection_
that helps ensure useful default values are present in the methods of our objects. We can solve the
first challenge of the need to provide a heartbeat channel with a helper function like this:

```go
// getHeartbeat converts a stop condition and a time duration into a heartbeat channel
func getHeartbeat(stop <-chan struct{}, d time.Duration) <-chan beat {
	hb := make(chan beat)
	go func() {
		defer close(hb)
		for {
			select {
			case <-stop:
				return
			case <-time.After(d):
				hb <- beat{}
			}
		}
	}()
	return hb
}
```

Next, we can use JIT to inject the default heartbeat channel as needed:

```go
// getTicker defines the ticker channel just-in-time to be used
func (r *RandIntStream) getTicker(stop <-chan struct{}) <-chan beat {
	if r.ticker == nil {
		return getHeartbeat(stop, 1 * time.Second)
	}

	return r.ticker
}

func (r *RandIntStream) worker(stop <-chan struct{}) <-chan int {
	values := make(chan int)
	go func() {
		defer fmt.Println("done working!")
		defer close(values)
		// get the ticker to use before starting our worker loop
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
```

We can see that now we are defining our `ticker` channel "just in time" to use the actual channel.  
A unit test would simply have to inject its own channel and at this point we have resolved our need to
decouple the `worker` method from an internal time duration. However, there is still a default internal
time duration and we would like a scalable way to expose this internal value.

## Config Injection

_Config Injection_ is a DI technique that relies on a config interface to grant abstract access to internal
aspects of an object. Here is an example of how we can use _Config Injection_ to access the internal
time duration of `RandIntStream`:

```go
// locally define a config interface that provides needed configuration values
type Config interface {
	PulseWidth() time.Duration // time between hearbeats
}

// compose RandIntStream with the local interface
type RandIntStream struct {
	Config
	ticker <-chan beat
}

// update the RandIntStream constructor to inject a config and ticker
func NewRandIntStream(cfg Config, ticker <-chan beat) *RandIntStream {
	return &RandIntStream{
		Config: cfg,
		ticker: ticker,
	}
}

// getTicker defines the ticker channel just-in-time to be used and fallsback to
// a heartbeat channel with a config defined pulse width
func (r *RandIntStream) getTicker(stop <-chan struct{}) <-chan beat {
	if r.ticker == nil {
		return getHeartbeat(stop, r.PulseWidth())
	}

	return r.ticker
}
```

Now we finally have achieved our two goals of decoupling `worker` from any actual timing variables
and also created a means to configure the pulse width, however, as is often the case with _Constructor Injection_
we have begun to "leak" the implementation details of our `RandIntStream`. Also, the new constructor
is cumbersome to use as its not clear why a caller would need to set a `Config` _and_ a `ticker`.

## Functional Options

_Functional Options_ is a method of specifying optional arguments to inject into an object. In brief,
the technique relies on applying a chain of functions with locally scoped config values to a shared
option state. Here is what it looks like in practice:

```go
// option is a private struct of the package that holds the default ticker value
type option struct {
	ticker <-chan beat
}

// by default the ticker value is nil, which will get replaced via JIT DI
func newOption() *option {
	return &option{}
}

// WithTicker is a public function for accessing option to set the ticker value
func WithTicker(ticker <-chan beat) func(*option) {
	return func(o *option){
		o.ticker = ticker
	}
}

// modify the constructor to use a variadic slice of opts to set up the options
func NewRandIntStream(cfg Config, opts ...func(*option)) *RandIntStream {
	o := newOption()

	// apply each function to the option
	for _, fn := range opts {
		fn(o)
	}

	return &RandIntStream{
		Config: cfg,
		ticker: o.ticker,
	}
}
```

We can apply the same technique to the config.

```go
type config struct {
	d      time.Duration
}

func newDefaultConfig() *config {
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
	cfg := newDefaultConfig()
	for _, fn := range opts {
		fn(cfg)
	}
	return &config{
		d:      cfg.d,
	}
}

func (c *config) PulseWidth() time.Duration {
	return c.d
}
```

## Unit Testing

Now that our `worker` has been fully decoupled using our various DI approaches, we can easily verify via unit tests that at least
two conditions are met:

1. a specific number of results are written by `worker`
2. `worker` is cleaned up when it reads from `stop`

One unit test that achieves these assertions is below:

```go
func TestBeatingRandIntStream(t *testing.T) {
	var (
		wantCount     = 100
		gotCount      = 0
		d, _          = t.Deadline() // get the test deadline if it exists
		timeout       = time.Duration(int64(95) * int64(time.Until(d)) / int64(100)) // timeout at 95% of deadline to avoid test panic
		ctxwt, cancel = context.WithTimeout(context.Background(), timeout)
		hb, isTicking = tickNTimes(wantCount)
		stopped       = make(chan struct{})
		ris           = NewRandIntStream(NewConfig(), WithTicker(hb))
	)

	t.Cleanup(func() {
		cancel()
	})

	// read while ticking, signal when stopped
	go func() {
		defer close(stopped)
		for range ris.worker(isTicking) {
			gotCount++
		}
	}()

	// this loop will run until we either timeout or wantCount and gotCount are equal (condition 1)
	for gotCount != wantCount {
		select {
		case <-ctxwt.Done():
			t.Logf("test timedout: %s", ctxwt.Err())
			t.FailNow()
		default:
		}
	}

	// expect the worker to be cleaned up when the heartbeat stops (condition 2)
	_, open := <-stopped
	if open {
		t.Errorf("expected worker to be stopped")
	}
}
```
