package inventory

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func newInventoryTestStore(t *testing.T) *Storage {
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
		CREATE TABLE items(
			id TEXT NOT NULL PRIMARY KEY,
			name TEXT NOT NULL,
			units INTEGER NOT NULL,
			price INTEGER NOT NULL,
			deleted INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		) STRICT;

		CREATE TABLE accounts(
			id TEXT NOT NULL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			deleted INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		) STRICT;

		CREATE TABLE item_audit_logs(
			id TEXT NOT NULL PRIMARY KEY,
			item_id TEXT NOT NULL,
			operation TEXT NOT NULL CHECK (operation IN ('create', 'update', 'delete', 'restore')),
			changes_json TEXT NOT NULL,
			actor_account_id TEXT NOT NULL,
			actor_name TEXT NOT NULL,
			actor_email TEXT NOT NULL,
			created_at INTEGER DEFAULT (unixepoch()),
			FOREIGN KEY (item_id) REFERENCES items(id),
			FOREIGN KEY (actor_account_id) REFERENCES accounts(id)
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

func insertAuditActor(t *testing.T, store *Storage, id string) {
	t.Helper()

	_, err := store.db.Exec(
		"INSERT INTO accounts(id, email, name, password_hash, deleted, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)",
		id, id+"@example.com", "Actor "+id, "hash", 0, 1, 1,
	)
	if err != nil {
		t.Fatalf("insert actor error: %v", err)
	}
}

func insertInventoryItem(t *testing.T, store *Storage, id string, name string, units int, price int, createdAt int, deleted int) {
	t.Helper()

	_, err := store.db.Exec(
		"INSERT INTO items(id, name, units, price, deleted, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)",
		id, name, units, price, deleted, createdAt, createdAt,
	)
	if err != nil {
		t.Fatalf("insert item error: %v", err)
	}
}

func TestStorageGetItems_CursorPagination(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-3", "item-3", 1, 100, 300, 0)
	insertInventoryItem(t, store, "item-2", "item-2", 1, 100, 200, 0)
	insertInventoryItem(t, store, "item-1", "item-1", 1, 100, 100, 0)

	firstPage, err := store.getItems(context.Background(), &inventoryQuery{Limit: 2, SortBy: "createdAt", SortOrder: "desc"})
	if err != nil {
		t.Fatalf("getItems first page error: %v", err)
	}

	if len(firstPage.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(firstPage.Items))
	}

	if !firstPage.HasNextPage {
		t.Fatalf("expected HasNextPage true")
	}

	if firstPage.HasPreviousPage {
		t.Fatalf("expected HasPreviousPage false")
	}

	if firstPage.Items[0].ID != "item-3" || firstPage.Items[1].ID != "item-2" {
		t.Fatalf("unexpected item order: %s, %s", firstPage.Items[0].ID, firstPage.Items[1].ID)
	}

	secondPage, err := store.getItems(context.Background(), &inventoryQuery{Limit: 2, After: firstPage.EndCursor, SortBy: "createdAt", SortOrder: "desc"})
	if err != nil {
		t.Fatalf("getItems second page error: %v", err)
	}

	if len(secondPage.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(secondPage.Items))
	}

	if secondPage.Items[0].ID != "item-1" {
		t.Fatalf("expected item-1, got %s", secondPage.Items[0].ID)
	}

	if secondPage.HasNextPage {
		t.Fatalf("expected HasNextPage false")
	}

	if !secondPage.HasPreviousPage {
		t.Fatalf("expected HasPreviousPage true")
	}
}

