package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func main() {
	workers := flag.Int("workers", 10, "количество параллельных воркеров")
	url := flag.String("url", "http://localhost:8080", "базовый URL калькулятора")
	flag.Parse()

	if *workers <= 0 {
		fmt.Fprintln(os.Stderr, "ошибка: workers должно быть > 0")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer stop()

	// Один общий клиент с пулом соединений под количество воркеров.
	// Не создаём новый клиент на каждый запрос.
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        *workers,
			MaxIdleConnsPerHost: *workers,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 30 * time.Second,
	}
	defer client.CloseIdleConnections()

	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(ctx, client, *url, id)
		}(i)
	}

	fmt.Printf("генератор: %d воркеров → %s\n", *workers, *url)
	wg.Wait()
	fmt.Println("генератор остановлен")
}


func worker(ctx context.Context, client *http.Client, baseURL string, id int) {
	// Собственный генератор на воркер: нет глобального состояния,
	// нет конкуренции за seed.
	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano())+uint64(id), uint64(id)*7919))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		num := rng.IntN(201) - 100
		target := baseURL + "/calc?num=" + strconv.Itoa(num)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, nil)
		if err != nil {
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			timer := time.NewTimer(50 * time.Millisecond)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
			continue
		}
		resp.Body.Close()
	}
}