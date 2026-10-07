package api

import (
	"bytes"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if remaining := c.limit - c.buf.Len(); remaining > 0 {
		if len(p) > remaining {
			c.buf.Write(p[:remaining])
		} else {
			c.buf.Write(p)
		}
	}

	return len(p), nil
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		body := &cappedBuffer{limit: 2048}
		ww.Tee(body)

		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}

		event := log.Info()
		switch {
		case status >= 500:
			event = log.Error()
		case status >= 400:
			event = log.Warn()
		}

		event = event.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", status).
			Int("bytes", ww.BytesWritten()).
			Dur("took", time.Since(start)).
			Str("remote", r.RemoteAddr)
		if status >= 400 {
			if message := strings.TrimSpace(body.buf.String()); message != "" {
				event = event.Str("response", message)
			}
		}

		event.Msg("request")
	})
}

func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			log.Error().
				Interface("panic", rec).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Bytes("stack", debug.Stack()).
				Msg("panic recovered")
			writeError(w, http.StatusInternalServerError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}
