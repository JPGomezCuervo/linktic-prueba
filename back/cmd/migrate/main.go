package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite"
)

type migrateLogger struct {
	verbose bool
}

func (l *migrateLogger) Printf(format string, v ...interface{}) {
	slog.Info("migrate: " + fmt.Sprintf(strings.TrimSpace(format), v...))
}

func (l *migrateLogger) Verbose() bool {
	return l.verbose
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command> [arguments]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Global Flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nCommands:")
	fmt.Fprintln(os.Stderr, "  up               Apply all pending 'up' migrations")
	fmt.Fprintln(os.Stderr, "  down             Apply all pending 'down' migrations")
	fmt.Fprintln(os.Stderr, "  force <version>  Force set the database version (does not run migrations)")
	fmt.Fprintln(os.Stderr, "  create <name>    Create a new blank migration file with a sequential prefix")
	fmt.Fprintln(os.Stderr, "\nExamples:")
	fmt.Fprintf(os.Stderr, "  %s -d up\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -db=/custom/path/to/db -m /custom/path/to/migrations/ create add_users_table\n", os.Args[0])
}

func main() {
	if err := cli(); err != nil {
		slog.Error("application failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func cli() error {
	var err error
	var migrationPath string
	var dbPath string

	useDefault := flag.Bool("d", false, "Set default database path and migrations directory")
	help := flag.Bool("h", false, "Print help")
	flag.StringVar(&dbPath, "db", "", "Set specific database directory")
	flag.StringVar(&migrationPath, "m", "", "Set specific migration directory")

	flag.Usage = printUsage
	flag.Parse()

	if *help || flag.NFlag() == 0 {
		flag.Usage()
		return nil
	}

	if *useDefault {
		wd := findProjectRoot()
		dbPath = filepath.Join(wd, "/db/linktic.db")
		migrationPath = filepath.Join(wd, "/migrations")
	}

	if dbPath == "" || migrationPath == "" {
		return errors.New("database and migration paths are required")
	}

	db, err := newDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database connection: %w", err)
	}

	shouldClose := true
	defer func() {
		if shouldClose {
			db.Close()
			slog.Info("Database closed")
		}
	}()

	slog.Info("Resolved database path", "migrations_path", fmt.Sprintf("'%s'", migrationPath))
	if _, err := os.Stat(migrationPath); err != nil {
		return errors.New("migration path doesn't exist")
	}

	m, err := setup(db, migrationPath)
	if err != nil {
		return err
	}
	defer m.Close()
	shouldClose = false

	args := flag.Args()
	if len(args) == 0 {
		return fmt.Errorf("Commands missing. For help: %s -help", os.Args[0])
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "up":
		slog.Info("Performing UP migrations...")
		startTime := time.Now()

		if err = m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				return fmt.Errorf("UP migration failed: %w", err)
			}
			slog.Info("Database is already up to date. Nothing to migrate.")
		} else {
			slog.Info("UP migrations completed successfully",
				"duration",
				time.Since(startTime).String())
		}

	case "down":
		slog.Info("Performing DOWN migrations...")
		startTime := time.Now()

		if err = m.Down(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				return fmt.Errorf("DOWN migration failed: %w", err)
			}
			slog.Info("No down migrations to apply. Nothing to migrate.")
		} else {
			slog.Info("DOWN migrations completed successfully",
				slog.String("duration", time.Since(startTime).String()))
		}

	case "force":
		if len(commandArgs) < 1 {
			return errors.New("Command FORCE requires a target version (e.g., 'force 1')")
		}

		v, err := strconv.Atoi(commandArgs[0])
		if err != nil {
			return fmt.Errorf("invalid version format; must be an integer: %w", err)
		}

		slog.Info("Forcing database to specific version", "target_version", v)
		if err = m.Force(v); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
		slog.Info("Successfully forced database version", "version", v)

	case "create":
		if len(commandArgs) < 1 {
			return errors.New("Command CREATE requires a name for the migration (e.g., 'create init_schema')")
		}
		name := commandArgs[0]
		dir := migrationPath
		seq := 1

		slog.Info("Scanning existing migrations to determine next sequence number", "directory", dir)
		files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
		if err != nil {
			return fmt.Errorf("failed to read migrations directory: %w", err)
		}

		if len(files) > 0 {
			base := filepath.Base(files[len(files)-1])
			idx := strings.Index(base, "_")
			if idx == -1 {
				return errors.New("existing migration file has invalid format, missing '_' separator")
			}

			next, err := strconv.Atoi(base[:idx])
			if err != nil {
				return fmt.Errorf("failed to parse sequence number from existing migration: %w", err)
			}
			seq = next + 1
		}

		// Add padding up to 6 digits
		version := fmt.Sprintf("%06d", seq)
		slog.Info("Creating new migration file", "version", version, "name", name)

		for _, direction := range []string{"up", "down"} {
			basename := fmt.Sprintf("%s_%s.%s.sql", version, name, direction)
			newFilePath := filepath.Join(dir, basename)
			slog.Info("Creating file:", slog.String("key_path", newFilePath))
			f, err := os.Create(newFilePath)
			if err != nil {
				return fmt.Errorf("failed to create migration file: %w", err)
			}
			f.Close()
		}

		slog.Info("Migration files created successfully")

	default:
		return fmt.Errorf("Unknown command: '%s. For help: %s -help'", command, os.Args[0])
	}

	return nil
}

func setup(db *sql.DB, migrationPath string) (*migrate.Migrate, error) {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite3 driver instance: %w", err)
	}

	absPath, err := filepath.Abs(migrationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for migrations: %w", err)
	}

	migrationPathURL := "file://" + filepath.ToSlash(absPath)

	m, err := migrate.NewWithDatabaseInstance(migrationPathURL, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrate instance: %w", err)
	}

	m.Log = &migrateLogger{verbose: true}
	return m, nil
}

func newDB(path string) (*sql.DB, error) {
	slog.Info("Resolved database path", "db_path", fmt.Sprintf("'%s'", path))

	dsn := fmt.Sprintf("file:%s?mode=rw&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open failed: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping failed: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	slog.Info("Database connection established successfully")
	return db, nil
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
