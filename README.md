# Dependency Injection in Go: A Brief Introduction

_tl;dr_ We can apply DI to decouple our go components. I used a combination of three complimentary DI techniques to enable straightforward unit testing of a struct that executes some arbitrary work at a given time interval. The unit test is explicitly decoupled from any internal timing mechanisms. See the final unit test [here](https://github.com/MStreet3/depinject/blob/e8bca38b3c9dcdb9a3fb146f8f108d9ef27ddd46/worker/v6/worker_test.go#L1) and an example of a `main` method that uses the struct [here](https://github.com/MStreet3/depinject/blob/309471721702f90625d1010cc1d5129cd2c6d0a4/main.go#L1).

### Setup

```bash
$ git clone git@github.com:mstreet3/depinject && cd depinject
$ go run main.go # runs main function for 5 seconds
$ go test ./... # run tests on packages
```

## Overview

Software engineers strive to write code that succeeds in solving a technical challenge and ultimately
delivers business value. To this end it is also important that the code be easy to read and reckon
about, easy to maintain, robust and well tested. "Well tested" is a subjective definition, but if
we recognize that all code that we write will decay over time it is important to allow room for refactoring
and improvements. Ultimately, code that has its functional requirements well documented via tests
is easier to change because there is less risk of introducing a regression.

To that end it is important to have tools & techniques that allow us as software engineers to decouple
aspects of our systems into testable units where appropriate. Dependency Injection (DI) is a well
used technique for achieving a level of decoupling when writing software, in particular DI enables
an "inversion of control" in the sense that the clients of software components can define their
specific use cases by defining the required dependencies.

DI in Go has a variety of implementations and the purpose of this post and the accompanying repository
is to demonstrate the complimentary nature of three DI patterns and their practical application in a small
system. The patterns discussed in this post are:

1. [Constructor Injection](#constructor-injection)
2. [Just-In-Time Injection](#just-in-time-jit-injection)
3. [Config Injection](#config-injection)

## The Challenge

A common use case for a goroutine is to periodically do a task on a given interval. The standard
library exports the `After` function from the `time` package for this exact purpose:

```go
// worker calls doWork once each second and sends the result onto the values channel.
func worker(stop <-chan struct{}) <-chan int {
	// initialize the value chan and the work function
	var (
		values = make(chan int)
		doWork = func() int {
			return rand.Int()
		}
	)

	// write values each second
	go func() {
		defer close(values)
		for {
			select {
			case <-stop:
				fmt.Println("done working!")
				return
			case <-time.After(1 * time.Second):
				fmt.Println("doing work...")
				values <- doWork()
			}
		}
	}()

	return values
}
```

Now that we have the function written we want to verify its functionality, but what happens when we try to add unit tests?

We run into at least one issue: our tests are tightly coupled to the implementation of `worker` via the internal time delay. This reliance on a
specific time duration will make tests slow and possibly unpredictable.

Through the remainder of this post we will apply DI patterns to this code to:

1. decouple `worker` from explicit use of time
2. allow a scalable means for setting the pulse width (i.e., time between heartbeats or work calls) for the `worker`

## Constructor Injection

The first pattern we will apply is common throughout many Go packages, it is a pattern called _Constructor Injection_. This pattern requires that the `worker` function be changed into a method of a struct (public for now), where some internal state can be set through a constructor of the struct. The private state we will apply
here is a heartbeat channel, which will allow us to decouple the method from an explicit timing mechanism.

```go
type beat struct{}

type RandIntStream struct {
	hb <-chan beat
}

// NewRandIntStream is the public constructor that accepts a heartbeat channel that signals
// when to do work
func NewRandIntStream(hb <-chan beat) *RandIntStream {
	return &RandIntStream{
		hb: hb,
	}
}

// Start is the public means of accessing the stream of results from doing work
func (r *RandIntStream) Start(ctx context.Context) <-chan int {
	return r.worker(ctx.Done())
}

func (r *RandIntStream) worker(stop <-chan struct{}) <-chan int {
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
```

We successfully decoupled the `worker` method from an explicit time duration by defining a struct called `RandIntStream` and exposing a constructor that accepts a heartbeat
channel.

But at what cost?

There are a couple of immediate disadvantages to the approach of using _Constructor Injection_ alone particularly in this design where the time duration itself is _not_ a struct property:

1. users of `RandIntStream` are now responsible for creating and providing a heartbeat channel instead of just a time duration, and
2. we have introduced a subtle bug in that a `nil` channel blocks forever and we should therefore supply a reasonable default heartbeat channel.

## Just-In-Time (JIT) Injection

_Just-in-Time_ or JIT dependency injection is a complementary technique to _Constructor Injection_
that helps ensure useful default values are present in the methods of our objects. We can solve the
first challenge of the need to provide a heartbeat channel with a helper function from a separate `heartbeat` package like this:

```go
type Beat struct{}

// BeatUntil returns a channel that beats with a pulse width of d until stop is closed
func BeatUntil(stop <-chan struct{}, d time.Duration) <-chan Beat {
	hb := make(chan Beat)
	go func() {
		defer close(hb)
		for {
			select {
			case <-stop:
				return
			case <-time.After(d):
				hb <- Beat{}
			}
		}
	}()
	return hb
}
```

Next, we can use JIT to inject the default heartbeat channel as needed:

```go
// getHeartbeat assigns a default heartbeat for the stream if one has not already been provided
func (r *randIntStream) getHeartbeat(stop <-chan struct{}) <-chan heartbeat.Beat {
	if r.hb == nil {
		r.hb = heartbeat.BeatUntil(stop, 1*time.Second) // oh no, time interval is back!
	}

	return r.hb
}

// worker now calls getHeartbeat to return a heartbeat channel "just in time" to start
// doing work
func (r *randIntStream) worker(stop <-chan struct{}) <-chan int {
	var (
		values = make(chan int)
		hb     = r.getHeartbeat(stop) // "just in time" call to get heartbeat channel
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
			case <-hb:
				fmt.Println("doing work...")
				values <- doWork()
			}
		}
	}()

	return values
}
```

We can see that now we are defining our `heartbeat` channel "just in time" to use the actual channel. At this point we have resolved our need to
decouple the `worker` method from an internal time duration for testing and made a helper function to handle `heartbeat` generation.

However, because we provide a default `heartbeat` channel if `nil` is passed to the constructor, there is still a default internal time duration. We would like a scalable way to expose this internal value. Also, since we have made `randIntStream` private, we need to address the ergonomics of the constructor, as it is a bit strange of a coding UX to pass a `nil` channel to get default behavior:

```go
// passing nil to get a default heartbeat shows the strange UX of the current constructor
ris := NewRandIntStream(nil)
```

## Config Injection

_Config Injection_ is a DI technique that relies on a config interface to grant abstract access to internal
aspects of an object. Here is an example of how we can use _Config Injection_ to access the internal
time duration of `randIntStream`:

```go
// locally define a config interface to expose configuration values
type config interface {
	PulseWidth() time.Duration // time between heartbeats
}

// compose randIntStream with the config interface
type randIntStream struct {
	config
	hb <-chan heartbeat.Beat
}

// NewRandIntStream now accepts as cfg a struct that satisfies the config interface along with a heartbeat channel
func NewRandIntStream(cfg config, hb <-chan heartbeat.Beat) *randIntStream {
	return &randIntStream{
		config: cfg,
		hb:     hb,
	}
}

// getHeartbeat now gets the pulse width from the config interface
func (r *randIntStream) getHeartbeat(stop <-chan struct{}) <-chan heartbeat.Beat {
	if r.hb == nil {
		r.hb = heartbeat.BeatUntil(stop, r.PulseWidth())
	}

	return r.hb
}
```

Now we finally have achieved our two goals of decoupling `worker` from any actual timing variables
and also created a means to configure the internal time duration, however, the new constructor
is cumbersome to use as its not clear why a caller would need to set a `Config` _and_ a `heartbeat`.

The answer is: _they don't_! They must set at least one, which isn't validated and adds to the overall confusion.

An alternative design of the constructor is to split it in two. Each constructor will now fail if the caller uses a `nil` value:

```go
// NewRandIntStreamf returns a rand int stream formatted via the provided heartbeat channel
func NewRandIntStreamf(hb <-chan heartbeat.Beat) (*randIntStream, error) {
	if hb == nil {
		return nil, errors.New("cannot provide a nil channel to constructor")
	}
	return &randIntStream{
		hb: hb,
	}, nil
}

func NewRandIntStream(cfg config) (*randIntStream, error) {
	if cfg == nil {
		return nil, errors.New("cannot provide a nil config to constructor")
	}
	return &randIntStream{
		config: cfg,
	}, nil
}
```

The final piece of our _Config Injection_ technique is to create a `config` package that exports some real configuration values that will be a dependency of the layer that actually uses the `randIntStream`. An example of the `config` package is shown here and it uses [functional options](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis) to allow increased extensibility:

```go
type config struct {
	d time.Duration
}

func (c *config) PulseWidth() time.Duration {
	return c.d
}

// newConfig creates a new instance of the default config
func newConfig() *config {
	return &config{
		d: 250 * time.Millisecond,
	}
}

// NewConfig is the public constructor of the config package that allows other callers to inject config as needed
func NewConfig(opts ...func(*config)) *config {
	cfg := newConfig()
	for _, fn := range opts {
		fn(cfg)
	}
	return &config{
		d: cfg.d,
	}
}

// WithPulsWidth is the public function for setting a config's pulse width
func WithPulseWidth(d time.Duration) func(*config) {
	return func(cfg *config) {
		cfg.d = d
	}
}
```

## Unit Testing

Now that our `worker` has been fully decoupled using our various DI approaches, we can easily verify via unit tests that at least two conditions are met:

1. a specific number of results are written by `worker`
2. `worker` is cleaned up when it reads from `stop`

A unit test that achieves these assertions using the `NewRandIntStreamf` constructor is below:

```go
func TestBeatingRandIntStream(t *testing.T) {
	var (
		wantCount = 100
		gotCount  = 0
		d, _      = t.Deadline()
		// timeout at 95% of deadline to avoid test panic
		timeout       = time.Duration(int64(95) * int64(time.Until(d)) / int64(100))
		ctxwt, cancel = context.WithTimeout(context.Background(), timeout)
		hb, isBeating = heartbeat.Beatn(wantCount)
		ris, err      = NewRandIntStreamf(hb)
	)

	t.Cleanup(func() {
		cancel()
	})

	if err != nil {
		t.Fatalf("unexpected constructor error: %s", err.Error())
	}

	// go count values read while heart is beating
	go func() {
		for range ris.Start(ctxwt) {
			gotCount++
		}
	}()

	// loop until counts match or timeout (condition 1)
	for gotCount != wantCount {
		select {
		case <-ctxwt.Done():
			t.Fatalf("unexpected timeout: %s", ctxwt.Err())
		default:
		}
	}

	// require that the beating is stopped (condition 2)
	_, open := <-isBeating
	if open {
		t.Fatalf("expected heartbeat to be stopped")
	}
}
```

The nice thing about the `NewRandIntStreamf` constructor is that it makes testing simple, but it is public
because its use can go beyond just testing. Since this constructor exposes a heartbeat channel directly,
it is possible to synchronize the work of the `randIntStream` with another process. In such a
scenario, the heartbeat of one process can become the forcing function of our `worker` method.

## Injecting meaning

Go provides a rich set of tools for developing decoupled, extensible software and the DI patterns reviewed
in this post simply scratch the surface of what is possible. The last bit of code I will end this post with is an example of how our `worker` could be consumed in a simple `main` function:

```go
func main() {
	var (
		ctxwt, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		cfg           = config.NewConfig()
		ris, err      = worker.NewRandIntStream(cfg)
		values        = ris.Start(ctxwt)
	)

	defer cancel()

	if err != nil {
		log.Fatal(err)
	}

	for val := range values {
		fmt.Println(val)
	}
}
```

In this final example we can see how config injection has made creating a new worker extremely simple.
With that we'll conclude this brief introduction to three key DI patterns in Go. Several books that inspired these musings on DI are:

1. [Hands on Dependency Injection in Go](https://www.packtpub.com/product/hands-on-dependency-injection-in-go/9781789132762)
2. [Concurrency in Go](https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/)
