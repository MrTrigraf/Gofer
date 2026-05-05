package clipboard

// На Linux требует наличия xclip / xsel / wl-copy.
// На macOS использует встроенный pbcopy.
// На Windows — встроенный clip.

import (
	"fmt"

	"github.com/atotto/clipboard"
)

func Copy(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("clipboard copy: %w", err)
	}
	return nil
}
