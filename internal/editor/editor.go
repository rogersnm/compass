package editor

import (
	"fmt"
	"os"
	"os/exec"
)

func editorCmd() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return "vi"
}

func Open(filepath string) error {
	editor := editorCmd()
	cmd := exec.Command(editor, filepath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %q: %w", editor, err)
	}
	return nil
}
