# Linktic

Full-stack internal operations tool for managing product inventory, orders, and restocking. Built with Go + SQLite on the backend and Vue 3 + Naive UI on the frontend.

## Project Structure

```
linktic/
├── back/          # Go backend (API server + migration tool)
│   ├── cmd/       # Entry points
│   │   ├── app/   # Main API server
│   │   └── migrate/ # Database migration CLI
│   ├── db/        # SQLite database files + seed data
│   ├── internal/  # Application code (auth, inventory, middleware)
│   ├── migrations/ # SQL migration files
│   └── scripts/   # Utility scripts (E2E setup/teardown)
└── front/         # Vue 3 frontend
    ├── src/       # Application source
    └── e2e/       # Playwright end-to-end tests
```

## Running with Docker (recommended)

The fastest way to get the whole stack running is Docker Compose. From the repo root:

```bash
docker compose up --build
```

This builds and starts both services:

- **Frontend** → http://localhost:5173 (Vite dev server with hot reload)
- **Backend API** → http://localhost:8080

The backend container automatically runs migrations and seeds the database on first
start (`SEED=true`), persisting the SQLite file in the `back-db` named volume. The
frontend proxies `/api` requests to the `back` container.

Useful variations:

```bash
docker compose up -d              # Run in the background (detached)
docker compose up --build         # Rebuild images after code changes
docker compose down               # Stop and remove containers
docker compose down -v            # Also delete the database volume
```

Override the JWT secret (defaults to `dev-secret-change-me`):

```bash
JWT_SECRET=your-secret docker compose up --build
```

> **Note:** The Docker images are runtime-only and **do not run the tests**. To run
> the test suites you need the toolchains installed locally — see
> [Prerequisites](#prerequisites) and the [Testing](#testing) sections below.

## Prerequisites

The following are only required for **local development** and **running the tests**
(the Docker setup above bundles everything else):

- **Go** 1.26+
- **Node.js** 20+
- **npm**
- **sqlite3** CLI (for seeding databases)

## Backend

### Environment Variables

Create a `.env` file in `back/` with the following:

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_PATH` | Absolute path to the SQLite database file | `/home/user/linktic/back/db/linktic.db` |
| `JWT_SECRET` | Secret key for signing JWT tokens | `YEC1q8P+ezIahZXrjbjW50U8HMSCKhX4RzyGKHd6tmY=` |

### Database Setup

```bash
mkdir -p back/db
touch back/db/linktic.db
```

### Migrations

```bash
cd back
go run cmd/migrate/main.go -d up
```

Use `-env` to run migrations against a different database:

```bash
# from /back
go run cmd/migrate/main.go -db=/path/to/db -m=migrations up
```

### Seeding

```bash
# from /back
sqlite3 db/linktic.db < db/seed.sql
```

### Running the Server

```bash
# from /back
go run cmd/app/main.go -d
```

Custom port or env file:

```bash
go run cmd/app/main.go -d -p 3000
go run cmd/app/main.go -env .env.production
```

### Testing

The backend tests run with the standard Go toolchain — they are **not** included in
the Docker image, so you need **Go 1.26+** installed locally:

```bash
# from /back
go test ./...           # Run all tests
go test ./... -v        # Verbose
go test ./internal/...  # A specific package tree
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | Public | Health check |
| POST | `/auth/signup` | Public | Create account |
| POST | `/auth/login` | Public | Login (sets JWT cookie) |
| GET | `/auth/me` | Protected | Get current user |
| PATCH | `/auth/me` | Protected | Update profile |
| POST | `/auth/logout` | Protected | Logout |
| GET | `/inventory` | Protected | List inventory (cursor pagination) |
| POST | `/inventory` | Protected | Create item |
| GET | `/inventory/{id}` | Protected | Get item details + audit history |
| PATCH | `/inventory/{id}` | Protected | Update item |
| DELETE | `/inventory/{id}` | Protected | Soft delete item |
| PATCH | `/inventory/{id}/restore` | Protected | Restore deleted item |
| PATCH | `/inventory/{id}/restock` | Protected | Place restock order |
| GET | `/inventory/deleted` | Protected | List deleted items |
| GET | `/orders` | Protected | List orders |

### Backend Dependencies

```
modernc.org/sqlite          # Pure Go SQLite driver
github.com/golang-migrate/migrate/v4  # Database migrations
github.com/golang-jwt/jwt/v5          # JWT token handling
github.com/google/uuid                # UUID generation
github.com/dustin/go-humanize         # Human-readable formatting
golang.org/x/crypto                   # bcrypt password hashing
```

## Frontend

### Installation

```bash
cd front
npm install
```

### Development

```bash
npm run dev
```

The dev server runs on `http://localhost:5173` and proxies `/api` requests to `http://localhost:8080`.

### Testing

> Tests are not run inside Docker. You need **Node.js 20+** and the project's dev
> dependencies installed locally (`npm install`) to run them.

```bash
# Unit tests (Vitest)
npm run test          # Watch mode
npm run test:run      # Single run

# E2E tests (Playwright)
npm run test:e2e      # Headless
npm run test:e2e:ui   # Interactive UI
```


E2E tests require a separate test database and both servers running:

```bash
# 1. Create and seed the E2E database
bash back/scripts/setup-e2e-db.sh

# 2. Start the backend with the E2E database
cd back && go run cmd/app/main.go -env .env.e2e -p 8080

# 3. Start the frontend (in another terminal)
cd front && npm run dev

# 4. Run tests
cd front && npm run test:e2e

# 5. Clean up
bash back/scripts/teardown-e2e-db.sh
```

### Frontend Dependencies

**Runtime:**

```
vue ^3.5          # UI framework
vue-router ^5.1   # Client-side routing
pinia ^3.0        # State management
naive-ui ^2.44    # Component library
tailwindcss ^4.3  # Utility CSS
```

**Development:**

```
typescript ~6.0   # Type checking
vite ^8.0         # Build tool
vitest ^4.1       # Unit testing
@playwright/test  # E2E testing
@vue/test-utils   # Vue component testing
oxfmt / oxlint    # Formatting and linting
```
