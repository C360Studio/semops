package semconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultUserAgent = "semops-semconnect-read-side-egress/0.1"

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ExecuteOption func(*executeConfig)

type executeConfig struct {
	client    HTTPDoer
	headers   http.Header
	userAgent string
}

type ExecuteResult struct {
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt time.Time       `json:"finished_at"`
	Requests   []RequestResult `json:"requests"`
}

type RequestResult struct {
	Resource     string `json:"resource"`
	SourceID     string `json:"source_id"`
	SemConnectID string `json:"semconnect_id"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	StatusCode   int    `json:"status_code"`
	Location     string `json:"location,omitempty"`
}

func WithHTTPClient(client HTTPDoer) ExecuteOption {
	return func(cfg *executeConfig) {
		cfg.client = client
	}
}

func WithHeader(key, value string) ExecuteOption {
	return func(cfg *executeConfig) {
		if cfg.headers == nil {
			cfg.headers = make(http.Header)
		}
		cfg.headers.Add(key, value)
	}
}

func WithUserAgent(userAgent string) ExecuteOption {
	return func(cfg *executeConfig) {
		cfg.userAgent = userAgent
	}
}

func ExecuteReadSidePlan(ctx context.Context, baseURL string, plan ReadSidePlan, options ...ExecuteOption) (ExecuteResult, error) {
	cfg := executeConfig{
		client:    http.DefaultClient,
		headers:   make(http.Header),
		userAgent: defaultUserAgent,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	if cfg.client == nil {
		return ExecuteResult{}, fmt.Errorf("http client required")
	}
	if strings.TrimSpace(baseURL) == "" {
		return ExecuteResult{}, fmt.Errorf("base URL required")
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("parse base URL: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return ExecuteResult{}, fmt.Errorf("base URL must include scheme and host")
	}

	result := ExecuteResult{
		StartedAt: time.Now().UTC(),
		Requests:  make([]RequestResult, 0, len(plan.Requests)),
	}
	for _, planned := range plan.Requests {
		requestResult, err := executeRequest(ctx, cfg, base, planned)
		result.Requests = append(result.Requests, requestResult)
		if err != nil {
			result.FinishedAt = time.Now().UTC()
			return result, err
		}
	}
	result.FinishedAt = time.Now().UTC()
	return result, nil
}

func executeRequest(ctx context.Context, cfg executeConfig, base *url.URL, planned Request) (RequestResult, error) {
	target, err := resolvePath(base, planned.Path)
	if err != nil {
		return RequestResult{Resource: planned.Resource, SourceID: planned.SourceID, SemConnectID: planned.SemConnectID}, err
	}
	body, err := json.Marshal(planned.Body)
	if err != nil {
		return RequestResult{Resource: planned.Resource, SourceID: planned.SourceID, SemConnectID: planned.SemConnectID}, fmt.Errorf("marshal %s %q: %w", planned.Resource, planned.SourceID, err)
	}
	req, err := http.NewRequestWithContext(ctx, planned.Method, target.String(), bytes.NewReader(body))
	if err != nil {
		return RequestResult{Resource: planned.Resource, SourceID: planned.SourceID, SemConnectID: planned.SemConnectID}, fmt.Errorf("build %s %q request: %w", planned.Resource, planned.SourceID, err)
	}
	req.Header.Set("Content-Type", planned.ContentType)
	req.Header.Set("Accept", MediaJSON)
	req.Header.Set("User-Agent", cfg.userAgent)
	for key, values := range cfg.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := cfg.client.Do(req)
	if err != nil {
		return RequestResult{
			Resource:     planned.Resource,
			SourceID:     planned.SourceID,
			SemConnectID: planned.SemConnectID,
			Method:       planned.Method,
			Path:         planned.Path,
		}, fmt.Errorf("send %s %q to SemConnect: %w", planned.Resource, planned.SourceID, err)
	}
	defer resp.Body.Close()

	requestResult := RequestResult{
		Resource:     planned.Resource,
		SourceID:     planned.SourceID,
		SemConnectID: planned.SemConnectID,
		Method:       planned.Method,
		Path:         planned.Path,
		StatusCode:   resp.StatusCode,
		Location:     resp.Header.Get("Location"),
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return requestResult, fmt.Errorf("SemConnect %s %s returned %d: %s",
			planned.Method, planned.Path, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return requestResult, nil
}

func resolvePath(base *url.URL, path string) (*url.URL, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("request path required")
	}
	rel, err := url.Parse(strings.TrimLeft(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse request path %q: %w", path, err)
	}
	root := *base
	if !strings.HasSuffix(root.Path, "/") {
		root.Path += "/"
	}
	return root.ResolveReference(rel), nil
}
