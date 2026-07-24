// Command server runs go-todo's server-rendered web application and JSON API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bcomnes/go-todo/pkg/config"
	"github.com/bcomnes/go-todo/pkg/database"
	"github.com/bcomnes/go-todo/pkg/httpapi"
	"github.com/bcomnes/go-todo/pkg/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	hostFlag := flag.String("host", "", "server host override")
	portFlag := flag.String("port", "", "server port override")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		info := version.Get()
		fmt.Printf("Service: %s\nCommit: %s\n", info.Service, info.Commit)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if *hostFlag != "" {
		cfg.Host = *hostFlag
	}
	if *portFlag != "" {
		cfg.Port = *portFlag
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	api, err := httpapi.NewWithOptions(db, cfg.TokenTTL, httpapi.Options{
		AllowInsecureCookies: !cfg.SecureCookie,
	})
	if err != nil {
		return fmt.Errorf("initialize API: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.Host + ":" + cfg.Port,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("starting server", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	logger.Info("server stopped")
	return nil
}
