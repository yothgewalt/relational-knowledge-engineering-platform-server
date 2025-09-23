#!/bin/sh

set -e

CONSUL_HTTP_ADDR="http://localhost:8500"
CONSUL_HTTP_TOKEN="${CONSUL_MANAGEMENT_TOKEN}"
APP_TOKEN="KCAdTyMHxnAkdKDKicSMByNXKblybcMcFoLJ8J0edQw="

echo "Starting Consul ACL initialization..."

echo "Waiting for Consul to be ready..."
until curl -s "${CONSUL_HTTP_ADDR}/v1/status/leader" | grep -q ":"; do
    echo "Waiting for Consul leader election..."
    sleep 2
done

echo "Consul is ready"

if ! consul acl policy read -name "application-policy" >/dev/null 2>&1; then
    echo "Creating application policy..."
    consul acl policy create \
        -name "application-policy" \
        -description "Policy for relational knowledge platform application" \
        -rules '
node_prefix "" {
  policy = "read"
}

service_prefix "" {
  policy = "write"
}

key_prefix "" {
  policy = "write"
}

session_prefix "" {
  policy = "write"
}

agent_prefix "" {
  policy = "read"
}

query_prefix "" {
  policy = "read"
}'
    echo "Application policy created"
else
    echo "Application policy already exists"
fi

if ! consul acl token read -id "${APP_TOKEN}" >/dev/null 2>&1; then
    echo "Creating application token..."
    consul acl token create \
        -secret="${APP_TOKEN}" \
        -description "Token for relational knowledge platform application" \
        -policy-name "application-policy"
    echo "Application token created: ${APP_TOKEN}"
else
    echo "Application token already exists"
fi

echo "Consul ACL initialization completed!"
echo "Application can now use token: ${APP_TOKEN}"