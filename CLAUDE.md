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
- `internal/app/` — dependency injection, wires all repositories/services
- `internal/handler/` — HTTP handlers grouped by domain (auth, space, dashboard, home)
- `internal/service/` — business logic
- `internal/repository/` — data access with sqlx, interface-based
- `internal/model/` — data structs with `db:` tags
- `internal/middleware/` — ordered chain: Config → Logging → NoCache → CSRF → Auth → URLPath
- `internal/router/` - custom router to group routes together and chain middleware.
- `internal/routes/routes.go` — all route definitions with middleware wrapping
- `internal/ui/` — templ templates organized as pages/, components/, layouts/, blocks/
- `internal/misc/` - miscellanous packages such as timezones
- `assets/` — static files (CSS, JS, fonts) embedded in binary via `go:embed`

## Key Patterns

**HTMX + hyperscript**: Server returns HTML fragments. Use `hx-*` attributes for AJAX, `_=` for client-side interactivity (toggle, events). Pattern `hx-swap="none"` + hyperscript `send <event> to #target` for fire-and-forget + refresh.

**`?from=card` query param**: Handlers check this to return different component variants for card vs detail page contexts (e.g., `UpdateList`, `DeleteList`, `ToggleItem`).

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

# templui Components

> templ-based UI components for Go. Open source. Customizable. Accessible.

## Overview

templui is a collection of beautifully designed, accessible UI components built with templ and Go.
Components are designed to be composable, customizable, and easy to integrate into your Go projects.

- [Introduction](https://templui.io/docs/introduction): Core principles and getting started guide
- [How to Use](https://templui.io/docs/how-to-use): CLI installation and usage guide
- [Components](https://templui.io/docs/components): Component overview and catalog
- [Themes](https://templui.io/docs/themes): Theme customization and styling
- [GitHub](https://github.com/templui/templui): Source code and issue tracker

## Form & Input

- [Button](https://templui.io/docs/components/button): Button component with multiple variants.
- [Calendar](https://templui.io/docs/components/calendar): Calendar component for date selection.
- [Checkbox](https://templui.io/docs/components/checkbox): Checkbox input component.
- [Date Picker](https://templui.io/docs/components/date-picker): Date picker component combining input and calendar.
- [Form](https://templui.io/docs/components/form): Form container with validation support.
- [Input](https://templui.io/docs/components/input): Text input component.
- [Input OTP](https://templui.io/docs/components/input-otp): One-time password input component.
- [Label](https://templui.io/docs/components/label): Form label component.
- [Radio](https://templui.io/docs/components/radio): Radio button group component.
- [Rating](https://templui.io/docs/components/rating): Star rating input component.
- [Select Box](https://templui.io/docs/components/select-box): Searchable select component.
- [Slider](https://templui.io/docs/components/slider): Slider input component.
- [Switch](https://templui.io/docs/components/switch): Toggle switch component.
- [Tags Input](https://templui.io/docs/components/tags-input): Tags input component.
- [Textarea](https://templui.io/docs/components/textarea): Multi-line text input component.
- [Time Picker](https://templui.io/docs/components/time-picker): Time picker component.

## Layout & Navigation

- [Accordion](https://templui.io/docs/components/accordion): Collapsible accordion component.
- [Breadcrumb](https://templui.io/docs/components/breadcrumb): Breadcrumb navigation component.
- [Pagination](https://templui.io/docs/components/pagination): Pagination component for lists and tables.
- [Separator](https://templui.io/docs/components/separator): Visual divider between content sections.
- [Sidebar](https://templui.io/docs/components/sidebar): Collapsible sidebar component for app layouts.
- [Tabs](https://templui.io/docs/components/tabs): Tabbed interface component.

## Overlays & Dialogs

- [Dialog](https://templui.io/docs/components/dialog): Modal dialog component.
- [Dropdown](https://templui.io/docs/components/dropdown): Dropdown menu component.
- [Popover](https://templui.io/docs/components/popover): Floating popover component.
- [Sheet](https://templui.io/docs/components/sheet): Slide-out panel component (drawer).
- [Tooltip](https://templui.io/docs/components/tooltip): Tooltip component for additional context.

## Feedback & Status

- [Alert](https://templui.io/docs/components/alert): Alert component for messages and notifications.
- [Badge](https://templui.io/docs/components/badge): Badge component for labels and status indicators.
- [Progress](https://templui.io/docs/components/progress): Progress bar component.
- [Skeleton](https://templui.io/docs/components/skeleton): Skeleton loading placeholder.
- [Toast](https://templui.io/docs/components/toast): Toast notification component.

## Display & Media

- [Aspect Ratio](https://templui.io/docs/components/aspect-ratio): Container that maintains aspect ratio.
- [Avatar](https://templui.io/docs/components/avatar): Avatar component for user profiles.
- [Card](https://templui.io/docs/components/card): Card container component.
- [Carousel](https://templui.io/docs/components/carousel): Carousel component with navigation controls.
- [Charts](https://templui.io/docs/components/charts): Chart components for data visualization.
- [Table](https://templui.io/docs/components/table): Table component for displaying data.

## Misc

- [Collapsible](https://templui.io/docs/components/collapsible): Collapsible container component.
- [Copy Button](https://templui.io/docs/components/copy-button): Copy to clipboard button component.
- [Icon](https://templui.io/docs/components/icon): SVG icon component library.

