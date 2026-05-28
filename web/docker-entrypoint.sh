#!/bin/sh
set -eu

API_URL="${ROUTER_SYNC_API_URL:-}"
# Escape backslashes and quotes for JS string
ESCAPED=$(printf '%s' "$API_URL" | sed 's/\\/\\\\/g; s/"/\\"/g')

cat > /usr/share/nginx/html/config.js <<EOF
window.__ROUTER_SYNC_CONFIG__ = {
  apiBaseUrl: "${ESCAPED}"
};
EOF

exec "$@"
