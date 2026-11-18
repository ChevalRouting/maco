package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

func editManifestLoop(ctx context.Context, original []byte, commit func([]byte) (bool, error)) error {
	tmpPath, cleanup, err := writeTempFile(original)
	if err != nil {
		return err
	}

	defer cleanup()

	for {
		if err := runEditor(ctx, tmpPath); err != nil {
			return err
		}

		edited, err := os.ReadFile(tmpPath)
		if err != nil {
			return err
		}

		if bytes.Equal(edited, original) {
			log.Info().Msg("no changes, not saved")
			return nil
		}

		retryable, err := commit(edited)
		if err == nil {
			return nil
		}

		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		if retryable && promptReedit() {
			continue
		}

		return fmt.Errorf("edit aborted, not saved")
	}
}

func writeTempFile(content []byte) (string, func(), error) {
	tmp, err := os.CreateTemp("", "maco-edit-*.yml")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", nil, fmt.Errorf("write temp file: %w", err)
	}

	_ = tmp.Close()
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}

func runEditor(ctx context.Context, path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	parts := strings.Fields(editor)
	args := append(parts[1:], path)
	editorCmd := exec.CommandContext(ctx, parts[0], args...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor exited: %w", err)
	}

	return nil
}

func promptReedit() bool {
	_, _ = fmt.Fprint(os.Stderr, "invalid: [e]dit again or [x] abort? ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}

	switch strings.TrimSpace(strings.ToLower(line)) {
	case "", "e", "edit":
		return true
	default:
		return false
	}
}
