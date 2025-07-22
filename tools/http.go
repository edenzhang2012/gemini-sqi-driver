package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type APIClient struct {
	client  *http.Client
	baseURL *url.URL
}

type RequestOptions struct {
	Method         string
	Path           string // 相对路径, e.g., "/users"
	Headers        map[string]string
	Query          url.Values
	Body           interface{}
	ResponseTarget interface{} //如果指定了ResponseTarget，则会在DO内解析res.Body并自动关闭，否则需要在调用完DO之后手动关闭
}

func NewAPIClient(baseURL string) (*APIClient, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &APIClient{
		client:  &http.Client{}, //默认配置
		baseURL: parsedBaseURL,
	}, nil
}

func (c *APIClient) Do(ctx context.Context, opts *RequestOptions) (*http.Response, error) {
	reqURL := c.baseURL.ResolveReference(&url.URL{Path: opts.Path})

	if opts.Query != nil {
		reqURL.RawQuery = opts.Query.Encode()
	}

	var bodyReader io.Reader
	if opts.Body != nil {
		jsonBytes, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBytes)
		if opts.Headers == nil {
			opts.Headers = make(map[string]string)
		}
		if _, ok := opts.Headers["Content-Type"]; !ok {
			opts.Headers["Content-Type"] = "application/json"
		}
	}

	req, err := http.NewRequestWithContext(ctx, opts.Method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	if opts.ResponseTarget != nil {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(opts.ResponseTarget); err != nil {
			return resp, fmt.Errorf("failed to decode response body: %w", err)
		}
	}

	return resp, nil
}
