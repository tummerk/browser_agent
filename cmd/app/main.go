package main

import (
	"browser-agent/internal/application"
	"context"
)

func main() {
	ctx := context.Background()
	err := application.Run(ctx)
	if err != nil {
		panic(err)
	}
}
