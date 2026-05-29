package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"linktic/internal/debug"
	"linktic/internal/telemetry"
)

type storer interface {
	getItems(ctx context.Context, query *inventoryQuery) (*inventoryPage, error)
	getDeletedItems(ctx context.Context, query *inventoryQuery) (*inventoryPage, error)
	createItem(ctx context.Context, newItem *item, actorAccountID string) (*item, error)
	getItemByID(ctx context.Context, id string) (*item, error)
	getItemByIDIncludingDeleted(ctx context.Context, id string) (*item, error)
	getItemHistoryByID(ctx context.Context, id string) ([]*itemAuditEntry, error)
	updateItem(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error)
	softDeleteItem(ctx context.Context, id string, actorAccountID string) error
	restoreItem(ctx context.Context, id string, actorAccountID string) (*item, error)
	createRestockOrder(ctx context.Context, itemID string, units int, paymentMethod string, deliveryAt int, actorAccountID string) (*order, error)
	getOrdersByAccountID(ctx context.Context, query *ordersQuery) (*ordersPage, error)
	completeDueRestockOrders(ctx context.Context, nowUnix int) (int, error)
}

const (
	defaultSortBy    = "createdAt"
	defaultSortOrder = "desc"
)

var (
	ErrInvalidCreateItemInput = errors.New("invalid create item input")
	ErrInvalidItemID          = errors.New("invalid item id")
	ErrInvalidUpdateItemInput = errors.New("invalid update item input")
	ErrInvalidRestockInput    = errors.New("invalid restock input")
	ErrInvalidPaginationInput = errors.New("invalid pagination input")
	ErrInvalidOrdersInput     = errors.New("invalid orders input")
	ErrItemNotFound           = errors.New("item not found")
	ErrUnauthorized       = errors.New("unauthorized operation")
)

const (
	paymentMethodCreditCard    = "credit_card"
	paymentMethodCheckingAccount = "checking_account"
)

type Service struct {
	log   *slog.Logger
	store storer
}

func NewService(store storer, log *slog.Logger) (*Service, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}

	if log == nil {
		return nil, errors.New("logger is required")
	}

	return &Service{
		log:   log.With("component", "inventory:service"),
		store: store,
	}, nil
}

func (s *Service) getInventory(ctx context.Context, input *getInventoryInput) (*inventory, error) {
	query, err := parseInventoryQuery(input)
	if err != nil {
		return nil, err
	}

	page, err := s.store.getItems(ctx, query)
	if err != nil {
		if errors.Is(err, ErrInvalidPaginationInput) {
			return nil, err
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "items"))
		}
		return nil, fmt.Errorf("service: retrieving items: %w", err)
	}

	if 	debug.DebugInventoryService {
		s.log.Debug("items_retrieved", slog.Int("count", len(page.Items)))
	}

	return &inventory{
		Items:           page.Items,
		HasNextPage:     page.HasNextPage,
		HasPreviousPage: page.HasPreviousPage,
		StartCursor:     page.StartCursor,
		EndCursor:       page.EndCursor,
	}, nil
}

func (s *Service) getDeletedInventory(ctx context.Context, input *getInventoryInput) (*inventory, error) {
	query, err := parseInventoryQuery(input)
	if err != nil {
		return nil, err
	}

	page, err := s.store.getDeletedItems(ctx, query)
	if err != nil {
		if errors.Is(err, ErrInvalidPaginationInput) {
			return nil, err
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "items"))
		}
		return nil, fmt.Errorf("service: retrieving deleted items: %w", err)
	}

	if 	debug.DebugInventoryService {
		s.log.Debug("deleted_items_retrieved", slog.Int("count", len(page.Items)))
	}

	return &inventory{
		Items:           page.Items,
		HasNextPage:     page.HasNextPage,
		HasPreviousPage: page.HasPreviousPage,
		StartCursor:     page.StartCursor,
		EndCursor:       page.EndCursor,
	}, nil
}

