package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"linktic/internal/debug"
	"linktic/internal/telemetry"
)

type Storage struct {
	log *slog.Logger
	db  *sql.DB
}

func NewStore(db *sql.DB, log *slog.Logger) (*Storage, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}

	if log == nil {
		return nil, errors.New("logger is required")
	}

	return &Storage{
		db:  db,
		log: log.With("component", "auth:store"),
	}, nil
}

func (s *Storage) createAccount(ctx context.Context, newAccount *account) (*account, error) {
	insertQuery := "INSERT INTO accounts(id, email, name, password_hash) VALUES(?, ?, ?, ?);"
	if 	debug.DebugAuthStore {
		s.log.Debug("creating_account", slog.String(telemetry.KeyQuery, insertQuery))
	}

	_, err := s.db.ExecContext(ctx, insertQuery, newAccount.ID, newAccount.Email, newAccount.Name, newAccount.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("store: inserting account: %w", err)
	}

	selectQuery := "SELECT id, email, name, deleted, created_at, updated_at FROM accounts WHERE id = ?;"
	if 	debug.DebugAuthStore {
		s.log.Debug("reading_created_account", slog.String(telemetry.KeyQuery, selectQuery))
	}

	createdAccount := &account{}
	err = s.db.QueryRowContext(ctx, selectQuery, newAccount.ID).Scan(
		&createdAccount.ID,
		&createdAccount.Email,
		&createdAccount.Name,
		&createdAccount.Deleted,
		&createdAccount.CreatedAt,
		&createdAccount.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting created account: %w", err)
	}

	return createdAccount, nil
}

func (s *Storage) getAccountByEmail(ctx context.Context, email string) (*account, error) {
	query := "SELECT id, email, name, password_hash, deleted, created_at, updated_at FROM accounts WHERE email = ? AND deleted = 0;"
	if 	debug.DebugAuthStore {
		s.log.Debug("querying_account_by_email", slog.String(telemetry.KeyQuery, query))
	}

	selectedAccount := &account{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&selectedAccount.ID,
		&selectedAccount.Email,
		&selectedAccount.Name,
		&selectedAccount.PasswordHash,
		&selectedAccount.Deleted,
		&selectedAccount.CreatedAt,
		&selectedAccount.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting account by email: %w", err)
	}

	return selectedAccount, nil
}

func (s *Storage) getAccountByID(ctx context.Context, id string) (*account, error) {
	query := "SELECT id, email, name, deleted, created_at, updated_at FROM accounts WHERE id = ? AND deleted = 0;"
	if 	debug.DebugAuthStore {
		s.log.Debug("querying_account_by_id", slog.String(telemetry.KeyQuery, query))
	}

	selectedAccount := &account{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&selectedAccount.ID,
		&selectedAccount.Email,
		&selectedAccount.Name,
		&selectedAccount.Deleted,
		&selectedAccount.CreatedAt,
		&selectedAccount.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting account by id: %w", err)
	}

	return selectedAccount, nil
}

func (s *Storage) getAccountByIDWithPassword(ctx context.Context, id string) (*account, error) {
	query := "SELECT id, email, name, password_hash, deleted, created_at, updated_at FROM accounts WHERE id = ? AND deleted = 0;"
	if 	debug.DebugAuthStore {
		s.log.Debug("querying_account_by_id_with_password", slog.String(telemetry.KeyQuery, query))
	}

	selectedAccount := &account{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&selectedAccount.ID,
		&selectedAccount.Email,
		&selectedAccount.Name,
		&selectedAccount.PasswordHash,
		&selectedAccount.Deleted,
		&selectedAccount.CreatedAt,
		&selectedAccount.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting account by id with password: %w", err)
	}

	return selectedAccount, nil
}

func (s *Storage) updateAccountByID(ctx context.Context, id string, input *updateAccountInput) (*account, error) {
	setStatements := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if input.Name != nil {
		setStatements = append(setStatements, "name = ?")
		args = append(args, *input.Name)
	}

	if input.Email != nil {
		setStatements = append(setStatements, "email = ?")
		args = append(args, *input.Email)
	}

	if input.PasswordHash != nil {
		setStatements = append(setStatements, "password_hash = ?")
		args = append(args, *input.PasswordHash)
	}

	if len(setStatements) == 0 {
		return nil, fmt.Errorf("store: updating account by id: %w", sql.ErrNoRows)
	}

	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = ? AND deleted = 0;", strings.Join(setStatements, ", "))
	args = append(args, id)

	if 	debug.DebugAuthStore {
		s.log.Debug("updating_account_by_id", slog.String(telemetry.KeyQuery, query))
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: updating account by id: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("store: retrieving updated row count: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("store: updating account by id: %w", sql.ErrNoRows)
	}

	updatedAccount, err := s.getAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return updatedAccount, nil
}
