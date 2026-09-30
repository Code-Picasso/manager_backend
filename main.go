package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"manager-backend/internal/config"
	"manager-backend/internal/database"
	"manager-backend/internal/httpapi"
	"manager-backend/internal/store"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	if err := database.PingWithRetry(ctx, pool, 30, 2*time.Second); err != nil {
		log.Fatalf("database unreachable: %v", err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httpapi.New(store.New(pool)).Routes(),
	}

	go func() {
		log.Printf("%s listening on :%s", cfg.AppName, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
