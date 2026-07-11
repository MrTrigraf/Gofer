package clipboard

// На Linux требует наличия xclip / xsel / wl-copy.
// На macOS использует встроенный pbcopy.
// На Windows — встроенный clip.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/atotto/clipboard"
)

var errNoBackend = errors.New("no clipboard backend found: install xclip, xsel or wl-copy")

var (
	detectOnce sync.Once
	backendErr error
)

func Available() error {
	detectOnce.Do(func() {
		backendErr = detectBackend()
	})
	return backendErr
}

func detectBackend() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return nil
		}
	}

	if _, err := exec.LookPath("xclip"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return nil
	}

	return errNoBackend
}

func Copy(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("clipboard copy: %w", err)
	}
	return nil
}
