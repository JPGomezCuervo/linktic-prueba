package auth

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func newAuthTestStore(t *testing.T) *Storage {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open error: %v", err)
	}
	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`
		CREATE TABLE accounts(
			id TEXT NOT NULL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			deleted INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER DEFAULT (unixepoch()),
			updated_at INTEGER DEFAULT (unixepoch())
		) STRICT;
	`)
	if err != nil {
		t.Fatalf("create table error: %v", err)
	}

	store, err := NewStore(db, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewStore error: %v", err)
	}

	return store
}

func TestStorageCreateAccount_InsertsAccount(t *testing.T) {
	store := newAuthTestStore(t)

	created, err := store.createAccount(context.Background(), &account{
		ID:           "acc-1",
		Email:        "user@example.com",
		Name:         "User",
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("createAccount error: %v", err)
	}

	if created.ID != "acc-1" {
		t.Fatalf("expected ID acc-1, got %s", created.ID)
	}
}

func TestStorageGetAccountByEmail_ExcludesDeleted(t *testing.T) {
	store := newAuthTestStore(t)

	_, err := store.db.Exec(
		"INSERT INTO accounts(id, email, name, password_hash, deleted, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)",
		"acc-2", "deleted@example.com", "Deleted", "hash", 1, 1, 1,
	)
	if err != nil {
		t.Fatalf("insert account error: %v", err)
	}

	_, err = store.getAccountByEmail(context.Background(), "deleted@example.com")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestStorageGetAccountByIDWithPassword_ReturnsPasswordHash(t *testing.T) {
	store := newAuthTestStore(t)

	_, err := store.db.Exec(
		"INSERT INTO accounts(id, email, name, password_hash, deleted, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)",
		"acc-3", "user@example.com", "User", "hash-123", 0, 1, 1,
	)
	if err != nil {
		t.Fatalf("insert account error: %v", err)
	}

	account, err := store.getAccountByIDWithPassword(context.Background(), "acc-3")
	if err != nil {
		t.Fatalf("getAccountByIDWithPassword error: %v", err)
	}

	if account.PasswordHash != "hash-123" {
		t.Fatalf("expected password hash hash-123, got %s", account.PasswordHash)
	}
}

func TestStorageUpdateAccountByID_PartialUpdate(t *testing.T) {
	store := newAuthTestStore(t)

	_, err := store.db.Exec(
		"INSERT INTO accounts(id, email, name, password_hash, deleted, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)",
		"acc-4", "old@example.com", "Old Name", "old-hash", 0, 1, 1,
	)
	if err != nil {
		t.Fatalf("insert account error: %v", err)
	}

	newEmail := "new@example.com"
	updated, err := store.updateAccountByID(context.Background(), "acc-4", &updateAccountInput{Email: &newEmail})
	if err != nil {
		t.Fatalf("updateAccountByID error: %v", err)
	}

	if updated.Email != "new@example.com" {
		t.Fatalf("expected updated email, got %s", updated.Email)
	}

	if updated.Name != "Old Name" {
		t.Fatalf("expected unchanged name Old Name, got %s", updated.Name)
	}
}
