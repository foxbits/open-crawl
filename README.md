# Open Crawl Proxy

A drop-in Go proxy for Tavily's `POST /crawl` endpoint. It forwards requests to a self-hosted Crawl4AI server, streams results, and returns them in Tavily-compatible format.

**No authentication** — Tavily API keys are accepted (and ignored). No API key is required.

## Quick Start

```bash
docker build -t foxbits/open-crawl .
docker run -p 8080:8080 -e CRAWL4AI_BASE_URL=http://host.docker.internal:11235 open-crawl
```

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `CRAWL4AI_BASE_URL` | `http://localhost:11235` | Crawl4AI REST API base URL |
| `LISTEN_ADDR` | `:8080` | Proxy HTTP listen address |
| `REQUEST_TIMEOUT` | `150s` | Maximum time for a single crawl |

## Endpoints

### `POST /extract`

Extract raw web page content from one or more specified URLs. Acts as a single-page proxy for Crawl4AI (no crawling breadth/depth parameters).

**Request Body**

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `urls` | string \| string[] | **Yes** | — | One or more URLs to extract content from (max 20) |
| `query` | string | No | — | Accepted for API compatibility; not yet used for reranking |
| `chunks_per_source` | integer | No | — | Accepted for API compatibility; not yet used for chunking |
| `extract_depth` | string | No | `basic` | `basic` or `advanced` |
| `include_images` | boolean | No | `false` | Include per-result `images` array |
| `include_favicon` | boolean | No | `false` | Include favicon URL per result |
| `format` | string | No | `markdown` | `markdown` or `text` |
| `timeout` | number | No | — | Max seconds to wait (1.0–60.0) |
| `include_usage` | boolean | No | `false` | Include credit usage info |

**Response Body** (`application/json`)

| Field | Type | Description |
|---|---|---|
| `results` | array | Extracted content per successful URL |
| `results[].url` | string | Extracted URL |
| `results[].raw_content` | string | Full content of the page in markdown or text format |
| `results[].favicon` | string | Favicon URL (if `include_favicon` was `true`) |
| `results[].images` | array | Image objects extracted from the page (if `include_images` was `true`). Each item: `{"url": "...", "description": "..."}` |
| `failed_results` | array | URLs that could not be processed; each entry has `url` and `error` |
| `response_time` | float | Time in seconds to complete the request |
| `usage` | object | Credit usage info (only if `include_usage` was `true`) |
| `request_id` | string | Unique request ID |

**Example Request**

```bash
curl -X POST http://localhost:8080/extract \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer tvly-ignored" \
  -d '{
    "urls": ["https://en.wikipedia.org/wiki/Artificial_intelligence"],
    "include_images": true,
    "include_favicon": true
  }'
```

**Example Response**

```json
{
  "results": [
    {
      "url": "https://en.wikipedia.org/wiki/Artificial_intelligence",
      "raw_content": "# Artificial intelligence\n\nArtificial intelligence (AI), in its broadest sense...",
      "favicon": "https://en.wikipedia.org/static/favicon/wikipedia.ico",
      "images": [
        {"url": "https://upload.wikimedia.org/wikipedia/commons/thumb/...", "description": "AI concept"}
      ]
    }
  ],
  "failed_results": [],
  "response_time": 1.23,
  "usage": {"credits": 1},
  "request_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

### `POST /crawl`

Crawl a website and return extracted content in Tavily-compatible format.

**Request Body**

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | **Yes** | — | Root URL to start crawling from |
| `max_depth` | integer | No | `1` | Maximum crawl depth |
| `max_breadth` | integer | No | `20` | Links to follow per level |
| `limit` | integer | No | `50` | Total links to process |
| `select_paths` | string[] | No | — | Regex patterns to include |
| `exclude_paths` | string[] | No | — | Regex patterns to exclude |
| `allow_external` | boolean | No | `true` | Include external domain links in results |
| `include_images` | boolean | No | `false` | Include per-result `images` array |
| `include_favicon` | boolean | No | `false` | Include favicon URL per result |
| `extract_depth` | string | No | `basic` | `basic` or `advanced` |
| `format` | string | No | `markdown` | `markdown` or `text` |
| `timeout` | number | No | `150` | Max seconds to wait |
| `include_usage` | boolean | No | `false` | Include credit usage info |

**Response Body** (`application/json`)

| Field | Type | Description |
|---|---|---|
| `base_url` | string | The base URL that was crawled |
| `results` | array | Extracted content per crawled URL |
| `results[].url` | string | Crawled URL |
| `results[].raw_content` | string | Full markdown content of the page, ready for LLM consumption |
| `results[].favicon` | string | Favicon URL (if `include_favicon` was `true`) |
| `results[].images` | array | Image objects extracted from the page (if `include_images` was `true`). Each item: `{"url": "...", "description": "..."}` |
| `response_time` | float | Time in seconds to complete the request |
| `usage` | object | Credit usage info placeholder (always `1` credit) |
| `request_id` | string | Unique request ID |

**Example Request**

```bash
curl -X POST http://localhost:8080/crawl \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer tvly-ignored" \
  -d '{
    "url": "https://docs.tavily.com",
    "max_depth": 1,
    "include_images": true,
    "include_favicon": true
  }'
```

**Example Response**

```json
{
  "base_url": "https://docs.tavily.com",
  "results": [
    {
      "url": "https://docs.tavily.com/documentation/api-reference/endpoint/crawl",
      "raw_content": "# Tavily Crawl\n\nPOST /crawl is a graph-based traversal tool...",
      "favicon": "https://mintlify.s3-us-west-1.amazonaws.com/tavilyai/_generated/favicon/apple-touch-icon.png",
      "images": [
        {"url": "https://mintlify.s3.us-west-1.amazonaws.com/tavilyai/logo/light.svg", "description": "Tavily light logo"}
      ]
    }
  ],
  "response_time": 3.45,
  "usage": {"credits": 1},
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Error Responses

All errors follow Tavily's `{"detail": {"error": "..."}}` format:

| Status Code | Cause |
|---|---|
| **400** | Missing `url` or invalid parameters |
| **502** | Crawl4AI server unreachable |
| **504** | Crawl4AI upstream timeout |
| **500** | Internal proxy error |

## How It Works

1. **Accepts** a Tavily `POST /crawl` request
2. **Ignores** authentication headers
3. **Transforms** the Tavily request into Crawl4AI `POST /crawl/stream` format with `stream: true`
4. **Streams** NDJSON lines from Crawl4AI as each page finishes crawling
5. **Transforms** each Crawl4AI result into Tavily-compatible `TavilyResult`
6. **Returns** a single JSON response containing all results
