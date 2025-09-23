package main

import (
	_ "github.com/yothgewalt/relational-knowledge-engineering-platform-server/docs"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
)

func main() {
	opts := container.Options{Timezone: "Asia/Bangkok"}
	c := container.New(&opts)

	if err := c.Bootstrap(); err != nil {
		panic(err)
	}

	c.WaitForShutdown()
}
