package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"calculator/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer stop()

	srv := server.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/calc", srv.HandleCalc)
	mux.HandleFunc("/metrics", srv.HandleMetrics)

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Printf("sum=%d\nsub=%d\n", srv.Sum(), srv.Sub())
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "http-сервер: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Println("калькулятор запущен на :8080")

	// Блокировка до получения сигнала.
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "graceful shutdown: %v\n", err)
		_ = httpServer.Close()
	}

	fmt.Printf("sum=%d\nsub=%d\n", srv.Sum(), srv.Sub())
}