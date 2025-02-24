# Tiny Result Library

A lightweight Go library for handling success and failure results using a monadic-style API with generics. This package provides a `Result[T, E]` type to encapsulate the outcome of operations, offering methods for chaining, mapping, and safely unwrapping values or errors.

## Features

- **Type-Safe Results**: Use generics to define success values (`T`) and error types (`E`).
- **Monadic Operations**: Chain computations with `Then` and transform values/errors with `Map` and `MapErr`.
- **Error Handling**: Safely handle failures with `Unwrap`, `OrElse`, or `Wrap`.
- **Asynchronous Support**: Process results asynchronously with `AsyncThen` and `AsyncThenWithTimeout`.
- **Aggregation**: Combine multiple results with `All`.

## Installation

To use this library in your Go project, run:

```bash
go get github.com/xxlv/go-tinylib
```

## Usage

Here’s a quick example to demonstrate the core functionality:

```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/xxlv/go-tinylib"
)

func divide(a, b int) tiny.Result[int, error] {
	if b == 0 {
		return tiny.Fail[int, error](errors.New("division by zero"))
	}
	return tiny.Ok[int, error](a / b)
}

func main() {
	// Success case
	result := tiny.Ok[int, error](10).
		Then(func(v int) tiny.Result[int, error] {
			return divide(v, 2)
		})
	fmt.Println(result) // Output: Ok(5)

	// Failure case
	result = tiny.Ok[int, error](10).
		Then(func(v int) tiny.Result[int, error] {
			return divide(v, 0)
		}).
		Wrap("failed to process")
	fmt.Println(result) // Output: Err(failed to process: division by zero)

	// Async with timeout
	ch := tiny.AsyncThenWithTimeout(
		tiny.Ok[int, error](20),
		func(v int) tiny.Result[int, error] {
			time.Sleep(2 * time.Second) // Simulate long operation
			return tiny.Ok[int, error](v * 2)
		},
		1*time.Second, // Timeout after 1 second
	)
	fmt.Println(<-ch) // Output: Err(operation timed out after 1s)
}
```

## API Overview

- **`Ok[T, E](value T)`**: Creates a successful `Result` with a value.
- **`Fail[T, E](err E)`**: Creates a failed `Result` with an error.
- **`Then(fn func(T) Result[T, E])`**: Chains a function on a successful `Result`.
- **`Map[T, U, E](r, fn)`**: Transforms the value of a `Result` or propagates the error.
- **`OrElse(defaultVal T)`**: Returns the value or a default if the `Result` failed.
- **`Wrap(msg string)`**: Wraps an error with additional context.
- **`Unwrap()`**: Returns the error or a zero value if successful.
- **`All[T, E](results ...Result[T, E])`**: Combines multiple `Result`s into one.
- **`UnwrapOrPanic()`**: Extracts the value or panics on failure.
- **`MapErr[T, E, F](r, fn)`**: Transforms the error of a failed `Result`.
- **`AsyncThen(fn)`**: Asynchronously applies a function to a `Result`.
- **`AsyncThenWithTimeout(fn, timeout)`**: Asynchronously applies a function with a timeout.

See the [source code](./pkg/tiny.go) for detailed documentation.

## Requirements

- Go 1.18 or later (due to generics support).

## Contributing

Contributions are welcome! Please submit a pull request or open an issue on the [repository](https://github.com/xxlv/go-tinylib) with your suggestions or bug reports.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

---

_Generated with assistance from Grok, created by xAI._
