package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/bootstrap"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--health" {
		performHealthCheck()
		return
	}

	bootstrap.New()
}

func performHealthCheck() {
	cfg := config.Load()
	healthURL := fmt.Sprintf("http://%s:%d/health", cfg.Server.Host, cfg.Server.Port)
	
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	
	resp, err := client.Get(healthURL)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		fmt.Println("Health check passed")
		os.Exit(0)
	} else {
		fmt.Printf("Health check failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}
}
