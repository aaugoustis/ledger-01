#!/usr/bin/env bash
# Sets up a local Ledger/01 on a fresh Ubuntu/Debian machine (a cloud VM or a dev container):
# toolchain, Postgres, a development .env, dependencies and migrations. Safe to re-run.
#
#   bash setup.sh                    # set everything up
#   START=1 bash setup.sh            # ...then run the API (:8080) and web app (:3000)
#   SKIP_PLAYWRIGHT=1 bash setup.sh  # skip the Chromium download for e2e tests
#
# Sign in without Google: the .env enables the dev-only login, so
#   curl -i -X POST localhost:3000/auth/dev/login -H 'content-type: application/json' \
#     -d '{"email":"dev@example.com"}'
# returns a session cookie (or use `page.request.post` in Playwright, as the e2e tests do).

set -euo pipefail
cd "$(dirname "$0")"

NODE_MAJOR="${NODE_MAJOR:-26}"
GO_VERSION="$(awk '/^go /{print $2; exit}' backend/go.mod)"
DEV_EMAILS="dev@example.com,e2e@example.com"

log() { printf '\n\033[1;33m==> %s\033[0m\n' "$*"; }
have() { command -v "$1" >/dev/null 2>&1; }
SUDO=""; [ "$(id -u)" -ne 0 ] && SUDO="sudo"
as_postgres() { if [ -z "$SUDO" ]; then runuser -u postgres -- "$@"; else sudo -u postgres "$@"; fi; }
export DEBIAN_FRONTEND=noninteractive

case "$(uname -m)" in
  x86_64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

export PATH="/usr/local/go/bin:$HOME/go/bin:$HOME/.bun/bin:$HOME/.local/bin:$PATH"
if ! grep -q '# ledger-01 toolchain' "$HOME/.bashrc" 2>/dev/null; then
  printf '\n# ledger-01 toolchain\nexport PATH="/usr/local/go/bin:$HOME/go/bin:$HOME/.bun/bin:$HOME/.local/bin:$PATH"\n' >>"$HOME/.bashrc"
fi

log "System packages"
$SUDO apt-get update -qq
$SUDO apt-get install -y -qq ca-certificates curl git gnupg unzip lsb-release postgresql-client >/dev/null

log "Go $GO_VERSION"
if [ "$(go env GOVERSION 2>/dev/null)" != "go$GO_VERSION" ]; then
  curl -fsSL "https://go.dev/dl/go$GO_VERSION.linux-$ARCH.tar.gz" -o /tmp/go.tgz
  $SUDO rm -rf /usr/local/go
  $SUDO tar -C /usr/local -xzf /tmp/go.tgz
  rm /tmp/go.tgz
fi
go version

log "Node.js $NODE_MAJOR"
if [ "$(node -v 2>/dev/null | cut -d. -f1)" != "v$NODE_MAJOR" ]; then
  curl -fsSL "https://deb.nodesource.com/setup_$NODE_MAJOR.x" | ${SUDO:+$SUDO -E} bash - >/dev/null
  $SUDO apt-get install -y -qq nodejs >/dev/null
fi
node -v

log "Bun"
have bun || curl -fsSL https://bun.sh/install | bash >/dev/null
bun --version

log "Task"
mkdir -p "$HOME/.local/bin"
have task || sh -c "$(curl -fsSL https://taskfile.dev/install.sh)" -- -d -b "$HOME/.local/bin"
task --version

log "Development .env"
if [ ! -f .env ]; then
  cp .env.example .env
  sed -i \
    -e 's/^AUTH_DEV_LOGIN=.*/AUTH_DEV_LOGIN=1/' \
    -e "s/^ALLOWED_EMAILS=.*/ALLOWED_EMAILS=$DEV_EMAILS/" \
    .env
  echo "wrote .env (dev login on, allowlist: $DEV_EMAILS)"
else
  echo ".env exists, leaving it alone"
fi
set -a; . ./.env; set +a

# Postgres: the compose service when Docker works (Go integration tests need Docker anyway),
# otherwise a native PostgreSQL 18 with the same user, password and database.
start_docker() {
  have docker || curl -fsSL https://get.docker.com | $SUDO sh >/dev/null
  if ! docker info >/dev/null 2>&1; then
    if ! { have systemctl && $SUDO systemctl start docker 2>/dev/null; }; then
      $SUDO sh -c 'nohup dockerd >/var/log/dockerd.log 2>&1 &'
    fi
    for _ in $(seq 1 30); do
      [ -S /var/run/docker.sock ] && $SUDO chmod 666 /var/run/docker.sock
      docker info >/dev/null 2>&1 && return 0
      sleep 1
    done
    return 1
  fi
}

native_postgres() {
  if ! have pg_ctlcluster || ! ls /etc/postgresql/18 >/dev/null 2>&1; then
    $SUDO install -d /usr/share/postgresql-common/pgdg
    $SUDO curl -fsSL -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc \
      https://www.postgresql.org/media/keys/ACCC4CF8.asc
    echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" |
      $SUDO tee /etc/apt/sources.list.d/pgdg.list >/dev/null
    $SUDO apt-get update -qq
    $SUDO apt-get install -y -qq postgresql-18 >/dev/null
  fi
  $SUDO sed -i "s/^#\?port = .*/port = ${DB_PORT:-5432}/" /etc/postgresql/18/main/postgresql.conf
  $SUDO pg_ctlcluster 18 main start 2>/dev/null || $SUDO pg_ctlcluster 18 main restart
  until pg_isready -h localhost -p "${DB_PORT:-5432}" -q; do sleep 1; done
  as_postgres psql -p "${DB_PORT:-5432}" -qtc "SELECT 1 FROM pg_roles WHERE rolname = 'ledger'" | grep -q 1 ||
    as_postgres psql -p "${DB_PORT:-5432}" -qc "CREATE ROLE ledger LOGIN PASSWORD 'ledger' CREATEDB"
  as_postgres psql -p "${DB_PORT:-5432}" -qtc "SELECT 1 FROM pg_database WHERE datname = 'ledger'" | grep -q 1 ||
    as_postgres createdb -p "${DB_PORT:-5432}" -O ledger ledger
}

log "Postgres"
if start_docker; then
  task db:up
else
  echo "Docker isn't available here; using a native PostgreSQL 18 (Go integration tests will be skipped or fail)."
  native_postgres
fi

log "Dependencies"
task setup

log "Migrations"
task migrate

if [ "${SKIP_PLAYWRIGHT:-0}" != "1" ]; then
  log "Playwright Chromium"
  (cd apps/web && bun x playwright install --with-deps chromium >/dev/null)
fi

log "Ready"
cat <<'EOF'
  task dev     API on :8080, web on http://localhost:3000
  task test    Go (needs Docker) + bun tests
  task e2e     Playwright (desktop + Pixel 7)
  Sign in:     POST /auth/dev/login {"email":"dev@example.com"}  (see the top of setup.sh)
EOF

if [ "${START:-0}" = "1" ]; then
  exec task dev
fi
