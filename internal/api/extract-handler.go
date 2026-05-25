package api

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type ExtractHandler struct {
	*BaseHandler
}

func NewExtractHandler(crawl4aiBaseURL string, timeout time.Duration) *ExtractHandler {
	return &ExtractHandler{
		BaseHandler: NewBaseHandler(crawl4aiBaseURL, timeout),
	}
}

func (h *ExtractHandler) path() string {
	return "/extract"
}

func (h *ExtractHandler) operationName() string {
	return "Extract"
}

func (h *ExtractHandler) getRequestValidator() requestValidator {
	return func(r *http.Request) (interface{}, error) {
		var req TavilyExtractRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, &ValidationError{Message: "Invalid JSON body"}
		}

		urls, err := parseURLs(req.URLs)
		if err != nil {
			return nil, &ValidationError{Message: err.Error()}
		}

		if len(urls) == 0 {
			return nil, &ValidationError{Message: "[400] urls field is required"}
		}

		return &extractRequestData{req, urls}, nil
	}
}

func (h *ExtractHandler) transformRequest(reqInterface interface{}) (Crawl4AIRequestBody, []string) {
	data := reqInterface.(*extractRequestData)
	return TavilyExtractRequestToCrawl4AI(data.req, data.urls), data.urls
}

func (h *ExtractHandler) getTimeout(reqInterface interface{}) time.Duration {
	data := reqInterface.(*extractRequestData)
	timeout := h.httpClient.Timeout
	if data.req.Timeout > 0 && time.Duration(data.req.Timeout)*time.Second < timeout {
		timeout = time.Duration(data.req.Timeout) * time.Second
	}
	return timeout
}

func (h *ExtractHandler) processStreamResults(resp *http.Response, reqInterface interface{}) ([]TavilyResult, []FailedResult) {
	_ = reqInterface.(*extractRequestData)
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

		tavilyResult := h.transformResult(c4Result, reqInterface)
		results = append(results, tavilyResult)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Warning: scanner error: %v", err)
	}

	return results, failedResults
}

func (h *ExtractHandler) transformResult(c4Result Crawl4AIStreamResult, reqInterface interface{}) TavilyResult {
	data := reqInterface.(*extractRequestData)
	return TransformCrawl4AIResult(c4Result, data.req.IncludeFavicon, data.req.IncludeImages)
}

func (h *ExtractHandler) logCompletion(requestID string, reqInterface interface{}, resultCount, failedCount int, elapsed time.Duration) {
	log.Printf("[DEBUG] Extract completed: requestID=%s results=%d failed=%d elapsed_ms=%d",
		requestID, resultCount, failedCount, elapsed.Milliseconds())
}

func (h *ExtractHandler) writeResponse(w http.ResponseWriter, reqInterface interface{}, results []TavilyResult, failedResults []FailedResult, elapsed time.Duration, requestID string) {
	data := reqInterface.(*extractRequestData)
	response := TavilyExtractResponse{
		Results:       results,
		FailedResults: failedResults,
		ResponseTime:  elapsed.Seconds(),
		RequestID:     requestID,
	}

	if data.req.IncludeUsage {
		response.Usage = &Usage{Credits: calculateCredits(len(results))}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type extractRequestData struct {
	req  TavilyExtractRequest
	urls []string
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