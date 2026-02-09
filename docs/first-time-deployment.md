# First-Time Deployment

This guide walks through every manual step needed on a fresh server before the CI/CD workflow can auto-deploy.
After completing this once, all future deploys happen automatically when you push a `v*` tag.

## Prerequisites

- A Linux server (Debian/Ubuntu assumed, adjust package commands for other distros)
- Root or sudo access
- A domain name pointed at the server's IP
- PostgreSQL 15+ installed
- Caddy installed

## Step 1: Create the system user

Create a dedicated `budgit` user with no login shell and no home directory:

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin budgit
```

Create a dedicated deploy user that CI will SSH into:

```bash
sudo useradd --create-home --shell /bin/bash deploy
```

Generate an SSH key pair (on your local machine or CI):

```bash
ssh-keygen -t ed25519 -f deploy_key -N "" -C "budgit-ci-deploy"
```

Install the public key on the server:

```bash
sudo mkdir -p /home/deploy/.ssh
sudo cp deploy_key.pub /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh
sudo chmod 700 /home/deploy/.ssh
sudo chmod 600 /home/deploy/.ssh/authorized_keys
```

Grant the deploy user the specific sudo permissions it needs (no password):

```bash
sudo tee /etc/sudoers.d/budgit-deploy > /dev/null << 'EOF'
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart budgit
EOF
sudo chmod 440 /etc/sudoers.d/budgit-deploy
```

The deploy user also needs write access to the deploy path:

```bash
sudo setfacl -m u:deploy:rwx /opt/budgit
```

Or alternatively, add `deploy` to the `budgit` group and ensure group write:

```bash
sudo usermod -aG budgit deploy
sudo chmod 770 /opt/budgit
```

## Step 2: Create the application directory

```bash
sudo mkdir -p /opt/budgit
sudo chown budgit:budgit /opt/budgit
sudo chmod 750 /opt/budgit
```

## Step 3: Set up PostgreSQL

Follow [docs/database-setup.md](database-setup.md) in full. By the end you should have:

- A `budgit-admin` PostgreSQL role
- A `budgit` database owned by `budgit-admin`
- `pg_hba.conf` peer auth with an ident map so the `budgit` system user authenticates as `budgit-admin`

Verify it works:

```bash
sudo -u budgit psql -U budgit-admin -d budgit -c "SELECT 1;"
```

## Step 4: Create the environment file

```bash
sudo -u budgit tee /opt/budgit/.env > /dev/null << 'EOF'
APP_ENV=production
APP_URL=https://budgit.now
HOST=127.0.0.1
PORT=9000

DB_DRIVER=pgx
DB_CONNECTION=postgres://budgit-admin@/budgit?host=/run/postgresql&sslmode=disable

JWT_SECRET=<run: openssl rand -base64 32>

MAILER_SMTP_HOST=
MAILER_SMTP_PORT=587
MAILER_IMAP_HOST=
MAILER_IMAP_PORT=993
MAILER_USERNAME=
MAILER_PASSWORD=
MAILER_EMAIL_FROM=
SUPPORT_EMAIL=
EOF
```

Generate and fill in the `JWT_SECRET`:

```bash
openssl rand -base64 32
```

Fill in the mailer variables if email is configured. Lock down permissions:

```bash
sudo chmod 600 /opt/budgit/.env
```

## Step 5: Do the initial binary deploy

Build locally (or on any machine with Go + Tailwind + Task installed):

```bash
task build
```

Copy the binary to the server:

```bash
scp ./dist/budgit your-user@your-server:/tmp/budgit
ssh your-user@your-server "sudo mv /tmp/budgit /opt/budgit/budgit && sudo chown budgit:budgit /opt/budgit/budgit && sudo chmod 755 /opt/budgit/budgit"
```

## Step 6: Install the systemd service

Copy the unit file from this repo:

```bash
scp docs/budgit.service your-user@your-server:/tmp/budgit.service
ssh your-user@your-server "sudo mv /tmp/budgit.service /etc/systemd/system/budgit.service"
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable budgit
sudo systemctl start budgit
```

Check it's running:

```bash
sudo systemctl status budgit
curl http://127.0.0.1:9000/healthz
```

You should see `ok`.

## Step 7: Configure Caddy

Replace the existing `budgit.now` site block in your Caddyfile (typically `/etc/caddy/Caddyfile`).

Before (static file server):

```caddyfile
budgit.now, www.budgit.now, mta-sts.budgit.now, autodiscover.budgit.now {
    import common_headers
    import budgit_now_ssl
    root * /var/www/budgit.now
    file_server
}
```

After (split app from other subdomains):

```caddyfile
budgit.now, www.budgit.now {
    import common_headers
    import budgit_now_ssl
    encode gzip zstd

    handle /.well-known/* {
        root * /var/www/budgit.now
        file_server
    }

    handle {
        reverse_proxy 127.0.0.1:9000 {
            health_uri /healthz
            health_interval 10s
            health_timeout 3s
        }
    }
}

mta-sts.budgit.now, autodiscover.budgit.now {
    import common_headers
    import budgit_now_ssl
    root * /var/www/budgit.now
    file_server
}
```

Reload Caddy:

```bash
sudo systemctl reload caddy
```

Verify the public endpoint:

```bash
curl https://budgit.now/healthz
```

## Step 8: Configure Forgejo secrets

In your Forgejo repository, go to **Settings > Secrets** and add:

| Secret | Value |
|---|---|
| `SSH_KEY` | Contents of `deploy_key` (the private key) |
| `SSH_USER` | `deploy` |
| `SSH_HOST` | Your server's IP or hostname |
| `DEPLOY_PATH` | `/opt/budgit` |
| `APP_URL` | `https://budgit.now` |

## Step 9: Verify auto-deploy

Tag and push to trigger the workflow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Watch the workflow in Forgejo's Actions tab. It should:

1. Build the binary with the version baked in
2. SCP it to the server
3. Restart the service
4. Pass the health check

Confirm the version is running:

```bash
journalctl -u budgit --no-pager -n 5
```

You should see a log line like `server starting version=v0.1.0`.

## Summary

After completing these steps, the deployment flow is:

```
git tag v1.2.3 && git push origin v1.2.3
  -> Forgejo workflow triggers
  -> Builds binary with version embedded
  -> SCPs to server, restarts systemd
  -> Health check verifies
  -> Auto-rollback on failure
```

No further manual steps are needed for subsequent deploys.
