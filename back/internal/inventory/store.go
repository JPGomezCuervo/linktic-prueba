package inventory

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"

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
		log: log.With("component", "inventory:store"),
	}, nil
}

func (s *Storage) getItems(ctx context.Context, query *inventoryQuery) (*inventoryPage, error) {
	return s.getItemsByDeletedState(ctx, query, false)
}

func (s *Storage) getDeletedItems(ctx context.Context, query *inventoryQuery) (*inventoryPage, error) {
	return s.getItemsByDeletedState(ctx, query, true)
}

func (s *Storage) getItemsByDeletedState(ctx context.Context, query *inventoryQuery, deleted bool) (*inventoryPage, error) {
	if query == nil {
		return nil, fmt.Errorf("%w: query input is required", ErrInvalidPaginationInput)
	}

	if query.Limit <= 0 {
		return nil, fmt.Errorf("%w: items must be greater than zero", ErrInvalidPaginationInput)
	}

	sortColumn, err := sortColumnFromSortBy(query.SortBy)
	if err != nil {
		return nil, err
	}

	sortOrderSQL, err := sortOrderToSQL(query.SortOrder)
	if err != nil {
		return nil, err
	}

	deletedInt := 0
	if deleted {
		deletedInt = 1
	}

	whereClauses := []string{"deleted = ?"}
	queryArgs := make([]any, 0, 10)
	queryArgs = append(queryArgs, deletedInt)

	if query.Name != "" {
		whereClauses = append(whereClauses, "LOWER(name) LIKE ?")
		queryArgs = append(queryArgs, "%"+strings.ToLower(query.Name)+"%")
	}

	if query.UnitsMin != nil {
		whereClauses = append(whereClauses, "units >= ?")
		queryArgs = append(queryArgs, *query.UnitsMin)
	}

	if query.UnitsMax != nil {
		whereClauses = append(whereClauses, "units <= ?")
		queryArgs = append(queryArgs, *query.UnitsMax)
	}

	if query.PriceMin != nil {
		whereClauses = append(whereClauses, "price >= ?")
		queryArgs = append(queryArgs, *query.PriceMin)
	}

	if query.PriceMax != nil {
		whereClauses = append(whereClauses, "price <= ?")
		queryArgs = append(queryArgs, *query.PriceMax)
	}

	if query.After != "" {
		afterSortValue, afterID, err := decodeInventoryCursor(query.After, query.SortBy, query.SortOrder)
		if err != nil {
			return nil, err
		}

		sortValueArg, err := decodeCursorSortValue(query.SortBy, afterSortValue)
		if err != nil {
			return nil, err
		}

		comparison := ">"
		if sortOrderSQL == "DESC" {
			comparison = "<"
		}

		cursorPredicate := fmt.Sprintf("(%s %s ? OR (%s = ? AND id %s ?))", sortColumn, comparison, sortColumn, comparison)
		whereClauses = append(whereClauses, cursorPredicate)
		queryArgs = append(queryArgs, sortValueArg, sortValueArg, afterID)
	}

	querySQL := fmt.Sprintf(
		"SELECT id, name, units, price, deleted, created_at, updated_at FROM items WHERE %s ORDER BY %s %s, id %s LIMIT ?;",
		strings.Join(whereClauses, " AND "),
		sortColumn,
		sortOrderSQL,
		sortOrderSQL,
	)

	queryLimit := query.Limit + 1
	queryArgs = append(queryArgs, queryLimit)

	if 	debug.DebugInventoryStore {
		s.log.Debug("querying_inventory_items", slog.String(telemetry.KeyQuery, querySQL))
	}

	var items []*item

	rows, err := s.db.QueryContext(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var i item

		err := rows.Scan(&i.ID, &i.Name, &i.Units, &i.Price, &i.Deleted, &i.CreatedAt, &i.UpdatedAt)
		if err != nil {
			if 	debug.DebugInventoryStore {
				s.log.Debug(telemetry.ErrDBQuery,
					slog.String(telemetry.KeyTable, "items"),
					slog.String(telemetry.KeyQuery, querySQL),
					slog.Any(telemetry.KeyErr, err))
			}
			return nil, fmt.Errorf("store: querying items table: %w", err)
		}
		items = append(items, &i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if 	debug.DebugInventoryStore {
		s.log.Debug("inventory_items_retrieved", slog.Int("count", len(items)))
	}

	hasNextPage := len(items) > query.Limit
	if hasNextPage {
		items = items[:query.Limit]
	}

	startCursor := ""
	endCursor := ""
	if len(items) > 0 {
		startSortValue := cursorSortValueFromItem(items[0], query.SortBy)
		endSortValue := cursorSortValueFromItem(items[len(items)-1], query.SortBy)
		startCursor = encodeInventoryCursor(startSortValue, items[0].ID, query.SortBy, query.SortOrder)
		endCursor = encodeInventoryCursor(endSortValue, items[len(items)-1].ID, query.SortBy, query.SortOrder)
	}

	return &inventoryPage{
		Items:           items,
		HasNextPage:     hasNextPage,
		HasPreviousPage: query.After != "",
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}, nil
}

type inventoryCursorPayload struct {
	SortValue string `json:"sortValue"`
	ID        string `json:"id"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

func decodeInventoryCursor(cursor string, sortBy string, sortOrder string) (string, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", "", fmt.Errorf("%w: invalid cursor encoding", ErrInvalidPaginationInput)
	}

	var payload inventoryCursorPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return "", "", fmt.Errorf("%w: invalid cursor format", ErrInvalidPaginationInput)
	}

	payload.SortBy = strings.TrimSpace(payload.SortBy)
	payload.SortOrder = strings.TrimSpace(payload.SortOrder)
	if payload.SortBy != sortBy || payload.SortOrder != sortOrder {
		return "", "", fmt.Errorf("%w: cursor sorting does not match current query", ErrInvalidPaginationInput)
	}

	id := strings.TrimSpace(payload.ID)
	if id == "" {
		return "", "", fmt.Errorf("%w: invalid cursor item id", ErrInvalidPaginationInput)
	}

	sortValue := strings.TrimSpace(payload.SortValue)
	if sortValue == "" {
		return "", "", fmt.Errorf("%w: invalid cursor sort value", ErrInvalidPaginationInput)
	}

	return sortValue, id, nil
}

func decodeCursorSortValue(sortBy string, value string) (any, error) {
	switch sortBy {
	case "name":
		return value, nil
	case "units", "price", "createdAt":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid cursor sort value", ErrInvalidPaginationInput)
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("%w: invalid sortBy value", ErrInvalidPaginationInput)
	}
}

func sortColumnFromSortBy(sortBy string) (string, error) {
	switch sortBy {
	case "createdAt":
		return "created_at", nil
	case "name":
		return "name", nil
	case "units":
		return "units", nil
	case "price":
		return "price", nil
	default:
		return "", fmt.Errorf("%w: invalid sortBy value", ErrInvalidPaginationInput)
	}
}

func sortOrderToSQL(sortOrder string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(sortOrder)) {
	case "asc":
		return "ASC", nil
	case "desc":
		return "DESC", nil
	default:
		return "", fmt.Errorf("%w: invalid sortOrder value", ErrInvalidPaginationInput)
	}
}

func cursorSortValueFromItem(i *item, sortBy string) string {
	switch sortBy {
	case "name":
		return i.Name
	case "units":
		return strconv.Itoa(i.Units)
	case "price":
		return strconv.Itoa(i.Price)
	default:
		return strconv.Itoa(i.CreatedAt)
	}
}

func encodeInventoryCursor(sortValue string, id string, sortBy string, sortOrder string) string {
	payload, err := json.Marshal(inventoryCursorPayload{
		SortValue: sortValue,
		ID:        id,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
	if err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(payload)
}

func (s *Storage) createItem(ctx context.Context, newItem *item, actorAccountID string) (*item, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback()

	insertQuery := "INSERT INTO items(id, name, units, price) VALUES(?, ?, ?, ?);"
	if 	debug.DebugInventoryStore {
		s.log.Debug("creating_inventory_item", slog.String(telemetry.KeyQuery, insertQuery))
	}

	_, err = tx.ExecContext(ctx, insertQuery, newItem.ID, newItem.Name, newItem.Units, newItem.Price)
	if err != nil {
		return nil, fmt.Errorf("store: inserting item: %w", err)
	}

	createdItem, err := selectItemByIDWithTx(ctx, tx, newItem.ID)
	if err != nil {
		return nil, fmt.Errorf("store: selecting created item: %w", err)
	}

	changes := map[string]auditFieldChange{
		"name": {
			From: nil,
			To:   createdItem.Name,
		},
		"units": {
			From: nil,
			To:   createdItem.Units,
		},
		"price": {
			From: nil,
			To:   createdItem.Price,
		},
		"deleted": {
			From: nil,
			To:   createdItem.Deleted,
		},
	}

	if err := s.insertItemAuditLog(ctx, tx, createdItem.ID, "create", changes, actorAccountID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: committing create item transaction: %w", err)
	}

	if 	debug.DebugInventoryStore {
		s.log.Debug("inventory_item_created", slog.String("item_id", createdItem.ID))
	}

	return createdItem, nil
}

func (s *Storage) getItemByID(ctx context.Context, id string) (*item, error) {
	query := "SELECT id, name, units, price, deleted, created_at, updated_at FROM items WHERE id = ? AND deleted = 0;"
	if 	debug.DebugInventoryStore {
		s.log.Debug("querying_inventory_item_by_id", slog.String(telemetry.KeyQuery, query))
	}

	selectedItem := &item{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&selectedItem.ID,
		&selectedItem.Name,
		&selectedItem.Units,
		&selectedItem.Price,
		&selectedItem.Deleted,
		&selectedItem.CreatedAt,
		&selectedItem.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting item by id: %w", err)
	}

	return selectedItem, nil
}

func (s *Storage) getItemByIDIncludingDeleted(ctx context.Context, id string) (*item, error) {
	query := "SELECT id, name, units, price, deleted, created_at, updated_at FROM items WHERE id = ?;"
	if 	debug.DebugInventoryStore {
		s.log.Debug("querying_inventory_item_by_id_including_deleted", slog.String(telemetry.KeyQuery, query))
	}

	selectedItem := &item{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&selectedItem.ID,
		&selectedItem.Name,
		&selectedItem.Units,
		&selectedItem.Price,
		&selectedItem.Deleted,
		&selectedItem.CreatedAt,
		&selectedItem.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting item by id including deleted: %w", err)
	}

	return selectedItem, nil
}

func selectItemByIDWithTx(ctx context.Context, tx *sql.Tx, id string) (*item, error) {
	query := "SELECT id, name, units, price, deleted, created_at, updated_at FROM items WHERE id = ? AND deleted = 0;"

	selectedItem := &item{}
	err := tx.QueryRowContext(ctx, query, id).Scan(
		&selectedItem.ID,
		&selectedItem.Name,
		&selectedItem.Units,
		&selectedItem.Price,
		&selectedItem.Deleted,
		&selectedItem.CreatedAt,
		&selectedItem.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting item by id with tx: %w", err)
	}

	return selectedItem, nil
}

func selectDeletedItemByIDWithTx(ctx context.Context, tx *sql.Tx, id string) (*item, error) {
	query := "SELECT id, name, units, price, deleted, created_at, updated_at FROM items WHERE id = ? AND deleted = 1;"

	selectedItem := &item{}
	err := tx.QueryRowContext(ctx, query, id).Scan(
		&selectedItem.ID,
		&selectedItem.Name,
		&selectedItem.Units,
		&selectedItem.Price,
		&selectedItem.Deleted,
		&selectedItem.CreatedAt,
		&selectedItem.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting deleted item by id with tx: %w", err)
	}

	return selectedItem, nil
}

func selectAnyItemByIDWithTx(ctx context.Context, tx *sql.Tx, id string) (*item, error) {
	query := "SELECT id, name, units, price, deleted, created_at, updated_at FROM items WHERE id = ?;"

	selectedItem := &item{}
	err := tx.QueryRowContext(ctx, query, id).Scan(
		&selectedItem.ID,
		&selectedItem.Name,
		&selectedItem.Units,
		&selectedItem.Price,
		&selectedItem.Deleted,
		&selectedItem.CreatedAt,
		&selectedItem.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting any item by id with tx: %w", err)
	}

	return selectedItem, nil
}

func (s *Storage) getItemHistoryByID(ctx context.Context, id string) ([]*itemAuditEntry, error) {
	query := "SELECT id, item_id, operation, changes_json, actor_account_id, actor_name, actor_email, created_at FROM item_audit_logs WHERE item_id = ? ORDER BY created_at DESC, id DESC;"
	if 	debug.DebugInventoryStore {
		s.log.Debug("querying_item_audit_history", slog.String(telemetry.KeyQuery, query))
	}

	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("store: querying item audit logs: %w", err)
	}
	defer rows.Close()

	history := make([]*itemAuditEntry, 0)
	for rows.Next() {
		entry := &itemAuditEntry{}
		var rawChanges string

		err := rows.Scan(
			&entry.ID,
			&entry.ItemID,
			&entry.Operation,
			&rawChanges,
			&entry.ActorAccountID,
			&entry.ActorName,
			&entry.ActorEmail,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: scanning item audit log row: %w", err)
		}

		entry.Changes = map[string]auditFieldChange{}
		if rawChanges != "" {
			if err := json.Unmarshal([]byte(rawChanges), &entry.Changes); err != nil {
				return nil, fmt.Errorf("store: unmarshalling item audit changes: %w", err)
			}
		}

		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterating item audit logs rows: %w", err)
	}

	return history, nil
}

type auditActor struct {
	ID    string
	Name  string
	Email string
}

func (s *Storage) getAuditActorByAccountID(ctx context.Context, tx *sql.Tx, accountID string) (*auditActor, error) {
	query := "SELECT id, name, email FROM accounts WHERE id = ? AND deleted = 0;"

	actor := &auditActor{}
	err := tx.QueryRowContext(ctx, query, accountID).Scan(&actor.ID, &actor.Name, &actor.Email)
	if err != nil {
		return nil, fmt.Errorf("store: selecting actor by account id: %w", err)
	}

	return actor, nil
}

func (s *Storage) insertItemAuditLog(
	ctx context.Context,
	tx *sql.Tx,
	itemID string,
	operation string,
	changes map[string]auditFieldChange,
	actorAccountID string,
) error {
	rawChanges, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("store: marshalling item audit changes: %w", err)
	}

	actor, err := s.getAuditActorByAccountID(ctx, tx, actorAccountID)
	if err != nil {
		return err
	}

	insertQuery := "INSERT INTO item_audit_logs(id, item_id, operation, changes_json, actor_account_id, actor_name, actor_email) VALUES(?, ?, ?, ?, ?, ?, ?);"
	_, err = tx.ExecContext(
		ctx,
		insertQuery,
		uuid.NewString(),
		itemID,
		operation,
		string(rawChanges),
		actor.ID,
		actor.Name,
		actor.Email,
	)
	if err != nil {
		return fmt.Errorf("store: inserting item audit log: %w", err)
	}

	return nil
}

func (s *Storage) updateItem(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback()

	existingItem, err := selectItemByIDWithTx(ctx, tx, id)
	if err != nil {
		return nil, fmt.Errorf("store: selecting item by id before update: %w", err)
	}

	setStatements := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if input.Name != nil {
		setStatements = append(setStatements, "name = ?")
		args = append(args, *input.Name)
	}

	if input.Units != nil {
		setStatements = append(setStatements, "units = ?")
		args = append(args, *input.Units)
	}

	if input.Price != nil {
		setStatements = append(setStatements, "price = ?")
		args = append(args, *input.Price)
	}

	if len(setStatements) == 0 {
		return nil, fmt.Errorf("store: updating item: %w", sql.ErrNoRows)
	}

	query := fmt.Sprintf("UPDATE items SET %s WHERE id = ? AND deleted = 0;", strings.Join(setStatements, ", "))
	args = append(args, id)

	if 	debug.DebugInventoryStore {
		s.log.Debug("updating_inventory_item", slog.String(telemetry.KeyQuery, query))
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: updating item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("store: retrieving updated row count: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("store: updating item: %w", sql.ErrNoRows)
	}

	updatedItem, err := selectItemByIDWithTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	changes := make(map[string]auditFieldChange)
	if existingItem.Name != updatedItem.Name {
		changes["name"] = auditFieldChange{From: existingItem.Name, To: updatedItem.Name}
	}
	if existingItem.Units != updatedItem.Units {
		changes["units"] = auditFieldChange{From: existingItem.Units, To: updatedItem.Units}
	}
	if existingItem.Price != updatedItem.Price {
		changes["price"] = auditFieldChange{From: existingItem.Price, To: updatedItem.Price}
	}

	if err := s.insertItemAuditLog(ctx, tx, updatedItem.ID, "update", changes, actorAccountID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: committing update item transaction: %w", err)
	}

	return updatedItem, nil
}

func (s *Storage) softDeleteItem(ctx context.Context, id string, actorAccountID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback()

	existingItem, err := selectItemByIDWithTx(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("store: selecting item by id before delete: %w", err)
	}

	query := "UPDATE items SET deleted = 1 WHERE id = ? AND deleted = 0;"
	if 	debug.DebugInventoryStore {
		s.log.Debug("soft_deleting_inventory_item", slog.String(telemetry.KeyQuery, query))
	}

	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("store: deleting item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: retrieving deleted row count: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("store: deleting item: %w", sql.ErrNoRows)
	}

	changes := map[string]auditFieldChange{
		"deleted": {
			From: existingItem.Deleted,
			To:   true,
		},
	}

	if err := s.insertItemAuditLog(ctx, tx, id, "delete", changes, actorAccountID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: committing delete item transaction: %w", err)
	}

	return nil
}

func (s *Storage) restoreItem(ctx context.Context, id string, actorAccountID string) (*item, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback()

	existingItem, err := selectDeletedItemByIDWithTx(ctx, tx, id)
	if err != nil {
		return nil, fmt.Errorf("store: selecting deleted item by id before restore: %w", err)
	}

	query := "UPDATE items SET deleted = 0 WHERE id = ? AND deleted = 1;"
	if 	debug.DebugInventoryStore {
		s.log.Debug("restoring_inventory_item", slog.String(telemetry.KeyQuery, query))
	}

	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("store: restoring item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("store: retrieving restored row count: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("store: restoring item: %w", sql.ErrNoRows)
	}

	restoredItem, err := selectItemByIDWithTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	changes := map[string]auditFieldChange{
		"deleted": {
			From: existingItem.Deleted,
			To:   false,
		},
	}

	if err := s.insertItemAuditLog(ctx, tx, id, "restore", changes, actorAccountID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: committing restore item transaction: %w", err)
	}

	return restoredItem, nil
}

func encodeOrdersCursor(createdAt int, id string) string {
	rawCursor := fmt.Sprintf("%d:%s", createdAt, id)
	return base64.RawURLEncoding.EncodeToString([]byte(rawCursor))
}

func decodeOrdersCursor(cursor string) (int, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", fmt.Errorf("%w: invalid cursor encoding", ErrInvalidOrdersInput)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("%w: invalid cursor format", ErrInvalidOrdersInput)
	}

	createdAt, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("%w: invalid cursor timestamp", ErrInvalidOrdersInput)
	}

	id := strings.TrimSpace(parts[1])
	if id == "" {
		return 0, "", fmt.Errorf("%w: invalid cursor order id", ErrInvalidOrdersInput)
	}

	return createdAt, id, nil
}

func (s *Storage) createRestockOrder(
	ctx context.Context,
	itemID string,
	units int,
	paymentMethod string,
	deliveryAt int,
	actorAccountID string,
) (*order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: beginning transaction: %w", err)
	}
	defer tx.Rollback()

	selectedItem, err := selectItemByIDWithTx(ctx, tx, itemID)
	if err != nil {
		return nil, fmt.Errorf("store: selecting active item by id before restock: %w", err)
	}

	orderID := uuid.NewString()
	totalPrice := selectedItem.Price * units
	insertQuery := "INSERT INTO orders(id, account_id, item_id, units, unit_price, total_price, payment_method, status, delivery_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?);"
	_, err = tx.ExecContext(
		ctx,
		insertQuery,
		orderID,
		actorAccountID,
		itemID,
		units,
		selectedItem.Price,
		totalPrice,
		paymentMethod,
		"pending",
		deliveryAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: inserting restock order: %w", err)
	}

	createdOrder := &order{}
	selectQuery := `
		SELECT o.id, o.account_id, o.item_id, i.name, o.units, o.unit_price, o.total_price, o.payment_method, o.status, o.delivery_at, o.completed_at, o.created_at, o.updated_at
		FROM orders o
		JOIN items i ON i.id = o.item_id
		WHERE o.id = ?;
	`
	var completedAt sql.NullInt64
	err = tx.QueryRowContext(ctx, selectQuery, orderID).Scan(
		&createdOrder.ID,
		&createdOrder.AccountID,
		&createdOrder.ItemID,
		&createdOrder.ItemName,
		&createdOrder.Units,
		&createdOrder.UnitPrice,
		&createdOrder.TotalPrice,
		&createdOrder.PaymentMethod,
		&createdOrder.Status,
		&createdOrder.DeliveryAt,
		&completedAt,
		&createdOrder.CreatedAt,
		&createdOrder.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store: selecting created restock order: %w", err)
	}
	if completedAt.Valid {
		val := int(completedAt.Int64)
		createdOrder.CompletedAt = &val
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: committing create restock order transaction: %w", err)
	}

	return createdOrder, nil
}

func (s *Storage) getOrdersByAccountID(ctx context.Context, query *ordersQuery) (*ordersPage, error) {
	if query == nil {
		return nil, fmt.Errorf("%w: query input is required", ErrInvalidOrdersInput)
	}

	if query.Limit <= 0 {
		return nil, fmt.Errorf("%w: items must be greater than zero", ErrInvalidOrdersInput)
	}

	whereClauses := []string{"o.account_id = ?"}
	queryArgs := make([]any, 0, 4)
	queryArgs = append(queryArgs, query.AccountID)

	if query.After != "" {
		afterCreatedAt, afterID, err := decodeOrdersCursor(query.After)
		if err != nil {
			return nil, err
		}

		whereClauses = append(whereClauses, "(o.created_at < ? OR (o.created_at = ? AND o.id < ?))")
		queryArgs = append(queryArgs, afterCreatedAt, afterCreatedAt, afterID)
	}

	querySQL := fmt.Sprintf(`
		SELECT o.id, o.account_id, o.item_id, i.name, o.units, o.unit_price, o.total_price, o.payment_method, o.status, o.delivery_at, o.completed_at, o.created_at, o.updated_at
		FROM orders o
		JOIN items i ON i.id = o.item_id
		WHERE %s
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT ?;
	`, strings.Join(whereClauses, " AND "))

	queryLimit := query.Limit + 1
	queryArgs = append(queryArgs, queryLimit)

	rows, err := s.db.QueryContext(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("store: querying orders table: %w", err)
	}
	defer rows.Close()

	ordersList := make([]*order, 0, queryLimit)
	for rows.Next() {
		o := &order{}
		var completedAt sql.NullInt64
		err := rows.Scan(
			&o.ID,
			&o.AccountID,
			&o.ItemID,
			&o.ItemName,
			&o.Units,
			&o.UnitPrice,
			&o.TotalPrice,
			&o.PaymentMethod,
			&o.Status,
			&o.DeliveryAt,
			&completedAt,
			&o.CreatedAt,
			&o.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: scanning orders row: %w", err)
		}

		if completedAt.Valid {
			val := int(completedAt.Int64)
			o.CompletedAt = &val
		}

		ordersList = append(ordersList, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterating orders rows: %w", err)
	}

	hasNextPage := len(ordersList) > query.Limit
	if hasNextPage {
		ordersList = ordersList[:query.Limit]
	}

	startCursor := ""
	endCursor := ""
	if len(ordersList) > 0 {
		startCursor = encodeOrdersCursor(ordersList[0].CreatedAt, ordersList[0].ID)
		endCursor = encodeOrdersCursor(ordersList[len(ordersList)-1].CreatedAt, ordersList[len(ordersList)-1].ID)
	}

	return &ordersPage{
		Orders:          ordersList,
		HasNextPage:     hasNextPage,
		HasPreviousPage: query.After != "",
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}, nil
}

func (s *Storage) completeDueRestockOrders(ctx context.Context, nowUnix int) (int, error) {
	query := "SELECT id, item_id, account_id, units, unit_price, total_price, payment_method FROM orders WHERE status = 'pending' AND delivery_at <= ? ORDER BY delivery_at ASC, id ASC LIMIT 200;"
	rows, err := s.db.QueryContext(ctx, query, nowUnix)
	if err != nil {
		return 0, fmt.Errorf("store: querying due orders: %w", err)
	}
	defer rows.Close()

	type dueOrder struct {
		ID            string
		ItemID        string
		AccountID     string
		Units         int
		UnitPrice     int
		TotalPrice    int
		PaymentMethod string
	}

	dueOrders := make([]dueOrder, 0)
	for rows.Next() {
		var o dueOrder
		if err := rows.Scan(&o.ID, &o.ItemID, &o.AccountID, &o.Units, &o.UnitPrice, &o.TotalPrice, &o.PaymentMethod); err != nil {
			return 0, fmt.Errorf("store: scanning due order row: %w", err)
		}
		dueOrders = append(dueOrders, o)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("store: iterating due orders rows: %w", err)
	}

	processed := 0
	for _, due := range dueOrders {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return processed, fmt.Errorf("store: beginning due order transaction: %w", err)
		}

		committed := false
		func() {
			defer func() {
				if !committed {
					_ = tx.Rollback()
				}
			}()

			existingItem, err := selectAnyItemByIDWithTx(ctx, tx, due.ItemID)
			if err != nil {
				return
			}

			result, err := tx.ExecContext(ctx, "UPDATE orders SET status = 'completed', completed_at = ? WHERE id = ? AND status = 'pending';", nowUnix, due.ID)
			if err != nil {
				return
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil || rowsAffected == 0 {
				return
			}

			_, err = tx.ExecContext(ctx, "UPDATE items SET units = units + ? WHERE id = ?;", due.Units, due.ItemID)
			if err != nil {
				return
			}

			updatedItem, err := selectAnyItemByIDWithTx(ctx, tx, due.ItemID)
			if err != nil {
				return
			}

			changes := map[string]auditFieldChange{
				"units": {
					From: existingItem.Units,
					To:   updatedItem.Units,
				},
				"boughtUnits": {
					From: nil,
					To:   due.Units,
				},
				"unitPrice": {
					From: nil,
					To:   due.UnitPrice,
				},
				"totalPrice": {
					From: nil,
					To:   due.TotalPrice,
				},
				"paymentMethod": {
					From: nil,
					To:   due.PaymentMethod,
				},
				"orderId": {
					From: nil,
					To:   due.ID,
				},
			}

			if err := s.insertItemAuditLog(ctx, tx, due.ItemID, "restock", changes, due.AccountID); err != nil {
				return
			}

			if err := tx.Commit(); err != nil {
				return
			}

			committed = true
			processed++
		}()
	}

	return processed, nil
}
