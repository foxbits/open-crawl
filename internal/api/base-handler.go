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

type BaseHandler struct {
	crawl4aiBaseURL string
	httpClient      *http.Client
}

func NewBaseHandler(crawl4aiBaseURL string, timeout time.Duration) *BaseHandler {
	return &BaseHandler{
		crawl4aiBaseURL: crawl4aiBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *BaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		path := r.URL.Path
		if h.shouldHandle(path) {
			h.handle(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func (h *BaseHandler) shouldHandle(path string) bool {
	return h.path() == path
}

func (h *BaseHandler) path() string {
	return ""
}

type requestValidator func(r *http.Request) (interface{}, error)

func (h *BaseHandler) handle(w http.ResponseWriter, r *http.Request) {
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

	results, failedResults := h.processStreamResults(resp, req)

	elapsed := time.Since(startTime)
	h.logCompletion(requestID, req, len(results), len(failedResults), elapsed)

	h.writeResponse(w, req, results, failedResults, elapsed, requestID)
}

func (h *BaseHandler) getRequestValidator() requestValidator {
	return nil
}

func (h *BaseHandler) transformRequest(req interface{}) (Crawl4AIRequestBody, []string) {
	return Crawl4AIRequestBody{}, nil
}

func (h *BaseHandler) getTimeout(req interface{}) time.Duration {
	return h.httpClient.Timeout
}

func (h *BaseHandler) processStreamResults(resp *http.Response, req interface{}) ([]TavilyResult, []FailedResult) {
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

		log.Printf("DEBUG %s crawl4ai result streamed: url=%q success=%v completed=%q error=%q", h.operationName(), c4Result.URL, c4Result.Success, c4Result.Status, c4Result.ErrorMessage)

		if !c4Result.Success && c4Result.Status != "completed" {
			h.handleFailedResult(c4Result)
			continue
		}

		tavilyResult := h.transformResult(c4Result, req)
		results = append(results, tavilyResult)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Warning: scanner error: %v", err)
	}

	return results, nil
}

func (h *BaseHandler) handleFailedResult(c4Result Crawl4AIStreamResult) {
	url := c4Result.URL
	if url == "" {
		url = "(unknown URL)"
	}
	log.Printf("%s failed for %s: %s", h.operationName(), url, c4Result.ErrorMessage)
}

func (h *BaseHandler) transformResult(c4Result Crawl4AIStreamResult, req interface{}) TavilyResult {
	return TransformCrawl4AIResult(c4Result, false, false)
}

func (h *BaseHandler) operationName() string {
	return "Crawl"
}

func (h *BaseHandler) logCompletion(requestID string, req interface{}, resultCount, failedCount int, elapsed time.Duration) {
	log.Printf("[DEBUG] %s completed: requestID=%s results=%d elapsed_ms=%d",
		h.operationName(), requestID, resultCount, elapsed.Milliseconds())
}

func (h *BaseHandler) writeResponse(w http.ResponseWriter, req interface{}, results []TavilyResult, failedResults []FailedResult, elapsed time.Duration, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Detail: ErrorDetail{Error: message},
	})
}
