package api

import "encoding/json"

type TavilyCrawlRequest struct {
	URL             string   `json:"url"`
	MaxDepth        int      `json:"max_depth,omitempty"`
	MaxBreadth      int      `json:"max_breadth,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	IncludeImages   bool     `json:"include_images,omitempty"`
	Format          string   `json:"format,omitempty"`
	Timeout         int      `json:"timeout,omitempty"`
	SelectPaths     []string `json:"select_paths,omitempty"`
	ExcludePaths    []string `json:"exclude_paths,omitempty"`
	AllowExternal   bool     `json:"allow_external,omitempty"`
	IncludeFavicon  bool     `json:"include_favicon,omitempty"`
	IncludeUsage    bool     `json:"include_usage,omitempty"`
	Instructions    string   `json:"instructions,omitempty"`
	ChunksPerSource int      `json:"chunks_per_source,omitempty"`
	SelectDomains   []string `json:"select_domains,omitempty"`
	ExcludeDomains  []string `json:"exclude_domains,omitempty"`
	ExtractDepth    string   `json:"extract_depth,omitempty"`
}

type TavilyCrawlResponse struct {
	BaseURL      string         `json:"base_url"`
	Results      []TavilyResult `json:"results"`
	ResponseTime float64        `json:"response_time"`
	Usage        *Usage         `json:"usage,omitempty"`
	RequestID    string         `json:"request_id"`
}

type Usage struct {
	Credits int `json:"credits"`
}

type TavilyResult struct {
	URL        string      `json:"url"`
	RawContent string      `json:"raw_content"`
	Favicon    string      `json:"favicon,omitempty"`
	Images     []ImageInfo `json:"images,omitempty"`
}

type ImageInfo struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type Crawl4AIStreamResult struct {
	URL          string          `json:"url"`
	Success      bool            `json:"success"`
	Status       string          `json:"status,omitempty"`
	Markdown     MarkdownContent `json:"markdown"`
	Media        MediaContent    `json:"media"`
	Links        LinksContent    `json:"links"`
	Metadata     MetadataContent `json:"metadata"`
	ErrorMessage string          `json:"error_message,omitempty"`
	StatusCode   int             `json:"status_code,omitempty"`
}

type MarkdownContent struct {
	RawMarkdown string `json:"raw_markdown"`
	FItMarkdown string `json:"fit_markdown,omitempty"`
}

type MediaContent struct {
	Images []MediaImage `json:"images"`
}

type MediaImage struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

type LinksContent struct {
	Favicon string `json:"favicon,omitempty"`
}

type MetadataContent struct {
	Favicon string `json:"favicon,omitempty"`
}

type Crawl4AIRequestBody struct {
	URLs          []string      `json:"urls"`
	BrowserConfig BrowserConfig `json:"browser_config"`
	CrawlerConfig CrawlerConfig `json:"crawler_config"`
}

type BrowserConfig struct {
	Headless bool `json:"headless"`
}

type CrawlerConfig struct {
	Params CrawlerParams `json:"params"`
}

type CrawlerParams struct {
	Stream            bool               `json:"stream"`
	CacheMode         string             `json:"cache_mode"`
	MarkdownGenerator string             `json:"markdown_generator,omitempty"`
	DeepCrawlStrategy *DeepCrawlStrategy `json:"deep_crawl_strategy,omitempty"`
}

type DeepCrawlStrategy struct {
	Type            string      `json:"type"`
	MaxDepth        int         `json:"max_depth,omitempty"`
	MaxPages        int         `json:"max_pages,omitempty"`
	IncludeExternal bool        `json:"include_external"`
	FilterChain     FilterChain `json:"filter_chain,omitempty"`
}

type FilterChain struct {
	Filters []Filter `json:"filters,omitempty"`
}

type Filter struct {
	Type    string   `json:"type"`
	Pattern []string `json:"pattern,omitempty"`
}

type ErrorDetail struct {
	Error string `json:"error"`
}

type ErrorResponse struct {
	Detail ErrorDetail `json:"detail"`
}

type TavilyExtractRequest struct {
	URLs            json.RawMessage `json:"urls"`
	Query           string          `json:"query,omitempty"`
	ChunksPerSource int             `json:"chunks_per_source,omitempty"`
	ExtractDepth    string          `json:"extract_depth,omitempty"`
	IncludeImages   bool            `json:"include_images,omitempty"`
	IncludeFavicon  bool            `json:"include_favicon,omitempty"`
	Format          string          `json:"format,omitempty"`
	Timeout         float64         `json:"timeout,omitempty"`
	IncludeUsage    bool            `json:"include_usage,omitempty"`
}

type TavilyExtractResponse struct {
	Results       []TavilyResult `json:"results"`
	FailedResults []FailedResult `json:"failed_results"`
	ResponseTime  float64        `json:"response_time"`
	Usage         *Usage         `json:"usage,omitempty"`
	RequestID     string         `json:"request_id"`
}

type FailedResult struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}
