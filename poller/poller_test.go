package poller

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

const (
	testInterval = 10 * time.Millisecond
	testTimeout  = 100 * time.Millisecond
)

func TestPoll_ImmediateSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls int
		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			calls++
			return Result[int]{Value: 42, Done: true}
		})

		got, err := p.Poll(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 42 {
			t.Fatalf("expected value 42, got %d", got)
		}
		if calls != 1 {
			t.Fatalf("expected fn to be called once, got %d", calls)
		}
	})
}

func TestPoll_ImmediateError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("boom")
		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			return Result[int]{Done: true, Err: wantErr}
		})

		got, err := p.Poll(context.Background())
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}
		if got != 0 {
			t.Fatalf("expected zero value, got %d", got)
		}
	})
}

func TestPoll_SuccessAfterTicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const wantCalls = 4
		var calls int
		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			calls++
			if calls < wantCalls {
				return Result[int]{Done: false}
			}
			return Result[int]{Value: 7, Done: true}
		})

		got, err := p.Poll(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 7 {
			t.Fatalf("expected value 7, got %d", got)
		}
		if calls != wantCalls {
			t.Fatalf("expected fn to be called %d times, got %d", wantCalls, calls)
		}
	})
}

func TestPoll_ErrorAfterTicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("late failure")
		var calls int
		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			calls++
			if calls < 3 {
				return Result[int]{Done: false}
			}
			return Result[int]{Done: true, Err: wantErr}
		})

		_, err := p.Poll(context.Background())
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}
		if calls != 3 {
			t.Fatalf("expected fn to be called 3 times, got %d", calls)
		}
	})
}

func TestPoll_Timeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			return Result[int]{Done: false}
		})

		_, err := p.Poll(context.Background())
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("expected ErrTimeout, got %v", err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected wrapped context.DeadlineExceeded, got %v", err)
		}
	})
}

func TestPoll_ParentContextCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			return Result[int]{Done: false}
		})

		go func() {
			time.Sleep(25 * time.Millisecond)
			cancel()
		}()

		_, err := p.Poll(ctx)
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("expected ErrTimeout, got %v", err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected wrapped context.Canceled, got %v", err)
		}
	})
}
