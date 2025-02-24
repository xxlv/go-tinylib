package tiny

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestThenWithContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		input   Result[string, error]
		fn      func(string) Result[string, error]
		want    Result[string, error]
		wantErr bool
	}{
		{
			name:  "success case",
			ctx:   context.Background(),
			input: Ok[string, error]("hello"),
			fn: func(s string) Result[string, error] {
				return Ok[string, error](s + " world")
			},
			want:    Ok[string, error]("hello world"),
			wantErr: false,
		},
		{
			name:    "failure case",
			ctx:     context.Background(),
			input:   Fail[string, error](errors.New("failed")),
			fn:      func(s string) Result[string, error] { return Ok[string, error](s) },
			want:    Fail[string, error](errors.New("failed")),
			wantErr: true,
		},
		{
			name:    "context canceled",
			ctx:     func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			input:   Ok[string, error]("hello"),
			fn:      func(s string) Result[string, error] { return Ok[string, error](s) },
			want:    Fail[string, error](context.Canceled),
			wantErr: true, // Fixed: Expect a failure due to context cancellation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ThenWithContext(tt.ctx, tt.input, tt.fn)
			if tt.wantErr && got.state != Failure {
				t.Errorf("ThenWithContext() expected failure, got %v", got)
				return
			}
			if !tt.wantErr && got.state != Success {
				t.Errorf("ThenWithContext() expected success, got %v", got)
				return
			}
			if tt.wantErr && got.fault.Error() != tt.want.fault.Error() {
				t.Errorf("ThenWithContext() error = %v, want %v", got.fault, tt.want.fault)
			}
			if !tt.wantErr && got.value != tt.want.value {
				t.Errorf("ThenWithContext() = %v, want %v", got.value, tt.want.value)
			}
		})
	}
}

func TestMapWithContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		input   Result[int, error]
		fn      func(int) (string, error)
		want    Result[string, error]
		wantErr bool
	}{
		{
			name:  "success case",
			ctx:   context.Background(),
			input: Ok[int, error](42),
			fn: func(i int) (string, error) {
				return fmt.Sprintf("%d", i), nil
			},
			want:    Ok[string, error]("42"),
			wantErr: false,
		},
		{
			name:    "failure case",
			ctx:     context.Background(),
			input:   Fail[int, error](errors.New("oops")),
			fn:      func(i int) (string, error) { return fmt.Sprintf("%d", i), nil },
			want:    Fail[string, error](errors.New("oops")),
			wantErr: true,
		},
		{
			name:    "context canceled",
			ctx:     func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			input:   Ok[int, error](42),
			fn:      func(i int) (string, error) { return fmt.Sprintf("%d", i), nil },
			want:    Fail[string, error](context.Canceled),
			wantErr: true, // Fixed: Expect failure due to canceled context
		},
		{
			name:    "function fails",
			ctx:     context.Background(),
			input:   Ok[int, error](42),
			fn:      func(i int) (string, error) { return "", errors.New("fn failed") },
			want:    Fail[string, error](errors.New("fn failed")),
			wantErr: true, // Fixed: Expect failure due to function error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapWithContext(tt.ctx, tt.input, tt.fn)
			if tt.wantErr && got.state != Failure {
				t.Errorf("MapWithContext() expected failure, got %v", got)
				return
			}
			if !tt.wantErr && got.state != Success {
				t.Errorf("MapWithContext() expected success, got %v", got)
				return
			}
			if tt.wantErr && got.fault.Error() != tt.want.fault.Error() {
				t.Errorf("MapWithContext() error = %v, want %v", got.fault, tt.want.fault)
			}
			if !tt.wantErr && got.value != tt.want.value {
				t.Errorf("MapWithContext() = %v, want %v", got.value, tt.want.value)
			}
		})
	}
}

func TestAsyncThenWithContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		input   Result[string, error]
		fn      func(string) Result[string, error]
		want    Result[string, error]
		wantErr bool
	}{
		{
			name:  "success case",
			ctx:   context.Background(),
			input: Ok[string, error]("start"),
			fn: func(s string) Result[string, error] {
				time.Sleep(50 * time.Millisecond)
				return Ok[string, error](s + " done")
			},
			want:    Ok[string, error]("start done"),
			wantErr: false,
		},
		{
			name:    "failure case",
			ctx:     context.Background(),
			input:   Fail[string, error](errors.New("failed")),
			fn:      func(s string) Result[string, error] { return Ok[string, error](s) },
			want:    Fail[string, error](errors.New("failed")),
			wantErr: true,
		},
		{
			name:    "context canceled before start",
			ctx:     func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			input:   Ok[string, error]("start"),
			fn:      func(s string) Result[string, error] { return Ok[string, error](s) },
			want:    Fail[string, error](context.Canceled),
			wantErr: true,
		},
		{
			name: "context canceled during execution",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)
				return ctx // Removed defer cancel() to avoid premature cancellation
			}(),
			input: Ok[string, error]("start"),
			fn: func(s string) Result[string, error] {
				time.Sleep(50 * time.Millisecond) // Longer than context timeout
				return Ok[string, error](s + " late")
			},
			want:    Fail[string, error](context.DeadlineExceeded),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := AsyncThenWithContext(tt.ctx, tt.input, tt.fn)
			got := <-ch
			if tt.wantErr && got.state != Failure {
				t.Errorf("AsyncThenWithContext() expected failure, got %v", got)
				return
			}
			if !tt.wantErr && got.state != Success {
				t.Errorf("AsyncThenWithContext() expected success, got %v", got)
				return
			}
			if tt.wantErr && got.fault.Error() != tt.want.fault.Error() {
				t.Errorf("AsyncThenWithContext() error = %v, want %v", got.fault, tt.want.fault)
			}
			if !tt.wantErr && got.value != tt.want.value {
				t.Errorf("AsyncThenWithContext() = %v, want %v", got.value, tt.want.value)
			}
		})
	}
}

func TestAsyncThenWithContextAndTimeout(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		input   Result[string, error]
		fn      func(string) Result[string, error]
		timeout time.Duration
		want    Result[string, error]
		wantErr bool
	}{
		{
			name:  "success case",
			ctx:   context.Background(),
			input: Ok[string, error]("start"),
			fn: func(s string) Result[string, error] {
				time.Sleep(50 * time.Millisecond)
				return Ok[string, error](s + " done")
			},
			timeout: 100 * time.Millisecond,
			want:    Ok[string, error]("start done"),
			wantErr: false,
		},
		{
			name:    "failure case",
			ctx:     context.Background(),
			input:   Fail[string, error](errors.New("failed")),
			fn:      func(s string) Result[string, error] { return Ok[string, error](s) },
			timeout: 100 * time.Millisecond,
			want:    Fail[string, error](errors.New("failed")),
			wantErr: true,
		},
		{
			name:    "context canceled before start",
			ctx:     func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			input:   Ok[string, error]("start"),
			fn:      func(s string) Result[string, error] { return Ok[string, error](s) },
			timeout: 100 * time.Millisecond,
			want:    Fail[string, error](context.Canceled),
			wantErr: true,
		},
		{
			name: "context deadline before timeout",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)
				return ctx
			}(),
			input: Ok[string, error]("start"),
			fn: func(s string) Result[string, error] {
				time.Sleep(50 * time.Millisecond) // Longer than context timeout
				return Ok[string, error](s + " late")
			},
			timeout: 100 * time.Millisecond,
			want:    Fail[string, error](context.DeadlineExceeded),
			wantErr: true,
		},
		{
			name:  "timeout before context deadline",
			ctx:   context.Background(),
			input: Ok[string, error]("start"),
			fn: func(s string) Result[string, error] {
				time.Sleep(100 * time.Millisecond) // Longer than timeout
				return Ok[string, error](s + " late")
			},
			timeout: 50 * time.Millisecond,
			want:    Fail[string, error](fmt.Errorf("operation timed out after %v", 50*time.Millisecond)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := AsyncThenWithContextAndTimeout(tt.ctx, tt.input, tt.fn, tt.timeout)
			got := <-ch
			if tt.wantErr && got.state != Failure {
				t.Errorf("AsyncThenWithContextAndTimeout() expected failure, got %v", got)
				return
			}
			if !tt.wantErr && got.state != Success {
				t.Errorf("AsyncThenWithContextAndTimeout() expected success, got %v", got)
				return
			}
			if tt.wantErr {
				if tt.name == "timeout before context deadline" {
					// Special case for timeout error since the error message includes a dynamic duration
					if !strings.Contains(got.fault.Error(), "operation timed out after") {
						t.Errorf("AsyncThenWithContextAndTimeout() error = %v, want error containing 'operation timed out after'", got.fault)
					}
				} else if got.fault.Error() != tt.want.fault.Error() {
					t.Errorf("AsyncThenWithContextAndTimeout() error = %v, want %v", got.fault, tt.want.fault)
				}
			}
			if !tt.wantErr && got.value != tt.want.value {
				t.Errorf("AsyncThenWithContextAndTimeout() = %v, want %v", got.value, tt.want.value)
			}
		})
	}
}
