# Database Setup

This guide covers setting up PostgreSQL for Budgit in production. The application connects as user `budgit-admin` via unix socket.

## Prerequisites

- PostgreSQL 15+ installed and running
- Root or sudo access on the server

## 1. Create the database user

```bash
sudo -u postgres createuser --login --no-superuser --no-createdb --no-createrole budgit-admin
```

Set a password (only needed if connecting over TCP rather than unix socket):

```bash
sudo -u postgres psql -c "ALTER USER \"budgit-admin\" PASSWORD 'your-secure-password';"
```

## 2. Create the database

```bash
sudo -u postgres createdb --owner=budgit-admin budgit
```

## 3. Configure `pg_hba.conf`

The app connects via unix socket from the `budgit` system user. Add the following line to `pg_hba.conf` (before any generic `local` rules):

```
# TYPE  DATABASE  USER          METHOD
local   budgit    budgit-admin  peer  map=budgit
```

Then add the mapping in `pg_ident.conf` so the `budgit` system user can authenticate as `budgit-admin`:

```
# MAPNAME  SYSTEM-USERNAME  PG-USERNAME
budgit     budgit           budgit-admin
```

Reload PostgreSQL to apply:

```bash
sudo systemctl reload postgresql
```

## 4. Verify the connection

Via unix socket (peer auth):

```bash
sudo -u budgit psql -U budgit-admin -d budgit -c "SELECT 1;"
```

Via TCP (password auth):

```bash
psql -h 127.0.0.1 -U budgit-admin -d budgit -c "SELECT 1;"
```

## 5. Application configuration

Set these variables in `/opt/budgit/.env`:

```bash
DB_DRIVER=pgx
DB_CONNECTION=postgres://budgit-admin@/budgit?host=/run/postgresql&sslmode=disable
```

The connection string uses:
- `budgit-admin` as the PostgreSQL user
- `/budgit` as the database name
- `host=/run/postgresql` to connect via unix socket (adjust the path if your distro uses a different socket directory, e.g. `/var/run/postgresql`)
- `sslmode=disable` since traffic stays on localhost

## 6. Migrations

Migrations run automatically on application startup via Goose (embedded in the binary from `internal/db/migrations/`). No manual migration step is needed.

To verify migrations ran:

```bash
sudo -u budgit psql -U budgit-admin -d budgit -c "\dt"
```

You should see tables: `users`, `tokens`, `profiles`, `files`, `spaces`, `space_members`, `shopping_lists`, `list_items`, `tags`, `expenses`, `expense_tags`, `space_invitations`, and `goose_db_version`.

## Troubleshooting

**"peer authentication failed"** -- The system user running the app doesn't match the `pg_hba.conf` peer mapping. Ensure the app runs as the `budgit` system user and the `pg_ident.conf` mapping is in place.

**"connection refused"** -- PostgreSQL isn't listening on the expected socket path. Check with `pg_lscluster` or `ss -xln | grep postgres` and adjust the `host=` parameter in `DB_CONNECTION`.

**"role budgit-admin does not exist"** -- The user wasn't created. Re-run step 1.

**"database budgit does not exist"** -- The database wasn't created. Re-run step 2.
