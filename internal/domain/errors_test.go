package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrors(t *testing.T) {
	err := ErrUserNotFound
	if !errors.Is(err, ErrUserNotFound) {
		t.Error("expected ErrUserNotFound")
	}

	wrapped := fmt.Errorf("usecase: %w", ErrUserNotFound)
	if !errors.Is(wrapped, ErrUserNotFound) {
		t.Error("expected wrapped ErrUserNotFound")
	}
}
