package inventory

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"linktic/internal/middleware"
)

type handlerServiceStub struct {
	getInventoryFn func(ctx context.Context, input *getInventoryInput) (*inventory, error)
	getDeletedInventoryFn func(ctx context.Context, input *getInventoryInput) (*inventory, error)
	getOrdersFn    func(ctx context.Context, accountID string, input *getOrdersInput) (*orders, error)
	createItemFn   func(ctx context.Context, input *createItemInput, actorAccountID string) (*item, error)
	getItemFn      func(ctx context.Context, id string) (*itemDetails, error)
	updateItemFn   func(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error)
	deleteItemFn   func(ctx context.Context, id string, actorAccountID string) error
	restoreItemFn  func(ctx context.Context, id string, actorAccountID string) (*item, error)
	restockItemFn  func(ctx context.Context, id string, input *restockItemInput, actorAccountID string) (*restockResult, error)
}

func (s *handlerServiceStub) getInventory(ctx context.Context, input *getInventoryInput) (*inventory, error) {
	return s.getInventoryFn(ctx, input)
}

func (s *handlerServiceStub) getDeletedInventory(ctx context.Context, input *getInventoryInput) (*inventory, error) {
	if s.getDeletedInventoryFn == nil {
		return &inventory{}, nil
	}
	return s.getDeletedInventoryFn(ctx, input)
}

func (s *handlerServiceStub) getOrders(ctx context.Context, accountID string, input *getOrdersInput) (*orders, error) {
	if s.getOrdersFn == nil {
		return &orders{}, nil
	}
	return s.getOrdersFn(ctx, accountID, input)
}

func (s *handlerServiceStub) createItem(ctx context.Context, input *createItemInput, actorAccountID string) (*item, error) {
	return s.createItemFn(ctx, input, actorAccountID)
}

func (s *handlerServiceStub) getItem(ctx context.Context, id string) (*itemDetails, error) {
	return s.getItemFn(ctx, id)
}

func (s *handlerServiceStub) updateItem(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error) {
	return s.updateItemFn(ctx, id, input, actorAccountID)
}

func (s *handlerServiceStub) deleteItem(ctx context.Context, id string, actorAccountID string) error {
	return s.deleteItemFn(ctx, id, actorAccountID)
}

func (s *handlerServiceStub) restoreItem(ctx context.Context, id string, actorAccountID string) (*item, error) {
	if s.restoreItemFn == nil {
		return nil, ErrItemNotFound
	}
	return s.restoreItemFn(ctx, id, actorAccountID)
}

func (s *handlerServiceStub) restockItem(ctx context.Context, id string, input *restockItemInput, actorAccountID string) (*restockResult, error) {
	if s.restockItemFn == nil {
		return nil, ErrItemNotFound
	}
	return s.restockItemFn(ctx, id, input, actorAccountID)
}

func testHandlerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetInventory_ReturnsBadRequestOnInvalidPagination(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{
		getInventoryFn: func(context.Context, *getInventoryInput) (*inventory, error) {
			return nil, ErrInvalidPaginationInput
		},
	}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/inventory?items=15", nil)
	rr := httptest.NewRecorder()

	h.GetInventory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetInventory_ReturnsBadRequestOnInvalidSortOrder(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{
		getInventoryFn: func(context.Context, *getInventoryInput) (*inventory, error) {
			return nil, ErrInvalidPaginationInput
		},
	}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/inventory?sortBy=price&sortOrder=sideways", nil)
	rr := httptest.NewRecorder()

	h.GetInventory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetInventory_PassesFilterAndSortParamsToService(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{
		getInventoryFn: func(_ context.Context, input *getInventoryInput) (*inventory, error) {
			if input.Name != "desk" {
				t.Fatalf("expected name=desk, got %q", input.Name)
			}
			if input.UnitsMin != "1" || input.UnitsMax != "10" {
				t.Fatalf("unexpected units range %q-%q", input.UnitsMin, input.UnitsMax)
			}
			if input.PriceMin != "200" || input.PriceMax != "1000" {
				t.Fatalf("unexpected price range %q-%q", input.PriceMin, input.PriceMax)
			}
			if input.SortBy != "price" || input.SortOrder != "asc" {
				t.Fatalf("unexpected sort %q/%q", input.SortBy, input.SortOrder)
			}
			return &inventory{}, nil
		},
	}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/inventory?name=desk&unitsMin=1&unitsMax=10&priceMin=200&priceMax=1000&sortBy=price&sortOrder=asc", nil)
	rr := httptest.NewRecorder()

	h.GetInventory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestCreateItem_ReturnsBadRequestOnInvalidJSON(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/inventory", strings.NewReader("{"))
	rr := httptest.NewRecorder()

	h.CreateItem(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestUpdateItem_ReturnsOK(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{
		updateItemFn: func(context.Context, string, *updateItemInput, string) (*item, error) {
			return &item{ID: "item-1"}, nil
		},
	}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/inventory/item-1", strings.NewReader(`{"price":3000}`))
	req.SetPathValue("id", "item-1")
	req = req.WithContext(middleware.WithAccountID(req.Context(), "acc-1"))
	rr := httptest.NewRecorder()

	h.UpdateItem(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestDeleteItem_NotFoundReturns404(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{
		deleteItemFn: func(context.Context, string, string) error {
			return ErrItemNotFound
		},
	}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/inventory/item-1", nil)
	req.SetPathValue("id", "item-1")
	req = req.WithContext(middleware.WithAccountID(req.Context(), "acc-1"))
	rr := httptest.NewRecorder()

	h.DeleteItem(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestGetDeletedInventory_ReturnsBadRequestOnInvalidPagination(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{
		getDeletedInventoryFn: func(context.Context, *getInventoryInput) (*inventory, error) {
			return nil, ErrInvalidPaginationInput
		},
	}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/inventory/deleted?items=15", nil)
	rr := httptest.NewRecorder()

	h.GetDeletedInventory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestRestoreItem_ReturnsOK(t *testing.T) {
	h, err := NewHandler(&handlerServiceStub{
		restoreItemFn: func(context.Context, string, string) (*item, error) {
			return &item{ID: "item-1", Deleted: false}, nil
		},
	}, testHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/inventory/item-1/restore", nil)
	req.SetPathValue("id", "item-1")
	req = req.WithContext(middleware.WithAccountID(req.Context(), "acc-1"))
	rr := httptest.NewRecorder()

	h.RestoreItem(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}
