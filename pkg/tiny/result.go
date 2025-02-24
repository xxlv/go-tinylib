package tiny

import (
	"errors"
	"fmt"
	"time"
)

// State represents the status of a Result, either Success or Failure.
type State int

const (
	// Success indicates a successful operation.
	Success State = iota
	// Failure indicates a failed operation.
	Failure
)

// Result encapsulates the outcome of an operation, holding either a value or an error.
// T is the type of the successful value, and E is the type of the error.
type Result[T any, E error] struct {
	state State // The state of the result (Success or Failure).
	value T     // The value in case of success.
	fault E     // The error in case of failure.
}

// Ok creates a Result with a successful value.
// It returns a Result in the Success state with the provided value.
func Ok[T any, E error](value T) Result[T, E] {
	return Result[T, E]{state: Success, value: value}
}

// Fail creates a Result with an error.
// It returns a Result in the Failure state with the provided error.
func Fail[T any, E error](err E) Result[T, E] {
	return Result[T, E]{state: Failure, fault: err}
}

// Then applies a function to the value of a successful Result.
// If the Result is in the Failure state, it returns itself unchanged.
// Otherwise, it applies fn to the value and returns the new Result.
func (r Result[T, E]) Then(fn func(T) Result[T, E]) Result[T, E] {
	if r.state == Failure {
		return r
	}
	return fn(r.value)
}

// Map transforms a Result's value using a function that may fail.
// If the Result is in the Failure state, it returns a new Failure Result with the original error.
// Otherwise, it applies fn to the value, returning a new Result with the transformed value or error.
func Map[T, U any, E error](r Result[T, E], fn func(T) (U, E)) Result[U, E] {
	if r.state == Failure {
		return Fail[U, E](r.fault)
	}
	val, err := fn(r.value)
	// Use type assertion to check if err is nil.
	if any(err) != nil && !errors.Is(err, nil) {
		return Fail[U, E](err)
	}
	return Ok[U, E](val)
}

// OrElse returns the value of a successful Result or a default value if it failed.
// If the Result is in the Success state, it returns the encapsulated value.
// Otherwise, it returns the provided defaultVal.
func (r Result[T, E]) OrElse(defaultVal T) T {
	if r.state == Success {
		return r.value
	}
	return defaultVal
}

// Wrap wraps the error of a failed Result with additional context.
// If the Result is in the Failure state, it returns a new Result with the error wrapped in a formatted message.
// If the Result is in the Success state, it returns a new Result with the original value and an error type.
func (r Result[T, E]) Wrap(msg string) Result[T, error] {
	if r.state == Failure {
		// Convert E to error interface for wrapping.
		wrappedErr := fmt.Errorf("%s: %w", msg, r.fault)
		return Fail[T, error](wrappedErr)
	}
	return Ok[T, error](r.value)
}

// Unwrap returns the error of a failed Result or a zero value if successful.
// If the Result is in the Failure state, it returns the encapsulated error.
// Otherwise, it returns the zero value of type E.
func (r Result[T, E]) Unwrap() E {
	if r.state == Failure {
		return r.fault
	}
	// Return zero value of type E when successful.
	var zero E
	return zero
}

// All combines multiple Results into a single Result containing a slice of values.
// If all Results are in the Success state, it returns a Result with a slice of their values.
// If any Result is in the Failure state, it returns a Failure Result with that error.
func All[T any, E error](results ...Result[T, E]) Result[[]T, E] {
	values := make([]T, 0, len(results))
	for _, r := range results {
		if r.state == Failure {
			return Fail[[]T, E](r.fault)
		}
		values = append(values, r.value)
	}
	return Ok[[]T, E](values)
}

// UnwrapOrPanic returns the value of a successful Result or panics if it failed.
// If the Result is in the Success state, it returns the encapsulated value.
// If the Result is in the Failure state, it panics with a message containing the error.
func (r Result[T, E]) UnwrapOrPanic() T {
	if r.state == Failure {
		panic(fmt.Sprintf("called UnwrapOrPanic on a Failure: %v", r.fault))
	}
	return r.value
}

// MapErr transforms the error of a failed Result using a function.
// If the Result is in the Failure state, it applies fn to the error and returns a new Result.
// If the Result is in the Success state, it returns a new Result with the original value and transformed error type.
func MapErr[T any, E, F error](r Result[T, E], fn func(E) F) Result[T, F] {
	if r.state == Failure {
		return Fail[T, F](fn(r.fault))
	}
	return Ok[T, F](r.value)
}

// AsyncThen applies a function to a successful Result asynchronously.
// It returns a channel that will receive the Result of applying fn to the value.
// If the Result is in the Failure state, the channel receives the original Result.
func AsyncThen[T any, E error](r Result[T, E], fn func(T) Result[T, E]) <-chan Result[T, E] {
	ch := make(chan Result[T, E], 1)
	go func() {
		defer close(ch)
		ch <- r.Then(fn)
	}()
	return ch
}

// String returns a string representation of the Result.
// For a Success state, it returns "Ok(value)".
// For a Failure state, it returns "Err(fault)".
func (r Result[T, E]) String() string {
	if r.state == Success {
		return fmt.Sprintf("Ok(%v)", r.value)
	}
	return fmt.Sprintf("Err(%v)", r.fault)
}

// AsyncThenWithTimeout applies a function to a successful Result asynchronously with a timeout.
// It returns a channel that will receive the Result of applying fn to the value or a timeout error.
// If the Result is in the Failure state, the channel receives the original Result immediately.
// If the operation exceeds the timeout, it returns a Failure Result with a timeout error.
func AsyncThenWithTimeout[T any, E error](r Result[T, E], fn func(T) Result[T, E], timeout time.Duration) <-chan Result[T, E] {
	ch := make(chan Result[T, E], 1)
	go func() {
		defer close(ch)

		resultChan := make(chan Result[T, E], 1)
		go func() {
			resultChan <- r.Then(fn)
		}()

		select {
		case result := <-resultChan:
			ch <- result
		case <-time.After(timeout):
			err := fmt.Errorf("operation timed out after %v", timeout)
			ch <- Fail[T, E](any(err).(E))
		}
	}()
	return ch
}
