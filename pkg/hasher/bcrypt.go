// Package hasher — хеширование паролей через bcrypt.
//
// Пакет отвечает за перевод ошибок bcrypt на доменный язык: наверх уходят
// domain-ошибки, а не пакетные. Благодаря этому usecase-слой ничего
// не знает про bcrypt и может быть заменён на другой алгоритм без правок
// в бизнес-логике.
package hasher

import (
	"errors"
	"fmt"

	"github.com/gofer/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct{}

func New() *BcryptHasher {
	return &BcryptHasher{}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

func (h *BcryptHasher) Compare(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return domain.ErrInvalidCredentials
	}
	return fmt.Errorf("compare password hash: %w", err)
}
