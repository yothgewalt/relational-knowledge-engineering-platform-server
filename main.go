package main

import (
	"os"

	"github.com/joho/godotenv"

	_ "github.com/yothgewalt/relational-knowledge-engineering-platform-server/docs"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/account"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/identity"
)

func main() {
	_ = godotenv.Load()

	opts := container.Options{Timezone: "Asia/Bangkok"}
	c := container.New(&opts)

	accountModule := account.NewAccountModule()
	if err := c.RegisterModule(accountModule); err != nil {
		panic(err)
	}

	fromEmail := os.Getenv("FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = "noreply@example.com"
	}
	
	identityModule := identity.NewIdentityModule(fromEmail)
	if err := c.RegisterModule(identityModule); err != nil {
		panic(err)
	}

	if err := c.Bootstrap(); err != nil {
		panic(err)
	}

	c.WaitForShutdown()
}
