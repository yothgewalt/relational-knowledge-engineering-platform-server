package main

import (
	"os"

	"github.com/joho/godotenv"

	_ "github.com/yothgewalt/relational-knowledge-engineering-platform-server/docs"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/account"
)

func main() {
	_ = godotenv.Load()

	opts := container.Options{Timezone: "Asia/Bangkok"}
	c := container.New(&opts)

	fromEmail := os.Getenv("FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = "noreply@example.com"
	}

	accountModule := account.NewAccountModule(fromEmail)
	if err := c.RegisterModule(accountModule); err != nil {
		panic(err)
	}

	if err := c.Bootstrap(); err != nil {
		panic(err)
	}

	c.WaitForShutdown()
}
