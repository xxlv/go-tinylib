package tiny

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func ExampleOk() {
	r := Ok[int, error](42)
	fmt.Println(r.UnwrapOrPanic())
	// Output: 42
}

func ExampleFail() {
	err := errors.New("something went wrong")
	r := Fail[int, error](err)
	fmt.Println(r.String())
	// Output: Err(something went wrong)
}

func ExampleThen() {
	r := Ok[int, error](5)
	result := r.Then(func(x int) Result[int, error] {
		return Ok[int, error](x * 2)
	})
	fmt.Println(result.UnwrapOrPanic())
	// Output: 10
}

func ExampleMap() {
	r := Ok[int, error](5)
	result := Map(r, func(x int) (string, error) {
		return fmt.Sprintf("value: %d", x), nil
	})
	fmt.Println(result.UnwrapOrPanic())
	// Output: value: 5
}

func ExampleOrElse() {
	r1 := Ok[int, error](42)
	fmt.Println(r1.OrElse(0))

	err := errors.New("test error")
	r2 := Fail[int, error](err)
	fmt.Println(r2.OrElse(0))
	// Output:
	// 42
	// 0
}

func ExampleWrap() {
	err := errors.New("original error")
	r := Fail[int, error](err)
	wrapped := r.Wrap("context")
	fmt.Println(wrapped.Unwrap().Error())
	// Output: context: original error
}

func ExampleAll() {
	r1 := Ok[int, error](1)
	r2 := Ok[int, error](2)
	r3 := Ok[int, error](3)
	result := All(r1, r2, r3)
	fmt.Println(result.UnwrapOrPanic())
	// Output: [1 2 3]
}

func TestResultBasic(t *testing.T) {
	okResult := Ok[int, error](42)
	if okResult.state != Success {
		t.Errorf("Ok should set state to Success, got %v", okResult.state)
	}
	if okResult.value != 42 {
		t.Errorf("Ok should set value to 42, got %v", okResult.value)
	}

	err := errors.New("test error")
	failResult := Fail[int, error](err)
	if failResult.state != Failure {
		t.Errorf("Fail should set state to Failure, got %v", failResult.state)
	}
	if failResult.fault != err {
		t.Errorf("Fail should set fault to err, got %v", failResult.fault)
	}
}

func TestThen(t *testing.T) {
	r1 := Ok[int, error](5)
	result := r1.Then(func(x int) Result[int, error] {
		return Ok[int, error](x * 2)
	})
	if result.UnwrapOrPanic() != 10 {
		t.Errorf("Then should multiply by 2, got %v", result.value)
	}

	err := errors.New("test error")
	r2 := Fail[int, error](err)
	result2 := r2.Then(func(x int) Result[int, error] {
		return Ok[int, error](x * 2)
	})
	if result2.state != Failure {
		t.Errorf("Then on Failure should preserve Failure state")
	}
}

func TestMap(t *testing.T) {
	r1 := Ok[int, error](5)
	result := Map(r1, func(x int) (string, error) {
		return fmt.Sprintf("value: %d", x), nil
	})
	if result.UnwrapOrPanic() != "value: 5" {
		t.Errorf("Map should transform value, got %v", result.value)
	}

	err := errors.New("test error")
	r2 := Fail[int, error](err)
	result2 := Map(r2, func(x int) (string, error) {
		return fmt.Sprintf("value: %d", x), nil
	})
	if result2.state != Failure {
		t.Errorf("Map on Failure should preserve Failure state")
	}
}

func TestOrElse(t *testing.T) {
	r1 := Ok[int, error](42)
	if r1.OrElse(0) != 42 {
		t.Errorf("OrElse on Success should return value, got %v", r1.OrElse(0))
	}
	err := errors.New("test error")
	r2 := Fail[int, error](err)
	if r2.OrElse(0) != 0 {
		t.Errorf("OrElse on Failure should return default, got %v", r2.OrElse(0))
	}
}

