package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/bcomnes/go-todo/internal/config"
	"github.com/bcomnes/go-todo/internal/database"
	"github.com/bcomnes/go-todo/internal/handlers"
	"github.com/bcomnes/go-todo/internal/middleware"
	"github.com/bcomnes/go-todo/internal/version"
)

var (
	host    = flag.String("host", "127.0.0.1", "Server host")
	port    = flag.String("port", "8080", "Server port")
	showVer = flag.Bool("version", false, "Show version and exit")
	showHelp = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVer {
		fmt.Printf("Service: go-todo\nCommit: %s\n", version.Get().Commit)
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("GET /", handlers.Root)

	// User routes
	mux.HandleFunc("POST /register", middleware.WithDB(db, handlers.RegisterUser))

	// Auth routes
	mux.HandleFunc("POST /login", middleware.WithDB(db, handlers.LoginUser))

	// Todo routes
	mux.HandleFunc("GET /todos", middleware.WithAuth(db, handlers.ListTodos))
	mux.HandleFunc("POST /todos", middleware.WithAuth(db, handlers.CreateTodo))
	mux.HandleFunc("GET /todos/{id}", middleware.WithAuth(db, handlers.GetTodo))
	mux.HandleFunc("PATCH /todos/{id}", middleware.WithAuth(db, handlers.UpdateTodo))
	mux.HandleFunc("DELETE /todos/{id}", middleware.WithAuth(db, handlers.DeleteTodo))

	addr := *host + ":" + *port
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		logger.Info("Starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("Shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Shutdown error", "error", err)
	}
	logger.Info("Server gracefully stopped")
}
