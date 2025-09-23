#!/bin/sh

set -e

echo "Starting Consul with ACL initialization..."

consul agent -config-file=/consul/config/server.json \
    -acl-master-token="${CONSUL_MANAGEMENT_TOKEN}" \
    -acl-agent-token="${CONSUL_AGENT_TOKEN}" &

CONSUL_PID=$!

sleep 5

/consul/config/consul-init.sh

wait $CONSUL_PID