func TestWrap(t *testing.T) {
	r1 := Ok[int, error](42)
	wrapped1 := r1.Wrap("context")
	if wrapped1.UnwrapOrPanic() != 42 {
		t.Errorf("Wrap on Success should preserve value")
	}

	err := errors.New("original error")
	r2 := Fail[int, error](err)
	wrapped2 := r2.Wrap("context")
	if wrapped2.state != Failure {
		t.Errorf("Wrap on Failure should preserve Failure state")
	}
	if wrapped2.Unwrap().Error() != "context: original error" {
		t.Errorf("Wrap should add context to error, got %v", wrapped2.Unwrap())
	}
}

func TestAll(t *testing.T) {
	r1 := Ok[int, error](1)
	r2 := Ok[int, error](2)
	r3 := Ok[int, error](3)
	result := All(r1, r2, r3)
	values := result.UnwrapOrPanic()
	if len(values) != 3 || values[0] != 1 || values[1] != 2 || values[2] != 3 {
		t.Errorf("All should collect all values, got %v", values)
	}

	err := errors.New("test error")
	r4 := Fail[int, error](err)
	result2 := All(r1, r4, r3)
	if result2.state != Failure {
		t.Errorf("All with Failure should return Failure")
	}
}

func TestUnwrapOrPanic(t *testing.T) {
	r1 := Ok[int, error](42)
	if r1.UnwrapOrPanic() != 42 {
		t.Errorf("UnwrapOrPanic on Success should return value")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("UnwrapOrPanic on Failure should panic")
		}
	}()
	err := errors.New("test error")
	r2 := Fail[int, error](err)
	r2.UnwrapOrPanic()
}

func TestAsyncThen(t *testing.T) {
	r := Ok[int, error](5)
	ch := AsyncThen(r, func(x int) Result[int, error] {
		time.Sleep(50 * time.Millisecond)
		return Ok[int, error](x * 2)
	})

	result := <-ch
	if result.UnwrapOrPanic() != 10 {
		t.Errorf("AsyncThen should multiply by 2, got %v", result.value)
	}
}

func TestAsyncThenWithTimeout(t *testing.T) {
	r1 := Ok[int, error](5)
	ch1 := AsyncThenWithTimeout(r1, func(x int) Result[int, error] {
		time.Sleep(50 * time.Millisecond)
		return Ok[int, error](x * 2)
	}, 100*time.Millisecond)

	result1 := <-ch1
	if result1.UnwrapOrPanic() != 10 {
		t.Errorf("AsyncThenWithTimeout should multiply by 2, got %v", result1.value)
	}

	ch2 := AsyncThenWithTimeout(r1, func(x int) Result[int, error] {
		time.Sleep(100 * time.Millisecond)
		return Ok[int, error](x * 2)
	}, 50*time.Millisecond)

	result2 := <-ch2
	if result2.state != Failure {
		t.Errorf("AsyncThenWithTimeout should fail on timeout")
	}
}

func TestMapErr(t *testing.T) {
	r1 := Ok[int, error](42)
	result := MapErr(r1, func(e error) error {
		return e
	})
	if result.UnwrapOrPanic() != 42 {
		t.Errorf("MapErr on Success should preserve value")
	}

	err := errors.New("test error")
	r2 := Fail[int, error](err)
	result2 := MapErr(r2, func(e error) error {
		return errors.New("mapped: " + e.Error())
	})
	if result2.Unwrap().Error() != "mapped: test error" {
		t.Errorf("MapErr should transform error, got %v", result2.Unwrap())
	}
}

func TestString(t *testing.T) {
	r1 := Ok[int, error](42)
	if r1.String() != "Ok(42)" {
		t.Errorf("String should return formatted success, got %v", r1.String())
	}

	err := errors.New("test error")
	r2 := Fail[int, error](err)
	if r2.String() != "Err(test error)" {
		t.Errorf("String should return formatted error, got %v", r2.String())
	}
}
