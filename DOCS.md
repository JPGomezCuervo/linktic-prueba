# Technical Documentation

## Architecture Overview

This is a monorepo with two independent services: a Go HTTP API (`back/`) and a Vue 3 SPA (`front/`). They communicate over HTTP — the frontend dev server proxies `/api` to the backend. There's no shared code, no RPC, no message queue.

## Backend Architecture

### Layered Design: Handler → Service → Store

Each domain (auth, inventory) follows the same three-layer pattern:

```
HTTP Request → Handler → Service → Store → SQLite
```

**Handler** (`internal/{domain}/handler.go`): Parses HTTP requests, validates query params and JSON bodies, calls the service, maps domain errors to HTTP status codes. No business logic.

**Service** (`internal/{domain}/service.go`): Validates input, enforces business rules, coordinates operations. Depends on the store through an interface (not a concrete type), which makes it testable.

**Store** (`internal/{domain}/store.go`): Raw SQL queries, transactions, cursor encoding/decoding. Knows the database schema directly.

Each layer has its own test file. The service layer tests use a mock store interface; handler tests test error mapping.

### Auth

JWT-based, cookie-delivered. The token is set as an HttpOnly cookie on login and read back on every request. The `AuthMiddleware` (in `internal/middleware/`) validates the token, extracts the account ID, and puts it in the request context. Handlers pull it from context — they never touch the token directly.

Password hashing uses bcrypt. UUIDs are generated with `google/uuid`.

The auth flow:
1. `POST /auth/signup` — creates account, returns 201
2. `POST /auth/login` — validates credentials, sets JWT cookie
3. `GET /auth/me` — returns current user from token
4. `PATCH /auth/me` — updates name/email/password (requires `currentPassword` for password changes)
5. `POST /auth/logout` — clears the cookie

### Pagination

Cursor-based, not offset-based. Two separate cursor schemes:

**Inventory**: Opaque base64-encoded JSON containing the sort value, item ID, sort field, and sort order. The cursor is validated against the current query's sort params to prevent mixing cursors across different sort configurations. Fetches `limit + 1` rows — if there's an extra row, `hasNextPage` is true and the extra row is dropped.

**Orders**: Simpler cursor — base64-encoded `timestamp:id` string. Orders are always sorted `created_at DESC, id DESC`.

Page sizes are restricted to 10, 20, or 50.

### Audit Logging

Every write operation on items creates an entry in `item_audit_logs`. The log is written inside the same transaction as the data change, so it's either both committed or both rolled back.

Operations tracked: `create`, `update`, `delete`, `restore`, `restock`.

Each entry stores: what changed (JSON diff of before/after values), who did it (account ID, name, email), and when. The `GET /inventory/{id}` endpoint returns both the item and its full history.

### Orders / Restock System

Restocking is asynchronous. When a user places a restock order:

1. An order record is created with `status = 'pending'` and a random `delivery_at` timestamp (1–8 minutes in the future)
2. The frontend shows a countdown timer
3. A background ticker in `main.go` runs every 5 seconds, calling `ProcessDueOrders`
4. When `delivery_at` passes, the order is marked `completed`, inventory units are incremented, and a `restock` audit entry is written

The `getOrders` endpoint also processes due orders before returning the list, so there's no gap between the ticker and what the user sees.

Orders are account-scoped in the response (you only see your own), but the ticker processes all pending orders globally.

### Database

SQLite via `modernc.org/sqlite` — pure Go, no CGO. WAL journaling mode, 5s busy timeout, foreign keys enabled. Max 10 open connections.

Schema (5 migrations):
- `000001`: `items` and `accounts` tables with triggers for `updated_at`
- `000002`: `item_audit_logs` table with CHECK constraint on operation enum
- `000003`: Expands audit CHECK to allow `restore` operation
- `000004`: `orders` table with indexes and update trigger
- `000005`: Expands audit CHECK to allow `restock` operation

Soft delete on items uses a `deleted` boolean column. The `orders` table has a foreign key to `items(id)` — no denormalized `item_name`, joins are done at query time.

