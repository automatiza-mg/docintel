package docintel

import "time"

// Valores padrão de polling. A análise em lote usa um tempo limite maior por
// processar múltiplos documentos.
const (
	defaultPollInterval     = 2 * time.Second
	defaultPollTimeout      = 5 * time.Minute
	defaultBatchPollTimeout = 30 * time.Minute
)

// pollConfig agrupa as configurações de polling aplicadas por chamada.
type pollConfig struct {
	interval time.Duration
	timeout  time.Duration
}

// PollOption configura uma chamada a [Client.PollResult] ou [Client.PollBatchResult].
type PollOption func(*pollConfig)

// WithPollInterval define o intervalo entre as consultas (padrão 2s).
func WithPollInterval(interval time.Duration) PollOption {
	return func(c *pollConfig) {
		c.interval = interval
	}
}

// WithPollTimeout define o tempo máximo de espera do polling.
//
// O padrão é 5min em [Client.PollResult] e 30min em [Client.PollBatchResult]. O
// deadline do [context.Context] informado na chamada também é respeitado; o que
// ocorrer primeiro encerra o polling.
func WithPollTimeout(timeout time.Duration) PollOption {
	return func(c *pollConfig) {
		c.timeout = timeout
	}
}

// newPollConfig resolve a configuração de polling a partir do tempo limite padrão
// e das opções informadas.
func newPollConfig(defaultTimeout time.Duration, opts []PollOption) pollConfig {
	cfg := pollConfig{
		interval: defaultPollInterval,
		timeout:  defaultTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
