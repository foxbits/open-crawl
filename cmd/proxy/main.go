package main

import (
	"log"
	"net/http"

	"github.com/foxbits/open-crawl/internal/api"
	"github.com/foxbits/open-crawl/internal/config"
)

func main() {
	cfg := config.Load()

	handler := api.NewCrawlHandler(cfg.Crawl4AIAPIURL, cfg.RequestTimeout)
	extractHandler := api.NewExtractHandler(cfg.Crawl4AIAPIURL, cfg.RequestTimeout)
	healthHandler := api.NewHealthHandler(cfg.Crawl4AIAPIURL, cfg.RequestTimeout)

	mux := http.NewServeMux()
	mux.Handle("/crawl", handler)
	mux.Handle("/extract", extractHandler)
	mux.Handle("/health", healthHandler)

	log.Printf("Starting Open Crawl Proxy on %s", cfg.ListenAddr)
	log.Printf("Crawl4AI API URL: %s", cfg.Crawl4AIAPIURL)

	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
