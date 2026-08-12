package serverstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDir   = "gofer"
	fileName = "servers.json"
)

const (
	dirPerm  os.FileMode = 0o700
	filePerm os.FileMode = 0o600
)

var ErrEmptyAddress = errors.New("server address is empty")

type Store struct {
	Servers  []string `json:"servers"`
	Selected string   `json:"selected"`

	path string // откуда грузим/куда сохраняем; в JSON не пишется
}

func Load() (*Store, error) {
	path, err := storePath()
	if err != nil {
		return nil, fmt.Errorf("serverstore: resolve path: %w", err)
	}

	s := &Store{path: path}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil // первый запуск: пустое хранилище
	}
	if err != nil {
		return nil, fmt.Errorf("serverstore: read %q: %w", path, err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("serverstore: parse %q: %w", path, err)
	}
	return s, nil
}

func (s *Store) Add(addr string) error {
	if addr == "" {
		return ErrEmptyAddress
	}
	for _, existing := range s.Servers {
		if existing == addr {
			return nil
		}
	}
	s.Servers = append(s.Servers, addr)
	return nil
}

func (s *Store) Delete(addr string) {
	kept := s.Servers[:0]
	for _, existing := range s.Servers {
		if existing != addr {
			kept = append(kept, existing)
		}
	}
	s.Servers = kept
	if s.Selected == addr {
		s.Selected = ""
	}
}

func (s *Store) Select(addr string) error {
	for _, existing := range s.Servers {
		if existing == addr {
			s.Selected = addr
			return nil
		}
	}
	return fmt.Errorf("serverstore: select unknown address %q", addr)
}

func (s *Store) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("serverstore: encode: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("serverstore: create dir %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, fileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("serverstore: temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("serverstore: write temp: %w", err)
	}
	if err := tmp.Chmod(filePerm); err != nil {
		tmp.Close()
		return fmt.Errorf("serverstore: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("serverstore: close temp: %w", err)
	}

	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("serverstore: rename into place: %w", err)
	}
	return nil
}

// storePath возвращает полный путь к файлу-хранилищу внутри папки настроек
// пользователя (например, ~/.config/gofer/servers.json на Linux).
func storePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(dir, appDir, fileName), nil
}

// unmarshalInto разбирает JSON в переданное хранилище, сохраняя его path.
// Вынесено для переиспользования в тестах (Load ходит в UserConfigDir и
// в тестах напрямую неприменим).
func unmarshalInto(s *Store, data []byte) error {
	path := s.path
	if err := json.Unmarshal(data, s); err != nil {
		return fmt.Errorf("serverstore: parse: %w", err)
	}
	s.path = path
	return nil
}
