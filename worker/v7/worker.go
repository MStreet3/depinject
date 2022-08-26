package v7

import "math/rand"

type worker[T any] interface {
	work() (T, error)
}

type randInt struct{}

func (r randInt) work() (int, error) {
	return rand.Int(), nil
}

type ones struct{}

func (o ones) work() (int, error) {
	return 1, nil
}
