package plugin

import "fmt"

// ErrPluginNotFound represents an error when a plugin cannot be found
type ErrPluginNotFound struct {
	Name string
}

func (e ErrPluginNotFound) Error() string {
	return fmt.Sprintf("plugin not found: %s", e.Name)
}

// ErrPluginExists represents an error when a plugin already exists
type ErrPluginExists struct {
	Name string
}

func (e ErrPluginExists) Error() string {
	return fmt.Sprintf("plugin already exists: %s", e.Name)
}

// ErrFuncNotFound represents an error when a function cannot be found
type ErrFuncNotFound struct {
	Name string
}

func (e ErrFuncNotFound) Error() string {
	return fmt.Sprintf("function not found: %s", e.Name)
}

// ErrCircuitOpen represents an error when the circuit breaker is open
type ErrCircuitOpen struct {
	Name string
}

func (e ErrCircuitOpen) Error() string {
	return fmt.Sprintf("circuit breaker is open for plugin: %s", e.Name)
}

// ErrPluginTimeout represents an error when a plugin operation times out
type ErrPluginTimeout struct {
	Name string
}

func (e ErrPluginTimeout) Error() string {
	return fmt.Sprintf("plugin operation timed out: %s", e.Name)
}

// ErrPluginInit represents an error during plugin initialization
type ErrPluginInit struct {
	Name string
	Err  error
}

func (e ErrPluginInit) Error() string {
	return fmt.Sprintf("failed to initialize plugin %s: %v", e.Name, e.Err)
}

// ErrPluginFree represents an error during plugin cleanup
type ErrPluginFree struct {
	Name string
	Err  error
}

func (e ErrPluginFree) Error() string {
	return fmt.Sprintf("failed to free plugin %s: %v", e.Name, e.Err)
}

// IsCircuitOpenError checks if the error is a circuit breaker open error
func IsCircuitOpenError(err error) bool {
	_, ok := err.(ErrCircuitOpen)
	return ok
}

// IsPluginNotFoundError checks if the error is a plugin not found error
func IsPluginNotFoundError(err error) bool {
	_, ok := err.(ErrPluginNotFound)
	return ok
}

// IsFuncNotFoundError checks if the error is a function not found error
func IsFuncNotFoundError(err error) bool {
	_, ok := err.(ErrFuncNotFound)
	return ok
}

// IsPluginTimeoutError checks if the error is a plugin timeout error
func IsPluginTimeoutError(err error) bool {
	_, ok := err.(ErrPluginTimeout)
	return ok
}

// ErrCircuitBreakerOpen represents a circuit breaker open error
type ErrCircuitBreakerOpen struct {
	Name string
}

func (e *ErrCircuitBreakerOpen) Error() string {
	return fmt.Sprintf("circuit breaker is open for plugin: %s", e.Name)
}
