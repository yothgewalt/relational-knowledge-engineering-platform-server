package main

import (
	"github.com/joho/godotenv"

	_ "github.com/yothgewalt/relational-knowledge-engineering-platform-server/docs"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
)

func main() {
	_ = godotenv.Load()

	opts := container.Options{Timezone: "Asia/Bangkok"}
	c := container.New(&opts)

	if err := c.Bootstrap(); err != nil {
		panic(err)
	}

	c.WaitForShutdown()
}
