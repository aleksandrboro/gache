package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aleksandrboro/gache/internal/aof"
	"github.com/aleksandrboro/gache/internal/command"
	"github.com/aleksandrboro/gache/internal/pubsub"
	"github.com/aleksandrboro/gache/internal/server"
	"github.com/aleksandrboro/gache/internal/storage"
)

func main() {
	store := storage.NewStore()
	router := command.NewRouter()
	router.RegisterCommands()

	aof.LoadAOF("appendonly.aof", store, router)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	aofWriter, err := aof.NewAOFWriter(ctx, "appendonly.aof", "always")
	if err != nil {
		panic(err)
	}

	go store.StartExpirationLoop(ctx)

	hub := pubsub.NewHub()

	server := server.NewServer(":6378", store, router, aofWriter, hub)

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
	aofWriter.Close()
	if err := server.Stop(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("server was stopped")
}
