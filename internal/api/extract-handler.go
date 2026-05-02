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
)

type ExtractHandler struct {
	crawl4aiBaseURL string
	httpClient      *http.Client
}

func NewExtractHandler(crawl4aiBaseURL string, timeout time.Duration) *ExtractHandler {
	return &ExtractHandler{
		crawl4aiBaseURL: crawl4aiBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *ExtractHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/extract" {
		h.handleExtract(w, r)
		return
	}
	http.NotFound(w, r)
}

func (h *ExtractHandler) handleExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TavilyExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	urls, err := parseURLs(req.URLs)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(urls) == 0 {
		sendError(w, http.StatusBadRequest, "[400] urls field is required")
		return
	}

	_ = r.Header.Get("Authorization")

	requestID := generateRequestID()
	startTime := time.Now()

	log.Printf("[DEBUG] Extract started: requestID=%s url_count=%d params=%+v", requestID, len(urls), req)

	crawlReq := TavilyExtractRequestToCrawl4AI(req, urls)
	log.Printf("[DEBUG] Extract Crawl4AI request: requestID=%s crawl4ai_params=%+v", requestID, crawlReq)

	jsonReq, err := json.Marshal(crawlReq)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to transform request")
		return
	}

	timeout := h.httpClient.Timeout
	if req.Timeout > 0 && time.Duration(req.Timeout)*time.Second < timeout {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
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
	var failedResults []FailedResult
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

		log.Printf("DEBUG extract crawl4ai result streamed: url=%q success=%v completed=%q error=%q", c4Result.URL, c4Result.Success, c4Result.Status, c4Result.ErrorMessage)

		if !c4Result.Success && c4Result.Status != "completed" {
			failedResults = append(failedResults, FailedResult{
				URL:   c4Result.URL,
				Error: c4Result.ErrorMessage,
			})
			continue
		}

		tavilyResult := TransformCrawl4AIResult(c4Result, req.IncludeFavicon, req.IncludeImages)
		results = append(results, tavilyResult)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Warning: scanner error: %v", err)
	}

	elapsed := time.Since(startTime)
	log.Printf("[DEBUG] Extract completed: requestID=%s results=%d failed=%d elapsed_ms=%d",
		requestID, len(results), len(failedResults), elapsed.Milliseconds())

	response := TavilyExtractResponse{
		Results:       results,
		FailedResults: failedResults,
		ResponseTime:  elapsed.Seconds(),
		RequestID:     requestID,
	}

	if req.IncludeUsage {
		response.Usage = &Usage{Credits: calculateCredits(len(results))}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func parseURLs(raw json.RawMessage) ([]string, error) {
	if raw == nil {
		return nil, nil
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil && single != "" {
		return []string{single}, nil
	}

	var multiple []string
	if err := json.Unmarshal(raw, &multiple); err == nil {
		return multiple, nil
	}

	return nil, nil
}

func calculateCredits(successful int) int {
	return (successful + 4) / 5
}