func parseInventoryQuery(input *getInventoryInput) (*inventoryQuery, error) {
	query := &inventoryQuery{
		Limit:     10,
		After:     "",
		Name:      "",
		SortBy:    defaultSortBy,
		SortOrder: defaultSortOrder,
	}

	if input != nil {
		query.After = strings.TrimSpace(input.After)
		query.Name = strings.TrimSpace(input.Name)

		items := strings.TrimSpace(input.Items)
		if items != "" {
			parsedItems, err := strconv.Atoi(items)
			if err != nil {
				return nil, fmt.Errorf("%w: items query param must be a number", ErrInvalidPaginationInput)
			}

			switch parsedItems {
			case 10, 20, 50:
				query.Limit = parsedItems
			default:
				return nil, fmt.Errorf("%w: items query param must be one of 10, 20 or 50", ErrInvalidPaginationInput)
			}
		}

		unitsMin, err := parseOptionalNonNegativeInt(input.UnitsMin, "unitsMin")
		if err != nil {
			return nil, err
		}
		query.UnitsMin = unitsMin

		unitsMax, err := parseOptionalNonNegativeInt(input.UnitsMax, "unitsMax")
		if err != nil {
			return nil, err
		}
		query.UnitsMax = unitsMax

		if query.UnitsMin != nil && query.UnitsMax != nil && *query.UnitsMin > *query.UnitsMax {
			return nil, fmt.Errorf("%w: unitsMin must be less than or equal to unitsMax", ErrInvalidPaginationInput)
		}

		priceMin, err := parseOptionalNonNegativeInt(input.PriceMin, "priceMin")
		if err != nil {
			return nil, err
		}
		query.PriceMin = priceMin

		priceMax, err := parseOptionalNonNegativeInt(input.PriceMax, "priceMax")
		if err != nil {
			return nil, err
		}
		query.PriceMax = priceMax

		if query.PriceMin != nil && query.PriceMax != nil && *query.PriceMin > *query.PriceMax {
			return nil, fmt.Errorf("%w: priceMin must be less than or equal to priceMax", ErrInvalidPaginationInput)
		}

		sortBy := strings.TrimSpace(input.SortBy)
		if sortBy != "" {
			switch sortBy {
			case "name", "units", "price", "createdAt":
				query.SortBy = sortBy
			default:
				return nil, fmt.Errorf("%w: invalid sortBy value", ErrInvalidPaginationInput)
			}
		}

		sortOrder := strings.TrimSpace(strings.ToLower(input.SortOrder))
		if sortOrder != "" {
			switch sortOrder {
			case "asc", "desc":
				query.SortOrder = sortOrder
			default:
				return nil, fmt.Errorf("%w: invalid sortOrder value", ErrInvalidPaginationInput)
			}
		}
	}

	return query, nil
}

func parseOptionalNonNegativeInt(raw string, field string) (*int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s query param must be a number", ErrInvalidPaginationInput, field)
	}

	if parsed < 0 {
		return nil, fmt.Errorf("%w: %s query param must be greater than or equal to 0", ErrInvalidPaginationInput, field)
	}

	return &parsed, nil
}

func (s *Service) createItem(ctx context.Context, input *createItemInput, actorAccountID string) (*item, error) {
	if strings.TrimSpace(actorAccountID) == "" {
		return nil, fmt.Errorf("%w: actor id is required", ErrUnauthorized)
	}

	if input == nil {
		return nil, fmt.Errorf("%w: payload is required", ErrInvalidCreateItemInput)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidCreateItemInput)
	}

	if input.Units < 0 {
		return nil, fmt.Errorf("%w: units must be greater than or equal to 0", ErrInvalidCreateItemInput)
	}

	if input.Price < 0 {
		return nil, fmt.Errorf("%w: price must be greater than or equal to 0", ErrInvalidCreateItemInput)
	}

	newItem := &item{
		ID:    uuid.NewString(),
		Name:  name,
		Units: input.Units,
		Price: input.Price,
	}

	createdItem, err := s.store.createItem(ctx, newItem, strings.TrimSpace(actorAccountID))
	if err != nil {
		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "items"))
		}
		return nil, fmt.Errorf("service: creating item: %w", err)
	}

	if 	debug.DebugInventoryService {
		s.log.Debug("item_created", slog.String("item_id", createdItem.ID))
	}

	return createdItem, nil
}

