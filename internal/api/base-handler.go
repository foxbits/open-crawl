package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type HandlerConfig struct {
	baseURL string
	client  *http.Client
}

func NewHandlerConfig(crawl4aiBaseURL string, timeout time.Duration) HandlerConfig {
	return HandlerConfig{
		baseURL: crawl4aiBaseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c HandlerConfig) getBaseURL() string {
	return c.baseURL
}

func (c HandlerConfig) getClient() *http.Client {
	return c.client
}

func (c HandlerConfig) getTimeout(req interface{}) time.Duration {
	return c.client.Timeout
}

func handleFailedResult(h Handler, c4Result Crawl4AIStreamResult) {
	url := c4Result.URL
	if url == "" {
		url = "(unknown URL)"
	}
	log.Printf("%s failed for %s: %s", h.operationName(), url, c4Result.ErrorMessage)
}

func logCompletion(h Handler, requestID string, resultCount, failedCount int, elapsed time.Duration) {
	log.Printf("[DEBUG] %s completed: requestID=%s results=%d failed=%d elapsed_ms=%d",
		h.operationName(), requestID, resultCount, failedCount, elapsed.Milliseconds())
}

type Handler interface {
	getBaseURL() string
	getClient() *http.Client
	path() string
	operationName() string
	getRequestValidator() func(*http.Request) (interface{}, error)
	transformRequest(req interface{}) (Crawl4AIRequestBody, []string)
	getTimeout(req interface{}) time.Duration
	processStreamResults(resp *http.Response, req interface{}) ([]TavilyResult, []FailedResult)
	transformResult(c4Result Crawl4AIStreamResult, req interface{}) TavilyResult
	writeResponse(w http.ResponseWriter, req interface{}, results []TavilyResult, failedResults []FailedResult, elapsed time.Duration, requestID string)
}

func ServeHTTP(h Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && h.path() == r.URL.Path {
		handleRequest(h, w, r)
		return
	}
	http.NotFound(w, r)
}

func handleRequest(h Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	validator := h.getRequestValidator()
	req, err := validator(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = r.Header.Get("Authorization")

	requestID := uuid.New().String()
	startTime := time.Now()

	log.Printf("[DEBUG] %s started: requestID=%s", h.operationName(), requestID)

	crawlReq, _ := h.transformRequest(req)
	log.Printf("[DEBUG] %s Crawl4AI request: requestID=%s crawl4ai_params=%+v", h.operationName(), requestID, crawlReq)

	jsonReq, err := json.Marshal(crawlReq)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to transform request")
		return
	}

	timeout := h.getTimeout(req)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.getBaseURL()+"/crawl/stream", strings.NewReader(string(jsonReq)))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.getClient().Do(httpReq)
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

	results, failedResults := h.processStreamResults(resp, req)

	elapsed := time.Since(startTime)
	logCompletion(h, requestID, len(results), len(failedResults), elapsed)

	h.writeResponse(w, req, results, failedResults, elapsed, requestID)
}

func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Detail: ErrorDetail{Error: message},
	})
}
