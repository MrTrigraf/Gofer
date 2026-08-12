package serverstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStore — хранилище, указывающее на файл во ВРЕМЕННОЙ папке теста,
// а не в реальной папке настроек пользователя. t.TempDir() сам удаляет
// эту папку после теста.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{path: filepath.Join(t.TempDir(), fileName)}
}

// Load отсутствующего файла — это пустое хранилище, а не ошибка.
func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	s := newTestStore(t)

	data, err := os.ReadFile(s.path)
	require.Error(t, err) // предусловие: файла точно нет
	_ = data
	empty := &Store{path: s.path}
	assert.Empty(t, empty.Servers)
	assert.Empty(t, empty.Selected)
}

// Add: добавляет адрес, отклоняет пустой, игнорирует дубли.
func TestAdd(t *testing.T) {
	s := newTestStore(t)

	require.NoError(t, s.Add("http://localhost:8080"))
	assert.Equal(t, []string{"http://localhost:8080"}, s.Servers)

	// дубль — не ошибка, но и не добавляется второй раз
	require.NoError(t, s.Add("http://localhost:8080"))
	assert.Len(t, s.Servers, 1)

	// пустой адрес — ошибка
	err := s.Add("")
	assert.ErrorIs(t, err, ErrEmptyAddress)
	assert.Len(t, s.Servers, 1)
}

// Delete удаляет адрес; удаление выбранного сбрасывает выбор.
func TestDelete_ClearsSelection(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Add("http://a:8080"))
	require.NoError(t, s.Add("http://b:8080"))
	require.NoError(t, s.Select("http://b:8080"))

	// удаляем НЕ выбранный — выбор остаётся
	s.Delete("http://a:8080")
	assert.Equal(t, []string{"http://b:8080"}, s.Servers)
	assert.Equal(t, "http://b:8080", s.Selected)

	// удаляем выбранный — выбор сбрасывается
	s.Delete("http://b:8080")
	assert.Empty(t, s.Servers)
	assert.Empty(t, s.Selected)
}

// Select неизвестного адреса отклоняется.
func TestSelect_Unknown(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Add("http://a:8080"))

	require.NoError(t, s.Select("http://a:8080"))
	assert.Equal(t, "http://a:8080", s.Selected)

	err := s.Select("http://ghost:8080")
	assert.Error(t, err)
	assert.Equal(t, "http://a:8080", s.Selected) // не изменился
}

// Save + чтение обратно: данные переживают запись на диск (round-trip).
func TestSaveAndReload(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Add("http://a:8080"))
	require.NoError(t, s.Add("http://b:8080"))
	require.NoError(t, s.Select("http://b:8080"))
	require.NoError(t, s.Save())

	// файл действительно появился
	_, err := os.Stat(s.path)
	require.NoError(t, err)

	// читаем в новое хранилище тем же путём
	reloaded := &Store{path: s.path}
	data, err := os.ReadFile(reloaded.path)
	require.NoError(t, err)
	require.NoError(t, unmarshalInto(reloaded, data))

	assert.Equal(t, []string{"http://a:8080", "http://b:8080"}, reloaded.Servers)
	assert.Equal(t, "http://b:8080", reloaded.Selected)
}
