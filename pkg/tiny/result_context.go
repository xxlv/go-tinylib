package tiny

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ThenWithContext applies a function to the value of a successful Result, respecting the provided context.
// If the context is canceled or times out before or during the function execution, it returns a Failure Result with the context error.
// If the Result is in the Failure state, it returns itself unchanged.
// Otherwise, it applies fn to the value and returns the new Result.
//
// Example:
//
//	r := Ok[string, error]("hello")
//	result := ThenWithContext(context.Background(), r, func(s string) Result[string, error] {
//	    return Ok[string, error](s + " world")
//	})
func ThenWithContext[T any, E error](ctx context.Context, r Result[T, E], fn func(T) Result[T, E]) Result[T, E] {
	if r.state == Failure {
		return r
	}
	// Check if context is already canceled before proceeding.
	if err := ctx.Err(); err != nil {
		return Fail[T, E](any(err).(E))
	}
	return fn(r.value)
}

// MapWithContext transforms a Result's value using a function that may fail, respecting the provided context.
// If the context is canceled or times out before or during the function execution, it returns a Failure Result with the context error.
// If the Result is in the Failure state, it returns a new Failure Result with the original error.
// Otherwise, it applies fn to the value, returning a new Result with the transformed value or error.
//
// Example:
//
//	r := Ok[int, error](5)
//	result := MapWithContext(context.Background(), r, func(i int) (string, error) {
//	    return fmt.Sprintf("%d", i), nil
//	})
func MapWithContext[T, U any, E error](ctx context.Context, r Result[T, E], fn func(T) (U, E)) Result[U, E] {
	if r.state == Failure {
		return Fail[U, E](r.fault)
	}
	// Check if context is already canceled before proceeding.
	if err := ctx.Err(); err != nil {
		return Fail[U, E](any(err).(E))
	}
	val, err := fn(r.value)
	if any(err) != nil && !errors.Is(err, nil) {
		return Fail[U, E](err)
	}
	return Ok[U, E](val)
}

// AsyncThenWithContext applies a function to a successful Result asynchronously, respecting the provided context.
// It returns a channel that will receive the Result of applying fn to the value.
// If the context is canceled or times out, the channel receives a Failure Result with the context error.
// If the Result is in the Failure state, the channel receives the original Result immediately.
//
// Example:
//
//	r := Ok[string, error]("start")
//	ch := AsyncThenWithContext(context.Background(), r, func(s string) Result[string, error] {
//	    return Ok[string, error](s + " done")
//	})
//	result := <-ch
func AsyncThenWithContext[T any, E error](ctx context.Context, r Result[T, E], fn func(T) Result[T, E]) <-chan Result[T, E] {
	ch := make(chan Result[T, E], 1)
	go func() {
		defer close(ch)
		// Check context before proceeding.
		if err := ctx.Err(); err != nil {
			ch <- Fail[T, E](any(err).(E))
			return
		}
		// Use a select to handle context cancellation during execution.
		resultChan := make(chan Result[T, E], 1)
		go func() {
			resultChan <- r.Then(fn)
		}()
		select {
		case result := <-resultChan:
			ch <- result
		case <-ctx.Done():
			ch <- Fail[T, E](any(ctx.Err()).(E))
		}
	}()
	return ch
}

// AsyncThenWithContextAndTimeout applies a function to a successful Result asynchronously, respecting the provided context and an additional timeout.
// It returns a channel that will receive the Result of applying fn to the value.
// If the context is canceled, times out, or the operation exceeds the timeout, the channel receives a Failure Result with the appropriate error.
// If the Result is in the Failure state, the channel receives the original Result immediately.
//
// The timeout parameter acts as an additional constraint beyond the context's deadline, whichever comes first.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
//	defer cancel()
//	r := Ok[string, error]("task")
//	ch := AsyncThenWithContextAndTimeout(ctx, r, func(s string) Result[string, error] {
//	    time.Sleep(1 * time.Second)
//	    return Ok[string, error](s + " completed")
//	}, 500*time.Millisecond)
//	result := <-ch
func AsyncThenWithContextAndTimeout[T any, E error](ctx context.Context, r Result[T, E], fn func(T) Result[T, E], timeout time.Duration) <-chan Result[T, E] {
	ch := make(chan Result[T, E], 1)
	go func() {
		defer close(ch)
		// Check context before proceeding.
		if err := ctx.Err(); err != nil {
			ch <- Fail[T, E](any(err).(E))
			return
		}
		// Create a context with timeout if it's stricter than the provided context's deadline.
		ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resultChan := make(chan Result[T, E], 1)
		go func() {
			resultChan <- r.Then(fn)
		}()

		select {
		case result := <-resultChan:
			ch <- result
		case <-ctxWithTimeout.Done():
			err := ctxWithTimeout.Err()
			if errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("operation timed out after %v", timeout)
			}
			ch <- Fail[T, E](any(err).(E))
		}
	}()
	return ch
}
