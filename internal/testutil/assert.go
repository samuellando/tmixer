package testutil

import (
	"fmt"
)

type ErrorNotEqual[T comparable] struct {
	Actual   T
	Expected T
	Message  string
}

func (err ErrorNotEqual[T]) Error() string {
	if len(err.Message) == 0 {
		return fmt.Sprintf("%v != %v", err.Actual, err.Expected)
	} else {
		return fmt.Sprintf("%s: %v != %v", err.Message, err.Actual, err.Expected)
	}
}

type ErrorEqual[T comparable] struct {
	Actual   T
	Expected T
	Message  string
}

func (err ErrorEqual[T]) Error() string {
	if len(err.Message) == 0 {
		return fmt.Sprintf("%v == %v", err.Actual, err.Expected)
	} else {
		return fmt.Sprintf("%s: %v == %v", err.Message, err.Actual, err.Expected)
	}
}
