package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aleksandrboro/gache/internal/command"
	"github.com/aleksandrboro/gache/internal/server"
	"github.com/aleksandrboro/gache/internal/storage"
)

func main() {
	store := storage.NewStore()
	router := command.NewRouter()
	router.RegisterCommands()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go store.StartExpirationLoop(ctx)

	server := server.NewServer(":6378", store, router)

	go func() {
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	sign := make(chan os.Signal, 1)

	signal.Notify(sign, syscall.SIGTERM, syscall.SIGINT)

	<-sign

	cancel()
	if err := server.Stop(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("server was stopped")
}
