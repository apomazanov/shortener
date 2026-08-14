package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type AliasPool struct {
	mu      sync.RWMutex
	aliases []string
}

func (p *AliasPool) Add(alias string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.aliases = append(p.aliases, alias)
}

func (p *AliasPool) GetRandom() (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.aliases) == 0 {
		return "", false
	}
	idx := rand.Intn(len(p.aliases))
	return p.aliases[idx], true
}

func (p *AliasPool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.aliases)
}

type ShortenRequest struct {
	LongURL string `json:"url"`
}

type ShortenResponse struct {
	Alias string `json:"result"`
}

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	targetURL   = "http://127.0.0.1:8080"
	targetRPS   = 2000
	writeRatio  = 0.5
	duration    = 10 * time.Second
	workers     = 10
)

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func main() {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			MaxIdleConns:        workers * 2,
			MaxIdleConnsPerHost: workers * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var pool AliasPool

	initialAlias, err := executeWrite(targetURL, httpClient)
	if err != nil {
		fmt.Printf("Ошибка при инициализации: %v\n", err)
		os.Exit(1)
	}
	pool.Add(initialAlias)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	rateTicker := time.NewTicker(time.Second / time.Duration(targetRPS))
	defer rateTicker.Stop()

	jobs := make(chan struct{}, workers*2)
	var wg sync.WaitGroup

	// Исправленный запуск воркеров
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				if rand.Float64() < writeRatio {
					alias, err := executeWrite(targetURL, httpClient)
					if err == nil {
						pool.Add(alias)
					} else {
						fmt.Printf("Ошибка записи: %v\n", err)
					}
				} else {
					alias, ok := pool.GetRandom()
					if !ok {
						continue
					}
					err := executeRead(alias, httpClient)
					if err != nil {
						fmt.Printf("Ошибка чтения: %v\n", err)
					}
				}
			}
		}()
	}

loop:
	for {
		select {
		case <-sigCtx.Done():
			break loop
		case <-rateTicker.C:
			select {
			case jobs <- struct{}{}:
			default:
			}
		}
	}

	close(jobs)
	wg.Wait()
	fmt.Println("Тестирование успешно завершено.")
}

func executeWrite(baseURL string, client *http.Client) (string, error) {
	longURL := fmt.Sprintf("https://example.com/%s/%s", randString(5), randString(10))
	body, _ := json.Marshal(ShortenRequest{LongURL: longURL})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/shorten", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var res ShortenResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if res.Alias == "" {
		return "", fmt.Errorf("empty alias received")
	}

	return res.Alias, nil
}

func executeRead(alias string, client *http.Client) error {
	req, err := http.NewRequest(http.MethodGet, alias, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTemporaryRedirect {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
