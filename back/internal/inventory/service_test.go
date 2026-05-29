package inventory

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

type serviceStoreStub struct {
	getItemsFn      func(ctx context.Context, query *inventoryQuery) (*inventoryPage, error)
	getDeletedItemsFn func(ctx context.Context, query *inventoryQuery) (*inventoryPage, error)
	createItemFn    func(ctx context.Context, newItem *item, actorAccountID string) (*item, error)
	getItemByIDFn   func(ctx context.Context, id string) (*item, error)
	getItemByIDIncludingDeletedFn func(ctx context.Context, id string) (*item, error)
	getItemHistoryFn func(ctx context.Context, id string) ([]*itemAuditEntry, error)
	updateItemFn    func(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error)
	softDeleteFn    func(ctx context.Context, id string, actorAccountID string) error
	restoreItemFn   func(ctx context.Context, id string, actorAccountID string) (*item, error)
	createRestockOrderFn func(ctx context.Context, itemID string, units int, paymentMethod string, deliveryAt int, actorAccountID string) (*order, error)
	getOrdersByAccountIDFn func(ctx context.Context, query *ordersQuery) (*ordersPage, error)
	completeDueRestockOrdersFn func(ctx context.Context, nowUnix int) (int, error)
	receivedNewItem *item
}

func (s *serviceStoreStub) getItems(ctx context.Context, query *inventoryQuery) (*inventoryPage, error) {
	return s.getItemsFn(ctx, query)
}

func (s *serviceStoreStub) getDeletedItems(ctx context.Context, query *inventoryQuery) (*inventoryPage, error) {
	if s.getDeletedItemsFn == nil {
		return &inventoryPage{}, nil
	}
	return s.getDeletedItemsFn(ctx, query)
}

func (s *serviceStoreStub) createItem(ctx context.Context, newItem *item, actorAccountID string) (*item, error) {
	s.receivedNewItem = newItem
	return s.createItemFn(ctx, newItem, actorAccountID)
}

func (s *serviceStoreStub) getItemByID(ctx context.Context, id string) (*item, error) {
	return s.getItemByIDFn(ctx, id)
}

func (s *serviceStoreStub) getItemByIDIncludingDeleted(ctx context.Context, id string) (*item, error) {
	if s.getItemByIDIncludingDeletedFn != nil {
		return s.getItemByIDIncludingDeletedFn(ctx, id)
	}
	return s.getItemByIDFn(ctx, id)
}

func (s *serviceStoreStub) getItemHistoryByID(ctx context.Context, id string) ([]*itemAuditEntry, error) {
	return s.getItemHistoryFn(ctx, id)
}

func (s *serviceStoreStub) updateItem(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error) {
	return s.updateItemFn(ctx, id, input, actorAccountID)
}

func (s *serviceStoreStub) softDeleteItem(ctx context.Context, id string, actorAccountID string) error {
	return s.softDeleteFn(ctx, id, actorAccountID)
}

func (s *serviceStoreStub) restoreItem(ctx context.Context, id string, actorAccountID string) (*item, error) {
	if s.restoreItemFn == nil {
		return nil, sql.ErrNoRows
	}
	return s.restoreItemFn(ctx, id, actorAccountID)
}

func (s *serviceStoreStub) createRestockOrder(ctx context.Context, itemID string, units int, paymentMethod string, deliveryAt int, actorAccountID string) (*order, error) {
	if s.createRestockOrderFn == nil {
		return nil, sql.ErrNoRows
	}
	return s.createRestockOrderFn(ctx, itemID, units, paymentMethod, deliveryAt, actorAccountID)
}

func (s *serviceStoreStub) getOrdersByAccountID(ctx context.Context, query *ordersQuery) (*ordersPage, error) {
	if s.getOrdersByAccountIDFn == nil {
		return &ordersPage{}, nil
	}
	return s.getOrdersByAccountIDFn(ctx, query)
}

func (s *serviceStoreStub) completeDueRestockOrders(ctx context.Context, nowUnix int) (int, error) {
	if s.completeDueRestockOrdersFn == nil {
		return 0, nil
	}
	return s.completeDueRestockOrdersFn(ctx, nowUnix)
}

func testServiceLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServiceGetInventory_DefaultItemsValue(t *testing.T) {
	stub := &serviceStoreStub{
		getItemsFn: func(_ context.Context, query *inventoryQuery) (*inventoryPage, error) {
			if query.Limit != 10 {
				t.Fatalf("expected default limit 10, got %d", query.Limit)
			}
			if query.After != "" {
				t.Fatalf("expected empty after cursor, got %q", query.After)
			}
			if query.SortBy != "createdAt" || query.SortOrder != "desc" {
				t.Fatalf("unexpected default sort %s/%s", query.SortBy, query.SortOrder)
			}
			return &inventoryPage{}, nil
		},
		createItemFn: func(context.Context, *item, string) (*item, error) { return nil, nil },
		getItemByIDFn: func(context.Context, string) (*item, error) {
			return nil, nil
		},
		getItemHistoryFn: func(context.Context, string) ([]*itemAuditEntry, error) { return nil, nil },
		updateItemFn: func(context.Context, string, *updateItemInput, string) (*item, error) { return nil, nil },
		softDeleteFn: func(context.Context, string, string) error { return nil },
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{})
	if err != nil {
		t.Fatalf("getInventory returned error: %v", err)
	}
}

func TestServiceGetInventory_InvalidItemsValue(t *testing.T) {
	stub := &serviceStoreStub{
		getItemsFn: func(_ context.Context, _ *inventoryQuery) (*inventoryPage, error) {
			return &inventoryPage{}, nil
		},
	}
	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{Items: "15"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestServiceGetInventory_InvalidSortByValue(t *testing.T) {
	stub := &serviceStoreStub{
		getItemsFn: func(_ context.Context, _ *inventoryQuery) (*inventoryPage, error) {
			return &inventoryPage{}, nil
		},
	}
	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{SortBy: "unknown"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestServiceGetInventory_ParsesFilterRanges(t *testing.T) {
	stub := &serviceStoreStub{
		getItemsFn: func(_ context.Context, query *inventoryQuery) (*inventoryPage, error) {
			if query.Name != "desk" {
				t.Fatalf("expected name filter desk, got %q", query.Name)
			}
			if query.UnitsMin == nil || *query.UnitsMin != 1 {
				t.Fatalf("expected unitsMin 1")
			}
			if query.UnitsMax == nil || *query.UnitsMax != 10 {
				t.Fatalf("expected unitsMax 10")
			}
			if query.PriceMin == nil || *query.PriceMin != 200 {
				t.Fatalf("expected priceMin 200")
			}
			if query.PriceMax == nil || *query.PriceMax != 1000 {
				t.Fatalf("expected priceMax 1000")
			}
			if query.SortBy != "price" || query.SortOrder != "asc" {
				t.Fatalf("unexpected sort %s/%s", query.SortBy, query.SortOrder)
			}
			return &inventoryPage{}, nil
		},
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{
		Name:      "desk",
		UnitsMin:  "1",
		UnitsMax:  "10",
		PriceMin:  "200",
		PriceMax:  "1000",
		SortBy:    "price",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("getInventory returned error: %v", err)
	}
}

func TestServiceGetInventory_InvalidSortOrderValue(t *testing.T) {
	stub := &serviceStoreStub{
		getItemsFn: func(_ context.Context, _ *inventoryQuery) (*inventoryPage, error) {
			return &inventoryPage{}, nil
		},
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{SortOrder: "sideways"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestServiceGetInventory_InvalidUnitsRange(t *testing.T) {
	stub := &serviceStoreStub{}
	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{UnitsMin: "11", UnitsMax: "10"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestServiceGetInventory_InvalidPriceRange(t *testing.T) {
	stub := &serviceStoreStub{}
	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{PriceMin: "5000", PriceMax: "1000"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestServiceGetInventory_InvalidUnitsMinNonNumeric(t *testing.T) {
	stub := &serviceStoreStub{}
	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{UnitsMin: "abc"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestServiceGetInventory_InvalidPriceMaxNegative(t *testing.T) {
	stub := &serviceStoreStub{}
	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{PriceMax: "-1"})
	if !errors.Is(err, ErrInvalidPaginationInput) {
		t.Fatalf("expected ErrInvalidPaginationInput, got %v", err)
	}
}

func TestServiceGetInventory_DefaultSortOrderWithExplicitSortBy(t *testing.T) {
	stub := &serviceStoreStub{
		getItemsFn: func(_ context.Context, query *inventoryQuery) (*inventoryPage, error) {
			if query.SortBy != "name" {
				t.Fatalf("expected sortBy name, got %s", query.SortBy)
			}
			if query.SortOrder != "desc" {
				t.Fatalf("expected default sort order desc, got %s", query.SortOrder)
			}
			return &inventoryPage{}, nil
		},
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.getInventory(context.Background(), &getInventoryInput{SortBy: "name"})
	if err != nil {
		t.Fatalf("getInventory returned error: %v", err)
	}
}

func TestServiceCreateItem_GeneratesUUIDAndTrimsName(t *testing.T) {
	stub := &serviceStoreStub{
		createItemFn: func(_ context.Context, newItem *item, actorAccountID string) (*item, error) {
			if actorAccountID != "acc-1" {
				t.Fatalf("expected actor account id acc-1, got %q", actorAccountID)
			}
			return newItem, nil
		},
		getItemHistoryFn: func(context.Context, string) ([]*itemAuditEntry, error) { return nil, nil },
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	created, err := svc.createItem(context.Background(), &createItemInput{
		Name:  "  Keyboard  ",
		Units: 4,
		Price: 8999,
	}, "acc-1")
	if err != nil {
		t.Fatalf("createItem returned error: %v", err)
	}

	if created.Name != "Keyboard" {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}

	if _, err := uuid.Parse(created.ID); err != nil {
		t.Fatalf("expected valid UUID, got %q", created.ID)
	}
}

func TestServiceUpdateItem_RequiresAtLeastOneField(t *testing.T) {
	stub := &serviceStoreStub{}
	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.updateItem(context.Background(), "id-1", &updateItemInput{}, "acc-1")
	if !errors.Is(err, ErrInvalidUpdateItemInput) {
		t.Fatalf("expected ErrInvalidUpdateItemInput, got %v", err)
	}
}

func TestServiceDeleteItem_NotFoundMapsError(t *testing.T) {
	stub := &serviceStoreStub{
		softDeleteFn: func(context.Context, string, string) error {
			return sql.ErrNoRows
		},
		getItemHistoryFn: func(context.Context, string) ([]*itemAuditEntry, error) { return nil, nil },
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	err = svc.deleteItem(context.Background(), "id-1", "acc-1")
	if !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got %v", err)
	}
}

func TestServiceGetDeletedInventory_UsesDeletedStorePath(t *testing.T) {
	stub := &serviceStoreStub{
		getDeletedItemsFn: func(_ context.Context, query *inventoryQuery) (*inventoryPage, error) {
			if query.SortBy != "createdAt" || query.SortOrder != "desc" {
				t.Fatalf("unexpected sort defaults %s/%s", query.SortBy, query.SortOrder)
			}
			return &inventoryPage{Items: []*item{{ID: "deleted-1", Deleted: true}}}, nil
		},
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	inv, err := svc.getDeletedInventory(context.Background(), &getInventoryInput{})
	if err != nil {
		t.Fatalf("getDeletedInventory returned error: %v", err)
	}

	if len(inv.Items) != 1 || !inv.Items[0].Deleted {
		t.Fatalf("expected one deleted item")
	}
}

func TestServiceRestoreItem_NotFoundMapsError(t *testing.T) {
	stub := &serviceStoreStub{
		restoreItemFn: func(context.Context, string, string) (*item, error) {
			return nil, sql.ErrNoRows
		},
	}

	svc, err := NewService(stub, testServiceLogger())
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	_, err = svc.restoreItem(context.Background(), "id-1", "acc-1")
	if !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got %v", err)
	}
}
