package gormq

import (
	"errors"
	"time"
)

// ReconnectStrategy represents reconnect stratwgy type
type ReconnectStrategy int

// Reconnect stratery constants
const (
	ReconnectStrategyIncrease = iota
	ReconnectStrategyPlain
	ReconnectStrategyNoReconnect
)

const (
	defaultMaxAttempts = 100
	defaultTimeout     = 3 * time.Second
)

// Errors
var (
	ErrNoCBProvided = errors.New("No callback for conn() and close() provided")
)

type reconnector struct {
	maxAttempts int
	attemp      int
	timeout     time.Duration
	strategy    ReconnectStrategy
	connect     func() error
	close       func() error
}

func newReconnector(
	maxAttempts int,
	strategy ReconnectStrategy,
	timeout time.Duration,
	connect, close func() error) *reconnector {
	return &reconnector{
		maxAttempts: maxAttempts,
		timeout:     timeout,
		connect:     connect,
		close:       close,
	}
}

func (r *reconnector) reconnect() error {
	if r.connect != nil && r.close != nil {
		r.attemp++

		if err := r.close(); err != nil {
			time.Sleep(r.getTimeout())

			return r.reconnect()
		}
		if err := r.connect(); err != nil {
			time.Sleep(r.getTimeout())

			return r.reconnect()
		}

		return nil
	}

	return ErrNoCBProvided
}

func (r *reconnector) getTimeout() time.Duration {
	if r.strategy == ReconnectStrategyIncrease {
		secs := r.timeout.Seconds()
		r.timeout = time.Duration(secs + 1)
	}

	return r.timeout
}
