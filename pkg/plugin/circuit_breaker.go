package plugin

import (
	"context"
	"sync/atomic"
	"time"
)

type CircuitState int32

const (
	StateClosed   CircuitState = 0
	StateOpen     CircuitState = 1
	StateHalfOpen CircuitState = 2
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	state       atomic.Int32 // use int32 to represent state
	failures    atomic.Int32
	lastFailure atomic.Int64 // store Unix nanosecond timestamp
	config      CircuitBreakerConfig
	resetTimer  *time.Timer
	cancel      context.CancelFunc
	done        chan struct{}
	logger      Logger
}

func NewCircuitBreaker(ctx context.Context, config CircuitBreakerConfig, logger Logger) *CircuitBreaker {
	ctx, cancel := context.WithCancel(ctx)
	cb := &CircuitBreaker{
		config: config,
		cancel: cancel,
		done:   make(chan struct{}),
		logger: logger,
	}
	cb.state.Store(int32(StateClosed))

	// Start the reset timer
	cb.resetTimer = time.NewTimer(config.ResetInterval)
	go func() {
		cb.resetLoop(ctx)
	}()

	return cb
}

func (cb *CircuitBreaker) resetLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			cb.logger.Error("Panic in circuit breaker reset loop", "error", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cb.resetTimer.C:
			if cb.state.Load() == int32(StateOpen) {
				cb.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen))
				cb.failures.Store(0)
			}
			cb.resetTimer.Reset(cb.config.ResetInterval)
		}
	}
}

func (cb *CircuitBreaker) Allow() bool {
	if cb == nil {
		return true
	}

	currentState := CircuitState(cb.state.Load())
	switch currentState {
	case StateClosed:
		return true
	case StateHalfOpen:
		return true
	case StateOpen:
		lastFailureTime := time.Unix(0, cb.lastFailure.Load())
		if time.Since(lastFailureTime) > cb.config.TimeoutDuration {
			if cb.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen)) {
				cb.failures.Store(0)
			}
			return true
		}
		return false
	default:
		return true
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	if cb == nil {
		return
	}

	currentState := CircuitState(cb.state.Load())
	if currentState == StateHalfOpen {
		cb.state.CompareAndSwap(int32(StateHalfOpen), int32(StateClosed))
		cb.failures.Store(0)
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	if cb == nil {
		return
	}

	cb.lastFailure.Store(time.Now().UnixNano())
	failures := cb.failures.Add(1)

	if failures >= int32(cb.config.MaxFailures) {
		cb.state.CompareAndSwap(int32(StateClosed), int32(StateOpen))
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	if cb == nil {
		return StateClosed
	}
	return CircuitState(cb.state.Load())
}

func (cb *CircuitBreaker) Close() {
	if cb != nil {
		cb.cancel()
		select {
		case <-cb.done:
			return
		default:
			close(cb.done)
		}
	}
}
