package main

import (
	"log"
	"net/http"

	"github.com/foxbits/open-crawl/internal/api"
	"github.com/foxbits/open-crawl/internal/config"
)

func main() {
	cfg := config.Load()

	handler := api.NewCrawlHandler(cfg.Crawl4AIBaseURL, cfg.RequestTimeout)
	extractHandler := api.NewExtractHandler(cfg.Crawl4AIBaseURL, cfg.RequestTimeout)
	healthHandler := api.NewHealthHandler(cfg.Crawl4AIBaseURL, cfg.RequestTimeout)

	mux := http.NewServeMux()
	mux.Handle("/crawl", handler)
	mux.Handle("/extract", extractHandler)
	mux.Handle("/health", healthHandler)

	log.Printf("Starting Open Crawl Proxy on %s", cfg.ListenAddr)
	log.Printf("Crawl4AI base URL: %s", cfg.Crawl4AIBaseURL)

	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
