package hasher

import (
	"testing"

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