func (s *Service) getItem(ctx context.Context, id string) (*itemDetails, error) {
	itemID := strings.TrimSpace(id)
	if itemID == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidItemID)
	}

	selectedItem, err := s.store.getItemByIDIncludingDeleted(ctx, itemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %s", ErrItemNotFound, itemID)
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "items"))
		}
		return nil, fmt.Errorf("service: retrieving item by id: %w", err)
	}

	history, err := s.store.getItemHistoryByID(ctx, itemID)
	if err != nil {
		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "item_audit_logs"))
		}
		return nil, fmt.Errorf("service: retrieving item audit history: %w", err)
	}

	return &itemDetails{
		Item:    selectedItem,
		History: history,
	}, nil
}

func (s *Service) updateItem(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error) {
	if strings.TrimSpace(actorAccountID) == "" {
		return nil, fmt.Errorf("%w: actor id is required", ErrUnauthorized)
	}

	itemID := strings.TrimSpace(id)
	if itemID == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidItemID)
	}

	if input == nil {
		return nil, fmt.Errorf("%w: payload is required", ErrInvalidUpdateItemInput)
	}

	if input.Name == nil && input.Units == nil && input.Price == nil {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidUpdateItemInput)
	}

	validated := &updateItemInput{}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", ErrInvalidUpdateItemInput)
		}
		validated.Name = &name
	}

	if input.Units != nil {
		if *input.Units < 0 {
			return nil, fmt.Errorf("%w: units must be greater than or equal to 0", ErrInvalidUpdateItemInput)
		}
		units := *input.Units
		validated.Units = &units
	}

	if input.Price != nil {
		if *input.Price < 0 {
			return nil, fmt.Errorf("%w: price must be greater than or equal to 0", ErrInvalidUpdateItemInput)
		}
		price := *input.Price
		validated.Price = &price
	}

	updatedItem, err := s.store.updateItem(ctx, itemID, validated, strings.TrimSpace(actorAccountID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %s", ErrItemNotFound, itemID)
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "items"))
		}
		return nil, fmt.Errorf("service: updating item: %w", err)
	}

	if 	debug.DebugInventoryService {
		s.log.Debug("item_updated", slog.String("item_id", updatedItem.ID))
	}

	return updatedItem, nil
}

func (s *Service) deleteItem(ctx context.Context, id string, actorAccountID string) error {
	if strings.TrimSpace(actorAccountID) == "" {
		return fmt.Errorf("%w: actor id is required", ErrUnauthorized)
	}

	itemID := strings.TrimSpace(id)
	if itemID == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidItemID)
	}

	err := s.store.softDeleteItem(ctx, itemID, strings.TrimSpace(actorAccountID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: id %s", ErrItemNotFound, itemID)
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "items"))
		}
		return fmt.Errorf("service: deleting item: %w", err)
	}

	if 	debug.DebugInventoryService {
		s.log.Debug("item_deleted", slog.String("item_id", itemID))
	}

	return nil
}

func (s *Service) restoreItem(ctx context.Context, id string, actorAccountID string) (*item, error) {
	if strings.TrimSpace(actorAccountID) == "" {
		return nil, fmt.Errorf("%w: actor id is required", ErrUnauthorized)
	}

	itemID := strings.TrimSpace(id)
	if itemID == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidItemID)
	}

	restoredItem, err := s.store.restoreItem(ctx, itemID, strings.TrimSpace(actorAccountID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %s", ErrItemNotFound, itemID)
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "items"))
		}
		return nil, fmt.Errorf("service: restoring item: %w", err)
	}

	if 	debug.DebugInventoryService {
		s.log.Debug("item_restored", slog.String("item_id", restoredItem.ID))
	}

	return restoredItem, nil
}

