package api

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type CrawlHandler struct {
	HandlerConfig
}

func NewCrawlHandler(crawl4aiBaseURL string, timeout time.Duration) *CrawlHandler {
	return &CrawlHandler{
		HandlerConfig: NewHandlerConfig(crawl4aiBaseURL, timeout),
	}
}

func (h *CrawlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ServeHTTP(h, w, r)
}

func (h *CrawlHandler) path() string {
	return "/crawl"
}

func (h *CrawlHandler) operationName() string {
	return "Crawl"
}

func (h *CrawlHandler) getRequestValidator() func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var req TavilyCrawlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, &ValidationError{Message: "Invalid JSON body"}
		}
		if req.URL == "" {
			return nil, &ValidationError{Message: "[400] No starting url provided"}
		}
		return req, nil
	}
}

func (h *CrawlHandler) transformRequest(reqInterface interface{}) (Crawl4AIRequestBody, []string) {
	req := reqInterface.(TavilyCrawlRequest)
	return TavilyRequestToCrawl4AI(req), nil
}



func (h *CrawlHandler) processStreamResults(resp *http.Response, reqInterface interface{}) ([]TavilyResult, []FailedResult) {
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

		log.Printf("DEBUG Crawl crawl4ai result streamed: url=%q success=%v completed=%q error=%q", c4Result.URL, c4Result.Success, c4Result.Status, c4Result.ErrorMessage)

		if !c4Result.Success && c4Result.Status != "completed" {
			handleFailedResult(h, c4Result)
			continue
		}

		tavilyResult := h.transformResult(c4Result, reqInterface)
		results = append(results, tavilyResult)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Warning: scanner error: %v", err)
	}

	return results, nil
}

func (h *CrawlHandler) transformResult(c4Result Crawl4AIStreamResult, reqInterface interface{}) TavilyResult {
	req := reqInterface.(TavilyCrawlRequest)
	return TransformCrawl4AIResult(c4Result, req.IncludeFavicon, req.IncludeImages)
}

func (h *CrawlHandler) writeResponse(w http.ResponseWriter, reqInterface interface{}, results []TavilyResult, _ []FailedResult, elapsed time.Duration, requestID string) {
	req := reqInterface.(TavilyCrawlRequest)
	response := BuildFinalResponse(req.URL, results, elapsed, requestID)

	if req.IncludeUsage {
		response.Usage = &Usage{Credits: 1}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}