storage "consul" {
  address = "consul:8500"
  path    = "vault/"
  scheme  = "http"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = "true"
}

api_addr = "http://kartoffel_vault:8200"
cluster_addr = "http://kartoffel_vault:8201"
ui = true

# Disable mlock for development
disable_mlock = true

# Log level
log_level = "INFO"

# Default lease TTL
default_lease_ttl = "168h"
max_lease_ttl = "720h"
