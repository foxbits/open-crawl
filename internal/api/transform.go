package api

import (
	"time"
)

func TavilyRequestToCrawl4AI(req TavilyCrawlRequest) Crawl4AIRequestBody {
	body := Crawl4AIRequestBody{
		URLs:          []string{req.URL},
		BrowserConfig: BrowserConfig{},
		CrawlerConfig: CrawlerConfig{
			Params: CrawlerParams{},
		},
	}

	if req.Format == "markdown" || req.Format == "" {
		body.CrawlerConfig.Params.MarkdownGenerator = "generate_markdown"
	}

	if needsDeepCrawl(req) {
		body.CrawlerConfig.Params.DeepCrawlStrategy = buildDeepCrawlStrategy(req)
	}

	return body
}

func needsDeepCrawl(req TavilyCrawlRequest) bool {
	if req.MaxDepth > 1 {
		return true
	}
	if req.Limit > 0 {
		return true
	}
	if len(req.SelectPaths) > 0 {
		return true
	}
	if len(req.ExcludePaths) > 0 {
		return true
	}
	if len(req.SelectDomains) > 0 {
		return true
	}
	if len(req.ExcludeDomains) > 0 {
		return true
	}
	return false
}

func buildDeepCrawlStrategy(req TavilyCrawlRequest) *DeepCrawlStrategy {
	strategy := &DeepCrawlStrategy{
		Type: "BFSDeepCrawlStrategy",
	}

	if req.MaxDepth > 1 {
		strategy.MaxDepth = req.MaxDepth
	} else {
		strategy.MaxDepth = 1
	}

	if req.Limit > 0 {
		strategy.MaxPages = req.Limit
	} else {
		strategy.MaxPages = 50
	}

	strategy.IncludeExternal = req.AllowExternal

	var filters []Filter

	for _, pattern := range req.SelectPaths {
		filters = append(filters, Filter{
			Type:    "URLPatternFilter",
			Pattern: []string{pattern},
		})
	}

	for _, pattern := range req.ExcludePaths {
		filters = append(filters, Filter{
			Type:    "URLPatternFilter",
			Pattern: []string{pattern},
		})
	}

	if len(req.SelectDomains) > 0 {
		filters = append(filters, Filter{
			Type:    "DomainFilter",
			Pattern: req.SelectDomains,
		})
	}

	if len(req.ExcludeDomains) > 0 {
		filters = append(filters, Filter{
			Type:    "DomainFilter",
			Pattern: req.ExcludeDomains,
		})
	}

	if len(filters) > 0 {
		strategy.FilterChain = FilterChain{
			Filters: filters,
		}
	}

	return strategy
}

func TransformCrawl4AIResult(c4Result Crawl4AIStreamResult, includeFavicon, includeImages bool) TavilyResult {
	result := TavilyResult{
		URL: c4Result.URL,
	}

	if c4Result.Markdown.FItMarkdown != "" {
		result.RawContent = c4Result.Markdown.FItMarkdown
	} else {
		result.RawContent = c4Result.Markdown.RawMarkdown
	}

	if includeFavicon {
		if c4Result.Metadata.Favicon != "" {
			result.Favicon = c4Result.Metadata.Favicon
		} else if c4Result.Links.Favicon != "" {
			result.Favicon = c4Result.Links.Favicon
		}
	}

	if includeImages && len(c4Result.Media.Images) > 0 {
		result.Images = make([]ImageInfo, 0, len(c4Result.Media.Images))
		for _, img := range c4Result.Media.Images {
			result.Images = append(result.Images, ImageInfo{
				URL:         img.Src,
				Description: img.Alt,
			})
		}
	}

	return result
}

func BuildFinalResponse(baseURL string, results []TavilyResult, elapsed time.Duration, requestID string) TavilyCrawlResponse {
	return TavilyCrawlResponse{
		BaseURL:      baseURL,
		Results:      results,
		ResponseTime: elapsed.Seconds(),
		Usage:        &Usage{Credits: 1},
		RequestID:    requestID,
	}
}

func TavilyExtractRequestToCrawl4AI(req TavilyExtractRequest, urls []string) Crawl4AIRequestBody {
	body := Crawl4AIRequestBody{
		URLs:          urls,
		BrowserConfig: BrowserConfig{},
		CrawlerConfig: CrawlerConfig{
			Params: CrawlerParams{},
		},
	}

	return body
}
