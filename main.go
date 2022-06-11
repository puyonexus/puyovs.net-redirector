package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	c = http.Client{
		Timeout: 5 * time.Minute,
	}

	s = http.Server{
		Addr: ":8080",

		ReadTimeout:       20 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       time.Minute,

		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.WriteHeader(405)
				return
			}

			if strings.HasPrefix(r.URL.Path, "/update") {
				proxyURL := *r.URL
				proxyURL.Host = "upd.puyovs.com"
				proxyURL.Scheme = "https"
				proxyURL.Path = proxyURL.Path[7:]
				err := proxyTo(proxyURL.String(), w)
				if err != nil {
					log.Printf("error proxying update request: %v", err)
					w.WriteHeader(500)
				}
				return
			}

			switch r.URL.Path {
			case "/files/servers.txt":
				err := proxyTo("https://puyovs.com/files/servers.txt", w)
				if err != nil {
					log.Printf("error grabbing server list: %v", err)
					w.WriteHeader(500)
					return
				}

			default:
				redirectTo := *r.URL
				redirectTo.Host = "puyovs.com"
				redirectTo.Scheme = "https"
				http.Redirect(w, r, redirectTo.String(), http.StatusMovedPermanently)
			}
		}),
	}
)

func main() {
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error in server: %v", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Printf("received signal %v, shutting down.", sig)
	signal.Reset()
	close(sigs)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := s.Shutdown(ctx)
	if err != nil {
		log.Printf("error during graceful shutdown: %v", err)
	}
}

func proxyTo(url string, w http.ResponseWriter) error {
	resp, err := c.Get(url)
	if err != nil {
		return fmt.Errorf("error sending request to origin: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("error from origin: %v", resp.Status)
	}
	if ctype := resp.Header.Get("Content-Type"); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("error piping response from origin: %w", err)
	}
	return nil
}
