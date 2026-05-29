package inventory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"modernc.org/sqlite"

	"linktic/internal/common"
	"linktic/internal/debug"
	"linktic/internal/middleware"
	"linktic/internal/telemetry"
)

type Handler struct {
	log     *slog.Logger
	service inventorer
}

type inventorer interface {
	getInventory(ctx context.Context, input *getInventoryInput) (*inventory, error)
	getDeletedInventory(ctx context.Context, input *getInventoryInput) (*inventory, error)
	getOrders(ctx context.Context, accountID string, input *getOrdersInput) (*orders, error)
	createItem(ctx context.Context, input *createItemInput, actorAccountID string) (*item, error)
	getItem(ctx context.Context, id string) (*itemDetails, error)
	updateItem(ctx context.Context, id string, input *updateItemInput, actorAccountID string) (*item, error)
	deleteItem(ctx context.Context, id string, actorAccountID string) error
	restoreItem(ctx context.Context, id string, actorAccountID string) (*item, error)
	restockItem(ctx context.Context, id string, input *restockItemInput, actorAccountID string) (*restockResult, error)
}

func NewHandler(svc inventorer, log *slog.Logger) (*Handler, error) {
	if svc == nil {
		return nil, errors.New("service is required")
	}

	if log == nil {
		return nil, errors.New("logger is required")
	}

	return &Handler{
		log:     log.With("component", "inventory:handler"),
		service: svc,
	}, nil
}

func (h *Handler) GetDeletedInventory(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_get_deleted_inventory", slog.String("path", r.URL.Path))
	}

	input := &getInventoryInput{
		Items:     r.URL.Query().Get("items"),
		After:     r.URL.Query().Get("after"),
		Name:      r.URL.Query().Get("name"),
		UnitsMin:  r.URL.Query().Get("unitsMin"),
		UnitsMax:  r.URL.Query().Get("unitsMax"),
		PriceMin:  r.URL.Query().Get("priceMin"),
		PriceMax:  r.URL.Query().Get("priceMax"),
		SortBy:    r.URL.Query().Get("sortBy"),
		SortOrder: r.URL.Query().Get("sortOrder"),
	}

	inventory, err := 	h.service.getDeletedInventory(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrInvalidPaginationInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, inventory)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_get_orders", slog.String("path", r.URL.Path))
	}

	accountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	input := &getOrdersInput{
		Items: r.URL.Query().Get("items"),
		After: r.URL.Query().Get("after"),
	}

	orders, err := 	h.service.getOrders(r.Context(), accountID, input)
	if err != nil {
		if errors.Is(err, ErrInvalidOrdersInput) || errors.Is(err, ErrInvalidPaginationInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, orders)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_get_inventory", slog.String("path", r.URL.Path))
	}

	input := &getInventoryInput{
		Items:     r.URL.Query().Get("items"),
		After:     r.URL.Query().Get("after"),
		Name:      r.URL.Query().Get("name"),
		UnitsMin:  r.URL.Query().Get("unitsMin"),
		UnitsMax:  r.URL.Query().Get("unitsMax"),
		PriceMin:  r.URL.Query().Get("priceMin"),
		PriceMax:  r.URL.Query().Get("priceMax"),
		SortBy:    r.URL.Query().Get("sortBy"),
		SortOrder: r.URL.Query().Get("sortOrder"),
	}

	inventory, err := 	h.service.getInventory(r.Context(), input)

	if err != nil {
		if errors.Is(err, ErrInvalidPaginationInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var sqliteErr *sqlite.Error

		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, inventory)

	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if 	debug.DebugInventoryHandler {
		h.log.Debug("inventory_response_written", slog.Int("status", http.StatusOK))
	}
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_create_item", slog.String("path", r.URL.Path))
	}

	input, err := common.DecodeJSON[createItemInput](r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	actorAccountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	createdItem, err := 	h.service.createItem(r.Context(), input, actorAccountID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if errors.Is(err, ErrInvalidCreateItemInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var sqliteErr *sqlite.Error

		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusCreated, createdItem)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if 	debug.DebugInventoryHandler {
		h.log.Debug("item_response_written", slog.Int("status", http.StatusCreated))
	}
}

func (h *Handler) RestockItem(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_restock_item", slog.String("path", r.URL.Path))
	}

	id := r.PathValue("id")

	input, err := common.DecodeJSON[restockItemInput](r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	actorAccountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	result, err := 	h.service.restockItem(r.Context(), id, input, actorAccountID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if errors.Is(err, ErrInvalidItemID) || errors.Is(err, ErrInvalidRestockInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrItemNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusCreated, result)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_get_item", slog.String("path", r.URL.Path))
	}

	id := r.PathValue("id")
	selectedItem, err := 	h.service.getItem(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrInvalidItemID) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrItemNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, selectedItem)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if 	debug.DebugInventoryHandler {
		h.log.Debug("item_response_written", slog.Int("status", http.StatusOK))
	}
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_update_item", slog.String("path", r.URL.Path))
	}

	id := r.PathValue("id")

	input, err := common.DecodeJSON[updateItemInput](r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	actorAccountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	updatedItem, err := 	h.service.updateItem(r.Context(), id, input, actorAccountID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if errors.Is(err, ErrInvalidItemID) || errors.Is(err, ErrInvalidUpdateItemInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrItemNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, updatedItem)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if 	debug.DebugInventoryHandler {
		h.log.Debug("item_response_written", slog.Int("status", http.StatusOK))
	}
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_delete_item", slog.String("path", r.URL.Path))
	}

	id := r.PathValue("id")
	actorAccountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	err := 	h.service.deleteItem(r.Context(), id, actorAccountID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if errors.Is(err, ErrInvalidItemID) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrItemNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	if 	debug.DebugInventoryHandler {
		h.log.Debug("item_response_written", slog.Int("status", http.StatusNoContent))
	}
}

func (h *Handler) RestoreItem(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugInventoryHandler {
		h.log.Debug("handling_restore_item", slog.String("path", r.URL.Path))
	}

	id := r.PathValue("id")
	actorAccountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	restoredItem, err := 	h.service.restoreItem(r.Context(), id, actorAccountID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if errors.Is(err, ErrInvalidItemID) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrItemNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, restoredItem)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
