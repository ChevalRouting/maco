package vm

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	consoleEscape byte = 0x01
	consoleQuit   byte = 'q'
)

func (d *Driver) Console(id string) error {
	if d.Status(id).Phase != PhaseRunning {
		return fmt.Errorf("vm %s is not running", id)
	}

	conn, err := net.DialTimeout("unix", d.consolePath(id), 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect console: %w", err)
	}

	defer func() { _ = conn.Close() }()

	_, _ = fmt.Fprintln(os.Stderr, "connected to console (Ctrl+A Q to quit)")

	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return err
		}

		defer func() { _ = term.Restore(fd, state) }()
	}

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(os.Stdout, conn)
		done <- struct{}{}
	}()

	go func() {
		_ = proxyConsoleInput(conn, os.Stdin)
		done <- struct{}{}
	}()

	<-done
	return nil
}

func proxyConsoleInput(dst io.Writer, src io.Reader) error {
	buf := make([]byte, 1)
	escaped := false
	for {
		n, err := src.Read(buf)
		if n > 0 {
			b := buf[0]
			switch {
			case escaped && (b == consoleQuit || b == 'Q'):
				return nil
			case escaped && b == consoleEscape:
				escaped = false
				if _, err := dst.Write([]byte{consoleEscape}); err != nil {
					return err
				}
			case escaped:
				escaped = false
				if _, err := dst.Write([]byte{b}); err != nil {
					return err
				}
			case b == consoleEscape:
				escaped = true
			default:
				if _, err := dst.Write([]byte{b}); err != nil {
					return err
				}
			}
		}

		if err != nil {
			return err
		}
	}
}
