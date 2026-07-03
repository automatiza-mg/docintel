// Package poller fornece um utilitário genérico para executar uma função
// repetidamente em intervalos regulares até que ela sinalize conclusão ou que
// um tempo limite seja atingido.
package poller

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrTimeout é retornado quando o tempo limite é atingido antes da conclusão do polling.
var ErrTimeout = errors.New("poller: operation timeout out")

type Result[T any] struct {
	Value T
	Err   error
	// Done indica se o polling deve parar.
	Done bool
}

type Poller[T any] struct {
	interval time.Duration
	timeout  time.Duration
	fn       func(ctx context.Context) Result[T]
}

// New cria um [Poller] que executa fn a cada interval até que fn sinalize conclusão
// ou que timeout seja atingido.
func New[T any](interval, timeout time.Duration, fn func(ctx context.Context) Result[T]) *Poller[T] {
	return &Poller[T]{
		interval: interval,
		timeout:  timeout,
		fn:       fn,
	}
}

// Poll executa a função repetidamente até que ela sinalize conclusão, retornando o valor produzido.
// Retorna [ErrTimeout] caso o tempo limite seja atingido, ou o erro da própria função, se houver.
func (p *Poller[T]) Poll(ctx context.Context) (T, error) {
	var zero T

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	res := p.fn(ctx)
	if res.Done {
		if res.Err != nil {
			return zero, res.Err
		}
		return res.Value, nil
	}

	for {
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("%w: %w", ErrTimeout, ctx.Err())
		case <-ticker.C:
			res := p.fn(ctx)
			if res.Done {
				if res.Err != nil {
					return zero, res.Err
				}
				return res.Value, nil
			}
		}
	}
}
