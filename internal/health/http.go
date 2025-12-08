package health

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

type HTTPChecker struct {
	name           string
	url            string
	method         string
	expectedStatus int
	client         *http.Client
}

func NewHTTPChecker(name string, config map[string]interface{}) (*HTTPChecker, error) {
	url, ok := config["url"].(string)
	if !ok {
		return nil, fmt.Errorf("http checker: missing or invalid 'url' field")
	}

	method := "GET"
	if m, ok := config["method"].(string); ok {
		method = m
	}

	expectedStatus := 200
	if status, ok := config["expected_status"]; ok {
		switch v := status.(type) {
		case int:
			expectedStatus = v
		case float64:
			expectedStatus = int(v)
		}
	}

	insecureSkipVerify := false
	if skip, ok := config["insecure_skip_verify"].(bool); ok {
		insecureSkipVerify = skip
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &HTTPChecker{
		name:           name,
		url:            url,
		method:         method,
		expectedStatus: expectedStatus,
		client:         client,
	}, nil
}

func (c *HTTPChecker) Name() string {
	return c.name
}

func (c *HTTPChecker) Type() string {
	return "http"
}

func (c *HTTPChecker) Check(ctx context.Context) CheckResult {
	req, err := http.NewRequestWithContext(ctx, c.method, c.url, nil)
	if err != nil {
		return CheckResultFailure
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return CheckResultFailure
	}
	defer resp.Body.Close()

	if resp.StatusCode == c.expectedStatus {
		return CheckResultSuccess
	}

	return CheckResultFailure
}
