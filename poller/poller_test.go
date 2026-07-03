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
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
		if errors.Is(err, ErrTimeout) {
			t.Fatalf("cancellation should not be reported as ErrTimeout, got %v", err)
		}
	})
}

func TestPoll_ParentDeadlineExceeded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()

		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			return Result[int]{Done: false}
		})

		_, err := p.Poll(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}
		if errors.Is(err, ErrTimeout) {
			t.Fatalf("parent deadline should not be reported as ErrTimeout, got %v", err)
		}
	})
}

func TestPoll_DelayOverridesInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const delay = 40 * time.Millisecond

		var times []time.Time
		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			times = append(times, time.Now())
			if len(times) == 1 {
				return Result[int]{Done: false, Delay: delay}
			}
			return Result[int]{Value: 1, Done: true}
		})

		_, err := p.Poll(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(times) != 2 {
			t.Fatalf("expected fn to be called twice, got %d", len(times))
		}
		if got := times[1].Sub(times[0]); got != delay {
			t.Fatalf("expected second call after %v, got %v", delay, got)
		}
	})
}

func TestPoll_DelayShorterThanIntervalIsIgnored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var times []time.Time
		p := New(testInterval, testTimeout, func(ctx context.Context) Result[int] {
			times = append(times, time.Now())
			if len(times) == 1 {
				return Result[int]{Done: false, Delay: time.Millisecond}
			}
			return Result[int]{Value: 1, Done: true}
		})

		_, err := p.Poll(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := times[1].Sub(times[0]); got != testInterval {
			t.Fatalf("expected second call after %v, got %v", testInterval, got)
		}
	})
}
