package hasher

import (
	"strings"
	"testing"

	"github.com/gofer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	hasher := New()
	password := "secret123"

	hash, err := hasher.Hash(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCompare(t *testing.T) {
	hasher := New()
	password := "secret123"

	hash, err := hasher.Hash(password)

	require.NoError(t, err)

	err = hasher.Compare(hash, password)
	assert.NoError(t, err)

	err = hasher.Compare(hash, "wrongpassword")
	assert.Error(t, err) // есть ошибка = неправильный пароль
}

func TestHash_LongPasswordRejected(t *testing.T) {
	hasher := New()

	longASCII := strings.Repeat("a", 73)
	_, err := hasher.Hash(longASCII)
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrPasswordTooLong)

	longCyrillic := strings.Repeat("а", 37)
	require.Greater(t, len(longCyrillic), 72)
	_, err = hasher.Hash(longCyrillic)
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrPasswordTooLong)
}

func TestCompare_AcceptsLongPassword(t *testing.T) {
	hasher := New()

	// Хешируем ровно 72 байта — предельный валидный вход.
	base := strings.Repeat("a", 72)
	hash, err := hasher.Hash(base)
	require.NoError(t, err)

	// Пароль с теми же первыми 72 байтами, но длиннее — Compare его ПРИМЕТ,
	// подтверждая, что за границей 72 байт защиты нет.
	longer := base + "EXTRA"
	err = hasher.Compare(hash, longer)
	assert.NoError(t, err) // совпало — обрезание на стороне Compare живёт
}
