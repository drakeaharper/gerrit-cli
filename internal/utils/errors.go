package utils

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrConfigNotFound     = errors.New("configuration not found")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrConnectionFailed   = errors.New("connection failed")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrChangeNotFound     = errors.New("change not found")
	ErrInvalidChangeID    = errors.New("invalid change ID")
	ErrGitNotFound        = errors.New("git not found in PATH")
	ErrNotGitRepo         = errors.New("not in a git repository")
)

type GerritError struct {
	Code    string
	Message string
	Details string
}

func (e *GerritError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewGerritError(code, message, details string) *GerritError {
	return &GerritError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func ExitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func CheckError(err error) {
	if err != nil {
		ExitWithError(err)
	}
}

func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrChangeNotFound) || errors.Is(err, ErrConfigNotFound)
}

func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthenticationFailed)
}

func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnectionFailed)
}