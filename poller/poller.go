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
var ErrTimeout = errors.New("poller: operation timed out")

type Result[T any] struct {
	Value T
	Err   error
	// Done indica se o polling deve parar.
	Done bool
	// Delay indica o tempo mínimo de espera antes da próxima execução.
	// Quando maior que o intervalo do [Poller], substitui o intervalo apenas
	// na próxima espera. Ignorado quando Done é true.
	Delay time.Duration
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
//
// Retorna [ErrTimeout] caso o tempo limite do [Poller] seja atingido, ou o erro
// da própria função, se houver. Caso o ctx informado seja cancelado ou atinja o
// próprio deadline, retorna o erro do ctx, sem envolvê-lo em [ErrTimeout].
func (p *Poller[T]) Poll(ctx context.Context) (T, error) {
	var zero T

	ctx, cancel := context.WithTimeoutCause(ctx, p.timeout, ErrTimeout)
	defer cancel()

	timer := time.NewTimer(p.interval)
	defer timer.Stop()

	for {
		res := p.fn(ctx)
		if res.Done {
			if res.Err != nil {
				return zero, res.Err
			}
			return res.Value, nil
		}

		timer.Reset(max(p.interval, res.Delay))

		select {
		case <-ctx.Done():
			if cause := context.Cause(ctx); errors.Is(cause, ErrTimeout) {
				return zero, fmt.Errorf("%w: %w", ErrTimeout, ctx.Err())
			}
			// Cancelamento (ou deadline) do ctx informado pelo caller: não é um
			// timeout do poller.
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
}
