package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"

	"github.com/coder/websocket"
	"github.com/creack/pty"
	"github.com/m-vinc/maco/pkg/auth"
)

type shellResize struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// @Summary hostConsole
// @ID hostConsole
// @Description Interactive host shell over WebSocket. Not exposed as a request/response MCP tool.
// @x-maco {"expose":false,"readOnly":true,"transport":"unsupported"}
// @Tags host
// @Success 101 "WebSocket upgrade"
// @Router /api/host/console [get]
func (s *Server) hostConsole(w http.ResponseWriter, r *http.Request) {
	ws, claims, release, err := s.acceptAuthenticated(w, r)
	if err != nil {
		return
	}

	defer release()
	defer func() { _ = ws.CloseNow() }()

	if !auth.CanMutate(claims.Role) {
		_ = ws.Close(websocket.StatusPolicyViolation, "administrator role required")
		return
	}

	ws.SetReadLimit(1024 * 1024)
	ctx, stop := context.WithCancel(r.Context())
	defer stop()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	cmd := exec.CommandContext(ctx, shell, "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "shell unavailable")
		return
	}

	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	go streamShellOutput(ctx, ws, ptmx, stop)

	for {
		kind, data, err := ws.Read(ctx)
		if err != nil {
			return
		}

		if kind == websocket.MessageText {
			var size shellResize
			if json.Unmarshal(data, &size) == nil && size.Cols > 0 && size.Rows > 0 {
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: size.Cols, Rows: size.Rows})
			}

			continue
		}

		if _, err := ptmx.Write(data); err != nil {
			return
		}
	}
}

func streamShellOutput(ctx context.Context, ws *websocket.Conn, ptmx *os.File, stop func()) {
	defer stop()

	buffer := make([]byte, 32*1024)
	for {
		n, err := ptmx.Read(buffer)
		if n > 0 {
			if writeErr := ws.Write(ctx, websocket.MessageBinary, buffer[:n]); writeErr != nil {
				return
			}
		}

		if err != nil {
			return
		}
	}
}
