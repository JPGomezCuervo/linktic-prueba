package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"linktic/internal/auth"
	"linktic/internal/common"
	"linktic/internal/debug"
	"linktic/internal/dotenv"
	"linktic/internal/inventory"
	"linktic/internal/middleware"
	"linktic/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var envPath string
	var loadedEnvCount int

	port := flag.Int("p", 8080, "Set Port number. Default is 8080")
	help := flag.Bool("h", false, "Help")
	useDefault := flag.Bool("d", false, "Set default .env path, dev-root/.env")
	customDotenvPath := flag.String("env", "", "Set specific .env path")

	flag.Usage = printUsage
	flag.Parse()

	if *help || flag.NFlag() == 0 {
		flag.Usage()
		return nil
	}

	{
		if *useDefault {
			wd := findProjectRoot()
			if wd == "" {
				return errors.New("unable to find project root for default .env path")
			}
			envPath = filepath.Join(wd, ".env")
		} else if *customDotenvPath != "" {
			if _, err := os.Stat(*customDotenvPath); err != nil {
				return errors.New("dotenv file doesn't exist")
			}
			envPath = *customDotenvPath
		} else {
			return errors.New("dotenv file doesn't exist")
		}

		envValues, err := dotenv.Load(envPath)
		if err != nil {
			return fmt.Errorf("failed to load dotenv file: %w", err)
		}
		loadedEnvCount = len(envValues)
	}

	if debug.DebugFlag {
		h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
		l := slog.New(&SourceOnlyOnErrorHandler{h})
		slog.SetDefault(l)
	} else {
		h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false,
		})
		l := slog.New(h)
		slog.SetDefault(l)
	}

	slog.Info("dotenv_loaded", slog.String("path", envPath), slog.Int("variables", loadedEnvCount))

	var database *sql.DB
	{
		dbPath := dotenv.GetEnv("DATABASE_PATH", "")
		if dbPath == "" {
			return errors.New("DATABASE_PATH is required")
		}

		absPath, err := filepath.Abs(dbPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		database, err = newDB(absPath)
		if err != nil {
			return fmt.Errorf("failed to connect to DB: %w", err)
		}
		defer database.Close()
	}

	var inventoryHandler *inventory.Handler
	var inventorySvc *inventory.Service
	var authHandler *auth.Handler
	jwtSecret := strings.TrimSpace(dotenv.GetEnv("JWT_SECRET", ""))
	if jwtSecret == "" {
		return errors.New("JWT_SECRET is required")
	}

	{
		store, err := inventory.NewStore(database, slog.Default())
		if err != nil {
			return fmt.Errorf("failed to create storage: %w", err)
		}

		inventorySvc, err = inventory.NewService(store, slog.Default())
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}

		inventoryHandler, err = inventory.NewHandler(inventorySvc, slog.Default())
		if err != nil {
			return fmt.Errorf("failed to create handler: %w", err)
		}

		authStore, err := auth.NewStore(database, slog.Default())
		if err != nil {
			return fmt.Errorf("failed to create auth storage: %w", err)
		}

		authSvc, err := auth.NewService(authStore, slog.Default(), jwtSecret, 24*time.Hour)
		if err != nil {
			return fmt.Errorf("failed to create auth service: %w", err)
		}

		authHandler, err = auth.NewHandler(authSvc, slog.Default())
		if err != nil {
			return fmt.Errorf("failed to create auth handler: %w", err)
		}
	}

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		err := common.EncodeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	})
	publicMux.HandleFunc("POST /auth/signup", authHandler.Signup)
	publicMux.HandleFunc("POST /auth/login", authHandler.Login)

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /auth/me", authHandler.Me)
	protectedMux.HandleFunc("PATCH /auth/me", authHandler.UpdateMe)
	protectedMux.HandleFunc("POST /auth/logout", authHandler.Logout)
	protectedMux.HandleFunc("GET /orders", inventoryHandler.GetOrders)
	protectedMux.HandleFunc("GET /inventory", inventoryHandler.GetInventory)
	protectedMux.HandleFunc("GET /inventory/deleted", inventoryHandler.GetDeletedInventory)
	protectedMux.HandleFunc("POST /inventory", inventoryHandler.CreateItem)
	protectedMux.HandleFunc("GET /inventory/{id}", inventoryHandler.GetItem)
	protectedMux.HandleFunc("PATCH /inventory/{id}", inventoryHandler.UpdateItem)
	protectedMux.HandleFunc("PATCH /inventory/{id}/restock", inventoryHandler.RestockItem)
	protectedMux.HandleFunc("PATCH /inventory/{id}/restore", inventoryHandler.RestoreItem)
	protectedMux.HandleFunc("DELETE /inventory/{id}", inventoryHandler.DeleteItem)

	rootMux := http.NewServeMux()
	rootMux.Handle("GET /health", publicMux)
	rootMux.Handle("POST /auth/signup", publicMux)
	rootMux.Handle("POST /auth/login", publicMux)
	rootMux.Handle("/", middleware.AuthMiddleware(protectedMux, jwtSecret))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      middleware.LoggingMiddleware(rootMux),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info(telemetry.OpServerStarted, slog.Int("port", *port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen_and_serve_failed: %w", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := inventorySvc.ProcessDueOrders(ctx)
			cancel()
			if err != nil {
				slog.Error("orders_due_processing_failed", slog.Any("error", err))
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case sig := <-quit:
		slog.Info("server_shutting_down", slog.String("signal", sig.String()))
	case err := <-serverErr:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server_forced_to_shutdown: %w", err)
	}

	slog.Info("server_exited_gracefully")
	return nil
}

func newDB(path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}

	dsn := fmt.Sprintf("file:%s?mode=rw&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	slog.Info("pinging_database", "path", path)
	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	slog.Info("database_connection_established")

	return db, nil
}

type SourceOnlyOnErrorHandler struct {
	slog.Handler
}

func (h *SourceOnlyOnErrorHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < slog.LevelError {
		r.PC = 0
	}
	return h.Handler.Handle(ctx, r)
}

func (h *SourceOnlyOnErrorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SourceOnlyOnErrorHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *SourceOnlyOnErrorHandler) WithGroup(name string) slog.Handler {
	return &SourceOnlyOnErrorHandler{Handler: h.Handler.WithGroup(name)}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", os.Args[0])

	fmt.Fprintln(os.Stderr, "Global Flags:")
	flag.PrintDefaults()

	fmt.Fprintln(os.Stderr, "\nExamples:")
	fmt.Fprintf(os.Stderr, "  %s -d\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -p 3000\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -env /custom/path/to/.env\n", os.Args[0])
}
func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
