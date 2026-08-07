package main

import (
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

	server := server.NewServer(":6379", store, router)

	go func() {
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	sign := make(chan os.Signal, 1)

	signal.Notify(sign, syscall.SIGTERM, syscall.SIGINT)

	<-sign

	if err := server.Stop(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("server was stopped")
}
