package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthHandler struct {
	crawl4aiBaseURL string
	httpClient      *http.Client
}

func NewHealthHandler(crawl4aiBaseURL string, timeout time.Duration) *HealthHandler {
	return &HealthHandler{
		crawl4aiBaseURL: crawl4aiBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/health" {
		h.handleHealth(w, r)
		return
	}
	http.NotFound(w, r)
}

func (h *HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	type HealthStatus struct {
		Status string `json:"status"`
	}

	response := struct {
		API      HealthStatus `json:"api"`
		Crawl4AI HealthStatus `json:"crawl4ai"`
	}{
		API:      HealthStatus{Status: "ok"},
		Crawl4AI: HealthStatus{Status: "ok"},
	}

	httpReq, err := http.NewRequest(http.MethodGet, h.crawl4aiBaseURL+"/health", nil)
	if err != nil {
		response.Crawl4AI.Status = "unreachable"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		response.Crawl4AI.Status = "unreachable"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response.Crawl4AI.Status = "not_ok"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}