func TestStorageGetItems_InvalidCursor(t *testing.T) {
	store := newInventoryTestStore(t)

	_, err := store.getItems(context.Background(), &inventoryQuery{Limit: 10, After: "not-valid-cursor", SortBy: "createdAt", SortOrder: "desc"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestStorageGetItems_FilterAndSortByPriceAscending(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-a", "Desk Lamp", 5, 500, 100, 0)
	insertInventoryItem(t, store, "item-b", "Desk Chair", 7, 1500, 200, 0)
	insertInventoryItem(t, store, "item-c", "Keyboard", 12, 3000, 300, 0)

	page, err := store.getItems(context.Background(), &inventoryQuery{
		Limit:     10,
		Name:      "desk",
		UnitsMin:  intPtr(5),
		UnitsMax:  intPtr(8),
		PriceMin:  intPtr(400),
		PriceMax:  intPtr(2000),
		SortBy:    "price",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("getItems filtered query error: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}

	if page.Items[0].ID != "item-a" || page.Items[1].ID != "item-b" {
		t.Fatalf("unexpected sorted order: %s, %s", page.Items[0].ID, page.Items[1].ID)
	}
}

func TestStorageGetItems_FilterByNameCaseInsensitive(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-a", "Desk Lamp", 1, 100, 100, 0)
	insertInventoryItem(t, store, "item-b", "desk chair", 1, 200, 90, 0)
	insertInventoryItem(t, store, "item-c", "Keyboard", 1, 300, 80, 0)

	page, err := store.getItems(context.Background(), &inventoryQuery{
		Limit:     10,
		Name:      "DESK",
		SortBy:    "createdAt",
		SortOrder: "desc",
	})
	if err != nil {
		t.Fatalf("getItems filtered query error: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
}

func TestStorageGetItems_FilterByUnitsMinOnly(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-a", "A", 2, 100, 100, 0)
	insertInventoryItem(t, store, "item-b", "B", 5, 100, 90, 0)
	insertInventoryItem(t, store, "item-c", "C", 7, 100, 80, 0)

	page, err := store.getItems(context.Background(), &inventoryQuery{
		Limit:     10,
		UnitsMin:  intPtr(5),
		SortBy:    "createdAt",
		SortOrder: "desc",
	})
	if err != nil {
		t.Fatalf("getItems units min query error: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
}

func TestStorageGetItems_FilterByPriceExact(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-a", "A", 1, 500, 100, 0)
	insertInventoryItem(t, store, "item-b", "B", 1, 700, 90, 0)
	insertInventoryItem(t, store, "item-c", "C", 1, 500, 80, 0)

	page, err := store.getItems(context.Background(), &inventoryQuery{
		Limit:     10,
		PriceMin:  intPtr(500),
		PriceMax:  intPtr(500),
		SortBy:    "createdAt",
		SortOrder: "desc",
	})
	if err != nil {
		t.Fatalf("getItems exact price query error: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
}

func TestStorageGetItems_SortByNameAscending(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-z", "Zulu", 1, 100, 100, 0)
	insertInventoryItem(t, store, "item-a", "Alpha", 1, 100, 100, 0)
	insertInventoryItem(t, store, "item-m", "Mike", 1, 100, 100, 0)

	page, err := store.getItems(context.Background(), &inventoryQuery{Limit: 10, SortBy: "name", SortOrder: "asc"})
	if err != nil {
		t.Fatalf("getItems sort name asc error: %v", err)
	}

	if page.Items[0].Name != "Alpha" || page.Items[1].Name != "Mike" || page.Items[2].Name != "Zulu" {
		t.Fatalf("unexpected name order: %s, %s, %s", page.Items[0].Name, page.Items[1].Name, page.Items[2].Name)
	}
}

func TestStorageGetItems_SortByUnitsDesc(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-a", "A", 1, 100, 100, 0)
	insertInventoryItem(t, store, "item-b", "B", 7, 100, 100, 0)
	insertInventoryItem(t, store, "item-c", "C", 4, 100, 100, 0)

	page, err := store.getItems(context.Background(), &inventoryQuery{Limit: 10, SortBy: "units", SortOrder: "desc"})
	if err != nil {
		t.Fatalf("getItems sort units desc error: %v", err)
	}

	if page.Items[0].Units != 7 || page.Items[1].Units != 4 || page.Items[2].Units != 1 {
		t.Fatalf("unexpected units order: %d, %d, %d", page.Items[0].Units, page.Items[1].Units, page.Items[2].Units)
	}
}

func TestStorageGetItems_InvalidCursorWhenSortContextChanges(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-3", "C", 3, 300, 300, 0)
	insertInventoryItem(t, store, "item-2", "B", 2, 200, 200, 0)
	insertInventoryItem(t, store, "item-1", "A", 1, 100, 100, 0)

	firstPage, err := store.getItems(context.Background(), &inventoryQuery{Limit: 2, SortBy: "price", SortOrder: "desc"})
	if err != nil {
		t.Fatalf("getItems first page error: %v", err)
	}

	_, err = store.getItems(context.Background(), &inventoryQuery{Limit: 2, After: firstPage.EndCursor, SortBy: "name", SortOrder: "desc"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestStorageGetDeletedItems_ReturnsOnlyDeleted(t *testing.T) {
	store := newInventoryTestStore(t)

	insertInventoryItem(t, store, "item-a", "A", 1, 100, 100, 1)
	insertInventoryItem(t, store, "item-b", "B", 1, 100, 90, 0)

	page, err := store.getDeletedItems(context.Background(), &inventoryQuery{Limit: 10, SortBy: "createdAt", SortOrder: "desc"})
	if err != nil {
		t.Fatalf("getDeletedItems error: %v", err)
	}

	if len(page.Items) != 1 || page.Items[0].ID != "item-a" {
		t.Fatalf("expected deleted item item-a")
	}
}

func TestStorageRestoreItem_UpdatesDeletedFlagAndWritesAudit(t *testing.T) {
	store := newInventoryTestStore(t)
	insertAuditActor(t, store, "actor-1")
	insertInventoryItem(t, store, "item-a", "A", 1, 100, 100, 1)

	restored, err := store.restoreItem(context.Background(), "item-a", "actor-1")
	if err != nil {
		t.Fatalf("restoreItem error: %v", err)
	}

	if restored.Deleted {
		t.Fatalf("expected restored item deleted=false")
	}

	var operation string
	err = store.db.QueryRow("SELECT operation FROM item_audit_logs WHERE item_id = ? ORDER BY created_at DESC LIMIT 1", "item-a").Scan(&operation)
	if err != nil {
		t.Fatalf("select audit log error: %v", err)
	}

	if operation != "restore" {
		t.Fatalf("expected restore audit operation, got %s", operation)
	}
}

func intPtr(v int) *int {
	return &v
}

func TestStorageSoftDeleteItem_NotFound(t *testing.T) {
	store := newInventoryTestStore(t)

	err := store.softDeleteItem(context.Background(), "missing", "actor-1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
