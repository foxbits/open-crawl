package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type CrawlHandler struct {
	*BaseHandler
}

func NewCrawlHandler(crawl4aiBaseURL string, timeout time.Duration) *CrawlHandler {
	return &CrawlHandler{
		BaseHandler: NewBaseHandler(crawl4aiBaseURL, timeout),
	}
}

func (h *CrawlHandler) path() string {
	return "/crawl"
}

func (h *CrawlHandler) operationName() string {
	return "Crawl"
}

func (h *CrawlHandler) getRequestValidator() requestValidator {
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

func (h *CrawlHandler) transformResult(c4Result Crawl4AIStreamResult, reqInterface interface{}) TavilyResult {
	req := reqInterface.(TavilyCrawlRequest)
	return TransformCrawl4AIResult(c4Result, req.IncludeFavicon, req.IncludeImages)
}

func (h *CrawlHandler) logCompletion(requestID string, reqInterface interface{}, resultCount, failedCount int, elapsed time.Duration) {
	req := reqInterface.(TavilyCrawlRequest)
	log.Printf("[DEBUG] Crawl completed: requestID=%s url=%q links_crawled=%d elapsed_ms=%d",
		requestID, req.URL, resultCount, elapsed.Milliseconds())
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