func parseOrdersQuery(input *getOrdersInput, accountID string) (*ordersQuery, error) {
	query := &ordersQuery{
		Limit:     10,
		After:     "",
		AccountID: strings.TrimSpace(accountID),
	}

	if query.AccountID == "" {
		return nil, ErrUnauthorized
	}

	if input == nil {
		return query, nil
	}

	query.After = strings.TrimSpace(input.After)
	items := strings.TrimSpace(input.Items)
	if items == "" {
		return query, nil
	}

	parsedItems, err := strconv.Atoi(items)
	if err != nil {
		return nil, fmt.Errorf("%w: items query param must be a number", ErrInvalidOrdersInput)
	}

	switch parsedItems {
	case 10, 20, 50:
		query.Limit = parsedItems
	default:
		return nil, fmt.Errorf("%w: items query param must be one of 10, 20 or 50", ErrInvalidOrdersInput)
	}

	return query, nil
}

func (s *Service) restockItem(ctx context.Context, id string, input *restockItemInput, actorAccountID string) (*restockResult, error) {
	if strings.TrimSpace(actorAccountID) == "" {
		return nil, fmt.Errorf("%w: actor id is required", ErrUnauthorized)
	}

	itemID := strings.TrimSpace(id)
	if itemID == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidItemID)
	}

	if input == nil {
		return nil, fmt.Errorf("%w: payload is required", ErrInvalidRestockInput)
	}

	if input.Units <= 0 {
		return nil, fmt.Errorf("%w: units must be greater than 0", ErrInvalidRestockInput)
	}

	paymentMethod := strings.TrimSpace(input.PaymentMethod)
	if paymentMethod != paymentMethodCreditCard && paymentMethod != paymentMethodCheckingAccount {
		return nil, fmt.Errorf("%w: payment method is invalid", ErrInvalidRestockInput)
	}

	deliveryMinutes := rand.Intn(8) + 1
	deliveryAt := int(time.Now().UTC().Add(time.Duration(deliveryMinutes) * time.Minute).Unix())

	createdOrder, err := s.store.createRestockOrder(ctx, itemID, input.Units, paymentMethod, deliveryAt, strings.TrimSpace(actorAccountID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %s", ErrItemNotFound, itemID)
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "orders"))
		}

		return nil, fmt.Errorf("service: creating restock order: %w", err)
	}

	nowUnix := int(time.Now().UTC().Unix())
	remainingSeconds := createdOrder.DeliveryAt - nowUnix
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}
	createdOrder.RemainingSeconds = remainingSeconds

	return &restockResult{
		Order:           createdOrder,
		DeliverySeconds: deliveryMinutes * 60,
	}, nil
}

func (s *Service) getOrders(ctx context.Context, accountID string, input *getOrdersInput) (*orders, error) {
	if _, err := s.store.completeDueRestockOrders(ctx, int(time.Now().UTC().Unix())); err != nil {
		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "orders"))
		}
		return nil, fmt.Errorf("service: processing due orders before list: %w", err)
	}

	query, err := parseOrdersQuery(input, accountID)
	if err != nil {
		return nil, err
	}

	page, err := s.store.getOrdersByAccountID(ctx, query)
	if err != nil {
		if errors.Is(err, ErrInvalidOrdersInput) || errors.Is(err, ErrInvalidPaginationInput) {
			return nil, err
		}

		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "orders"))
		}
		return nil, fmt.Errorf("service: retrieving account orders: %w", err)
	}

	nowUnix := int(time.Now().UTC().Unix())
	for _, o := range page.Orders {
		if o.Status != "pending" {
			o.RemainingSeconds = 0
			continue
		}

		remaining := o.DeliveryAt - nowUnix
		if remaining < 0 {
			remaining = 0
		}
		o.RemainingSeconds = remaining
	}

	return &orders{
		Orders:          page.Orders,
		HasNextPage:     page.HasNextPage,
		HasPreviousPage: page.HasPreviousPage,
		StartCursor:     page.StartCursor,
		EndCursor:       page.EndCursor,
	}, nil
}

func (s *Service) ProcessDueOrders(ctx context.Context) (int, error) {
	processed, err := s.store.completeDueRestockOrders(ctx, int(time.Now().UTC().Unix()))
	if err != nil {
		if 	debug.DebugInventoryService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "orders"))
		}
		return 0, fmt.Errorf("service: processing due orders: %w", err)
	}

	return processed, nil
}
