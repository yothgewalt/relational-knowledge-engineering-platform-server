package main

import (
	_ "github.com/yothgewalt/relational-knowledge-engineering-platform-server/docs"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
)

func main() {
	cfg := config.MustLoad()
	_ = cfg
}
