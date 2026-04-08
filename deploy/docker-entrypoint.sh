#!/bin/sh
set -eu

urlencode() {
    node -e 'process.stdout.write(encodeURIComponent(process.argv[1] ?? ""))' "$1"
}

build_payment_database_url() {
    if [ -n "${SUB2APIPAY_DATABASE_URL:-}" ]; then
        printf '%s' "${SUB2APIPAY_DATABASE_URL}"
        return
    fi

    if [ -n "${DATABASE_URL:-}" ]; then
        printf '%s' "${DATABASE_URL}"
        return
    fi

    db_host="${DATABASE_HOST:-localhost}"
    db_port="${DATABASE_PORT:-5432}"
    db_user="${DATABASE_USER:-postgres}"
    db_password="${DATABASE_PASSWORD:-}"
    db_name="${DATABASE_DBNAME:-sub2api}"
    db_sslmode="${DATABASE_SSLMODE:-disable}"

    user_enc="$(urlencode "${db_user}")"
    password_enc="$(urlencode "${db_password}")"

    if [ -n "${db_password}" ]; then
        auth_part="${user_enc}:${password_enc}"
    else
        auth_part="${user_enc}"
    fi

    printf 'postgresql://%s@%s:%s/%s?sslmode=%s' \
        "${auth_part}" \
        "${db_host}" \
        "${db_port}" \
        "${db_name}" \
        "${db_sslmode}"
}

shutdown_children() {
    if [ -n "${pay_pid:-}" ] && kill -0 "${pay_pid}" 2>/dev/null; then
        kill "${pay_pid}" 2>/dev/null || true
        wait "${pay_pid}" 2>/dev/null || true
    fi
    if [ -n "${app_pid:-}" ] && kill -0 "${app_pid}" 2>/dev/null; then
        kill "${app_pid}" 2>/dev/null || true
        wait "${app_pid}" 2>/dev/null || true
    fi
}

# Fix writable paths when running as root, then drop privileges.
if [ "$(id -u)" = "0" ]; then
    mkdir -p /app/data
    chown -R sub2api:sub2api /app/data 2>/dev/null || true
    exec su-exec sub2api "$0" "$@"
fi

# Compatibility: allow "docker run image --help".
if [ "${1#-}" != "$1" ]; then
    set -- /app/sub2api "$@"
fi

# If the command is overridden, do not auto-start the integrated pay stack.
if [ "${1:-}" != "/app/sub2api" ]; then
    exec "$@"
fi

if [ -z "${JWT_SECRET:-}" ]; then
    echo "JWT_SECRET is required for integrated payment mode" >&2
    exit 1
fi

export DATABASE_URL="$(build_payment_database_url)"
export SUB2API_INTERNAL_BASE_URL="${SUB2API_INTERNAL_BASE_URL:-http://127.0.0.1:${SERVER_PORT:-8080}}"
export SUB2APIPAY_INTERNAL_URL="${SUB2APIPAY_INTERNAL_URL:-http://127.0.0.1:3000}"
export NEXT_TELEMETRY_DISABLED=1

cd /app/sub2apipay
/app/prisma-runtime/node_modules/.bin/prisma migrate deploy --config prisma.config.ts

HOSTNAME=127.0.0.1 PORT=3000 node server.js &
pay_pid=$!

cd /app
"$@" &
app_pid=$!

trap shutdown_children INT TERM EXIT

while :; do
    if ! kill -0 "${pay_pid}" 2>/dev/null; then
        wait "${pay_pid}"
        pay_status=$?
        shutdown_children
        exit "${pay_status}"
    fi

    if ! kill -0 "${app_pid}" 2>/dev/null; then
        wait "${app_pid}"
        app_status=$?
        shutdown_children
        exit "${app_status}"
    fi

    sleep 1
done
