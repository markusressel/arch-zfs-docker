package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/markusressel/arch-zfs-docker/internal/builder"
	"github.com/markusressel/arch-zfs-docker/internal/config"
	"github.com/markusressel/arch-zfs-docker/internal/k8s"
	"github.com/markusressel/arch-zfs-docker/internal/server"
)

func main() {
	// Build jobs run this same binary in builder mode (see internal/builder).
	if len(os.Args) > 1 && os.Args[1] == builder.Command {
		os.Exit(builder.Main(os.Args[2:]))
	}

	log.Println("[main] Starting Arch ZFS Repository Service...")

	cfg := config.Load()

	k8sClient, err := k8s.NewClient(cfg)
	if err != nil {
		log.Fatalf("[main] Failed to initialize Kubernetes client: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.NewServer(cfg, k8sClient)
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("[main] Server terminated with error: %v", err)
	}

	log.Println("[main] Server gracefully stopped.")
}
