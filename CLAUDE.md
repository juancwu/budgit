# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Start full dev environment (tailwind watch + templ hot-reload + Go server)
task dev

# Generate templ files only
go tool templ generate

# Build (must run templ generate first if .templ files changed)
go build ./...

# Run the server directly
go run ./cmd/server/main.go

# Tailwind CSS
tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css         # one-shot
tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --watch  # watch mode

# Database migrations (auto-run on startup, but can run manually)
# Migrations are in internal/db/migrations/ using Goose format
```

## Architecture

**Stack**: Go + templ (HTML templating) + HTMX + hyperscript + Tailwind CSS v4 + PostgreSQL

**Layered architecture**: handler → service → repository → DB

- `cmd/server/main.go` — entry point, loads config, initializes app, starts server
- `internal/app/` — dependency injection, wires all repositories/services/event broker
- `internal/handler/` — HTTP handlers grouped by domain (auth, space, dashboard, home)
- `internal/service/` — business logic, event publishing
- `internal/repository/` — data access with sqlx, interface-based
- `internal/model/` — data structs with `db:` tags
- `internal/middleware/` — ordered chain: Config → Logging → NoCache → CSRF → Auth → URLPath
- `internal/routes/routes.go` — all route definitions with middleware wrapping
- `internal/event/` — SSE pub/sub broker, space-scoped channels
- `internal/ui/` — templ templates organized as pages/, components/, layouts/, blocks/
- `assets/` — static files (CSS, JS, fonts) embedded in binary via `go:embed`

## Key Patterns

**HTMX + hyperscript**: Server returns HTML fragments. Use `hx-*` attributes for AJAX, `_=` for client-side interactivity (toggle, events). Pattern `hx-swap="none"` + hyperscript `send <event> to #target` for fire-and-forget + refresh.

**`?from=card` query param**: Handlers check this to return different component variants for card vs detail page contexts (e.g., `UpdateList`, `DeleteList`, `ToggleItem`).

**SSE events**: `event.Broker` publishes space-scoped events. Templates subscribe via `hx-sse="connect:/app/spaces/{id}/stream"` and trigger refreshes with `hx-trigger="sse:event_name"`.

**CSRF**: Double-submit cookie pattern. Use `@csrf.Token()` in every form.

**Auth flow**: JWT in HTTP-only cookies. Routes wrapped with `middleware.RequireAuth` and `middleware.RequireSpaceAccess` for space routes.

**UI rendering**: All handlers use `ui.Render(w, r, component)` which injects config, user, and CSRF context.

**Element IDs**: Must be unique across repeated components — use entity IDs like `list-card-{id}`, `lch-{id}`, `item-{id}`.

## templui Component Library (v1.2.0)

Components live in `internal/ui/components/` — button, input, checkbox, dialog, pagination, icon, sidebar, toast, etc. Icons use `icon.Pencil`, `icon.Trash2`, `icon.Plus`, `icon.X`, `icon.ChevronLeft` etc. from Lucide via `internal/ui/components/icon/icon_defs.go`.

## Configuration

App reads from `.env` file via `godotenv`. Key vars: `APP_ENV`, `APP_URL`, `DB_DRIVER` (pgx/sqlite), `DB_CONNECTION`, `JWT_SECRET`, `PORT`. See `internal/config/config.go` for all fields.

## Database

PostgreSQL (pgx driver) or SQLite. Migrations auto-run on startup from `internal/db/migrations/` (Goose SQL format, embedded via `go:embed`). 8 migration files covering users, tokens, profiles, spaces, shopping lists, tags, expenses, invitations.
