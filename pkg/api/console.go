package api

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/vm"
)

type ConsoleAuth struct {
	Token string `json:"token"`
}

// @Summary consoleVM
// @ID consoleVM
// @Description Interactive VM serial console over WebSocket. Not exposed as a request/response MCP tool.
// @x-maco {"expose":false,"readOnly":true,"transport":"unsupported"}
// @Tags vms
// @Param id path string true "id"
// @Success 101 "WebSocket upgrade"
// @Router /api/vms/{id}/console [get]
func (s *Server) consoleVM(w http.ResponseWriter, r *http.Request) {
	s.streamVMConsole(w, r, false)
}

// @Summary displayVM
// @ID displayVM
// @Description Interactive VM graphical display over WebSocket. Not exposed as a request/response MCP tool.
// @x-maco {"expose":false,"readOnly":true,"transport":"unsupported"}
// @Tags vms
// @Param id path string true "id"
// @Success 101 "WebSocket upgrade"
// @Router /api/vms/{id}/display [get]
func (s *Server) displayVM(w http.ResponseWriter, r *http.Request) {
	s.streamVMConsole(w, r, true)
}

func (s *Server) streamVMConsole(w http.ResponseWriter, r *http.Request, graphical bool) {
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

	m, st, err := s.engine.StatusVM(chi.URLParam(r, "id"))
	if err != nil || st.Phase != vm.PhaseRunning {
		_ = ws.Close(websocket.StatusPolicyViolation, "VM is not running")
		return
	}

	runDir := s.engine.Paths().VMRunDir(m.ID)
	socket := "console.sock"
	if graphical {
		socket = "vnc.sock"
	}

	conn, err := net.DialTimeout("unix", filepath.Join(runDir, socket), 3*time.Second)
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "console unavailable")
		return
	}
	defer func() { _ = conn.Close() }()

	ctx, stop := context.WithCancel(r.Context())
	defer stop()
	ws.SetReadLimit(1024 * 1024)
	if graphical {
		if err := ws.Write(ctx, websocket.MessageText, []byte(`{"type":"ready"}`)); err != nil {
			return
		}
	} else {
		if err := replaySerial(ctx, ws, filepath.Join(runDir, "serial.log")); err != nil {
			return
		}
	}

	stream := websocket.NetConn(ctx, ws, websocket.MessageBinary)
	defer func() { _ = stream.Close() }()
	done := make(chan struct{}, 1)
	go copyConsole(conn, stream, done)
	_, _ = io.Copy(stream, conn)
	_ = stream.Close()
	_ = conn.Close()
	<-done
}

func replaySerial(ctx context.Context, ws *websocket.Conn, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	offset := max(int64(0), info.Size()-64*1024)
	data := make([]byte, info.Size()-offset)
	n, err := file.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return err
	}

	return ws.Write(ctx, websocket.MessageBinary, data[:n])
}

func copyConsole(dst io.Writer, src io.Reader, done chan<- struct{}) {
	_, _ = io.Copy(dst, src)
	if conn, ok := dst.(net.Conn); ok {
		_ = conn.Close()
	}

	done <- struct{}{}
}
