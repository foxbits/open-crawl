package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CrawlHandler struct {
	crawl4aiBaseURL string
	httpClient      *http.Client
}

func NewCrawlHandler(crawl4aiBaseURL string, timeout time.Duration) *CrawlHandler {
	return &CrawlHandler{
		crawl4aiBaseURL: crawl4aiBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *CrawlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/crawl" {
		h.handleCrawl(w, r)
		return
	}
	http.NotFound(w, r)
}

func (h *CrawlHandler) handleCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TavilyCrawlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.URL == "" {
		sendError(w, http.StatusBadRequest, "[400] No starting url provided")
		return
	}

	_ = r.Header.Get("Authorization")

	requestID := generateRequestID()
	startTime := time.Now()

	crawlReq := TavilyRequestToCrawl4AI(req)

	jsonReq, err := json.Marshal(crawlReq)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to transform request")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.httpClient.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.crawl4aiBaseURL+"/crawl/stream", strings.NewReader(string(jsonReq)))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			sendError(w, http.StatusGatewayTimeout, "Crawl4AI upstream timeout")
			return
		}
		sendError(w, http.StatusBadGateway, "Crawl4AI server unreachable")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		sendError(w, resp.StatusCode, string(body))
		return
	}

	var results []TavilyResult
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var c4Result Crawl4AIStreamResult
		if err := json.Unmarshal(line, &c4Result); err != nil {
			log.Printf("Warning: failed to parse NDJSON line: %v", err)
			continue
		}

		if !c4Result.Success {
			log.Printf("Crawl failed for %s: %s", c4Result.URL, c4Result.ErrorMessage)
			continue
		}

		tavilyResult := TransformCrawl4AIResult(c4Result, req.IncludeFavicon, req.IncludeImages)
		results = append(results, tavilyResult)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Warning: scanner error: %v", err)
	}

	elapsed := time.Since(startTime)
	response := BuildFinalResponse(req.URL, results, elapsed, requestID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Detail: ErrorDetail{Error: message},
	})
}

func generateRequestID() string {
	return uuid.New().String()
}