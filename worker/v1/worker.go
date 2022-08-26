package v1

import (
	"fmt"
	"math/rand"
	"time"
)

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
