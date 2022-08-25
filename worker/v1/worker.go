package v1

import (
	"fmt"
	"math/rand"
	"time"
)

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
