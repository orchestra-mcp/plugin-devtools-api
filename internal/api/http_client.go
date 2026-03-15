package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var varPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// InterpolateVars replaces {{var}} in s with values from vars.
func InterpolateVars(s string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		key := match[2 : len(match)-2]
		if v, ok := vars[key]; ok {
			return v
		}
		return match
	})
}

const maxResponseBody = 1 << 20 // 1MB

// ExecuteRequest performs an HTTP request with optional variable interpolation.
func ExecuteRequest(req Request, env *Environment) (*Response, error) {
	vars := make(map[string]string)
	if env != nil {
		for k, v := range env.Variables {
			vars[k] = v
		}
	}

	reqURL := InterpolateVars(req.URL, vars)
	body := InterpolateVars(req.Body, vars)

	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	httpReq, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Apply headers
	for k, v := range req.Headers {
		httpReq.Header.Set(InterpolateVars(k, vars), InterpolateVars(v, vars))
	}

	// Set content type from body_type if not already set
	if httpReq.Header.Get("Content-Type") == "" && body != "" {
		switch req.BodyType {
		case "json", "graphql":
			httpReq.Header.Set("Content-Type", "application/json")
		case "xml":
			httpReq.Header.Set("Content-Type", "text/xml")
		case "form":
			httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		case "text":
			httpReq.Header.Set("Content-Type", "text/plain")
		}
	}

	// Apply auth
	if req.Auth != nil {
		applyAuth(httpReq, req.Auth, vars)
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       string(respBody),
		Duration:   duration,
		Size:       int64(len(respBody)),
	}, nil
}

func applyAuth(req *http.Request, auth *AuthConfig, vars map[string]string) {
	switch auth.Type {
	case "bearer":
		token := InterpolateVars(auth.Token, vars)
		req.Header.Set("Authorization", "Bearer "+token)
	case "basic":
		user := InterpolateVars(auth.User, vars)
		pass := InterpolateVars(auth.Pass, vars)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+pass)))
	case "api_key":
		key := InterpolateVars(auth.Key, vars)
		value := InterpolateVars(auth.Value, vars)
		if auth.In == "query" {
			q := req.URL.Query()
			q.Set(key, value)
			req.URL.RawQuery = q.Encode()
		} else {
			req.Header.Set(key, value)
		}
	}
}
