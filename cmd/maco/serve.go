package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/m-vinc/maco/pkg/api"
	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/jobs"
	"github.com/m-vinc/maco/pkg/tlscert"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	var (
		addr      string
		devDir    string
		redisURL  string
		reconcile bool
		useTLS    bool
		tlsCert   string
		tlsKey    string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "start the web UI and HTTP API",
		RunE: func(cmd *cobra.Command, _ []string) error {
			secret, err := auth.LoadOrCreateSecret(filepath.Join(cli.Paths.Root, "jwt.secret"))
			if err != nil {
				return err
			}

			created, generated, err := eng().EnsureAdmin(cmd.Context(), os.Getenv("MACO_ADMIN_PASSWORD"))
			if err != nil {
				return err
			}

			if created {
				if generated != "" {
					log.Warn().Str("user", "admin").Str("password", generated).Msg("created admin with generated password")
				} else {
					log.Info().Str("user", "admin").Msg("created admin from MACO_ADMIN_PASSWORD")
				}
			}

			if _, err := eng().SeedDefaultTemplates(cmd.Context()); err != nil {
				log.Warn().Err(err).Msg("seed default templates")
			}

			static := resolveStaticFS(devDir)
			queue, err := jobs.New(eng(), redisURL)
			if err != nil {
				return err
			}

			if err := queue.Start(cmd.Context()); err != nil {
				queue.Close()
				return err
			}
			defer queue.Close()

			if reconcile {
				if _, err := queue.Submit(cmd.Context(), jobs.Payload{Action: "boot.reconcile", Target: "host"}); err != nil {
					log.Error().Err(err).Msg("enqueue boot reconcile")
				}
			}

			certFile, keyFile := "", ""
			if useTLS {
				certFile, keyFile, err = resolveTLS(tlsCert, tlsKey)
				if err != nil {
					return err
				}
			}

			srv := api.New(cli.Paths, secret, static, queue)
			log.Info().Str("addr", addr).Bool("ui", static != nil).Bool("tls", useTLS).Msg("starting web server")
			return serveHTTP(cmd.Context(), addr, certFile, keyFile, srv.Handler())
		},
	}

	defaultRedis := os.Getenv("MACO_REDIS_URL")
	if defaultRedis == "" {
		defaultRedis = "redis://localhost:6379/0"
	}

	cmd.Flags().StringVar(&redisURL, "redis-url", defaultRedis, "Redis URL for background jobs")
	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address")
	cmd.Flags().StringVar(&devDir, "dev-dir", "", "serve static files from this directory instead of the embedded UI")
	cmd.Flags().BoolVar(&reconcile, "reconcile", true, "start autostart VMs and refresh the state cache on boot")
	cmd.Flags().BoolVar(&useTLS, "tls", true, "serve over HTTPS (use --tls=false for plain HTTP in dev)")
	cmd.Flags().StringVar(&tlsCert, "tls-cert", "", "TLS certificate to serve (default: self-signed, generated in the data dir)")
	cmd.Flags().StringVar(&tlsKey, "tls-key", "", "TLS private key matching --tls-cert")
	return cmd
}

func resolveTLS(tlsCert, tlsKey string) (string, string, error) {
	if (tlsCert == "") != (tlsKey == "") {
		return "", "", fmt.Errorf("--tls-cert and --tls-key must be provided together")
	}

	if tlsCert != "" {
		return tlsCert, tlsKey, nil
	}

	cert, key := cli.Paths.TLSCertPath(), cli.Paths.TLSKeyPath()
	if err := tlscert.EnsureSelfSigned(cert, key); err != nil {
		return "", "", err
	}

	return cert, key, nil
}

func resolveStaticFS(devDir string) fs.FS {
	if devDir != "" {
		return os.DirFS(devDir)
	}

	sub, err := fs.Sub(api.EmbeddedFS, "dist")
	if err != nil {
		return nil
	}

	entries, err := fs.ReadDir(sub, ".")
	if err != nil || len(entries) == 0 {
		return nil
	}

	return sub
}

func serveHTTP(parent context.Context, addr, certFile, keyFile string, handler http.Handler) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	done := make(chan error, 1)
	go listenHTTP(server, certFile, keyFile, done)

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}

		return <-done
	}
}

func listenHTTP(server *http.Server, certFile, keyFile string, done chan<- error) {
	var err error
	if certFile != "" {
		err = server.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = server.ListenAndServe()
	}

	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	done <- err
}