### Cross-Cutting Concerns

**Logging**: `slog` throughout. Debug logging is opt-in via build-time flags in `internal/debug/`. Each package has its own debug flag (`DebugHandlerFlag`, `DebugServiceFlag`, `DebugStoreFlag`, `DebugInventoryPackageFlag`).

**Error handling**: Domain errors are defined as `var Err...` sentinel values in the service layer. Handlers use `errors.Is` and `errors.As` to map them to HTTP status codes. SQLite errors are caught with type assertion to `*sqlite.Error`.

**Middleware**: `AuthMiddleware` wraps the protected mux, `LoggingMiddleware` wraps the root mux.

## Frontend Architecture

### Structure

```
src/
├── views/          # Page-level components (one per route)
├── components/     # Reusable UI components
├── composables/    # Logic extraction (API calls, formatters, validation)
├── stores/         # Pinia state (auth, table filters, sync signals)
├── router/         # Vue Router with auth guard
└── test/           # Test helpers and mocks
```

### State Management

Pinia stores handle three kinds of state:

**Auth store** (`stores/auth.ts`): User session, login/logout/signup, profile updates, session restoration on page load.

**Table state stores** (`stores/inventoryTable.ts`, `stores/deletedTable.ts`): Filter and sort preferences for each table view. These survive navigation within the same session.

**Sync store** (`stores/ordersSync.ts`): A simple nonce counter that triggers cross-view refreshes. When a restock order is placed, the nonce bumps and the OrdersView refetches.

### API Layer

`useApi.ts` provides a typed `request<T>()` function that handles fetch, credentials (cookies included), error parsing, and 204 responses. `useInventory.ts` and `useOrders.ts` wrap it with domain-specific methods.

### Pagination in Views

Views manage their own cursor state (`currentAfter`, `cursorHistory`) with a fallback mechanism: if a cursor returns empty results and there's history, it pops back to the previous cursor. This handles the edge case where filtering changes make a stored cursor invalid.

### Shared Utilities

Extracted composables to avoid duplication across views:

- `useFormatters.ts`: currency, timestamps, payment methods, order status badges
- `useValidation.ts`: email, name, units, price, password validators
- `useCursorPagination.ts`: reusable cursor pagination logic (not currently used by views but available)
- `EmptyState.vue`: consistent empty state component

### Routing

Vue Router with a `beforeEach` guard. `/login` is the only public route. All `/app/*` routes require authentication. The auth store attempts session restoration on first navigation.

## Testing

### Backend

Unit tests per layer: `handler_test.go`, `service_test.go`, `store_test.go` in both `auth/` and `inventory/`. Run with `go test ./...` from `back/`.

### Frontend

**Unit tests** (Vitest): Store tests mock `fetch` globally and test state transitions. View tests use `@vue/test-utils` with `shallowMount` and mock composables. 47 tests across 6 files.

**E2E tests** (Playwright): Single test covering the full user flow — signup, login, create product, edit name, delete, verify in deleted items. Uses a separate SQLite database (`.env.e2e`) to avoid polluting dev data. Setup and teardown scripts in `back/scripts/`.

## Key Design Decisions

- **No ORM**: Raw SQL in the store layer. SQLite is simple enough that an ORM adds more friction than it solves.
- **Interface-based service dependencies**: The service depends on a `storer` interface, not a concrete `Storage` struct. This enables unit testing without a real database.
- **Prices stored as cents (integers)**: Avoids floating-point rounding issues. The frontend converts to/from dollars for display.
- **HttpOnly JWT cookies**: Prevents XSS token theft. The frontend never reads the token directly.
- **Audit logs in the same transaction as data changes**: Guarantees consistency. No separate event sourcing or outbox pattern.
- **Async restock with countdown**: Simulates a real procurement delay. The frontend schedules a refresh based on the delivery time returned by the API.
- **Cursor pagination with sort-value validation**: Prevents cursor reuse across different sort configurations, which would return incorrect results.
