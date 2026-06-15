package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Provider interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type OpenAICompatibleClient struct {
	endpoint   string
	apiKey     string
	model      string
	dimensions int
	httpClient *http.Client
}

func NewOpenAICompatibleClient(
	baseURL string,
	apiKey string,
	model string,
	dimensions int,
	timeout time.Duration,
) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		endpoint:   strings.TrimRight(baseURL, "/") + "/embeddings",
		apiKey:     apiKey,
		model:      model,
		dimensions: dimensions,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *OpenAICompatibleClient) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(map[string]any{
		"model":      c.model,
		"input":      inputs,
		"dimensions": c.dimensions,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.apiKey)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request embeddings: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding endpoint returned %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decoded.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding response returned %d vectors for %d inputs", len(decoded.Data), len(inputs))
	}
	sort.Slice(decoded.Data, func(i, j int) bool {
		return decoded.Data[i].Index < decoded.Data[j].Index
	})
	embeddings := make([][]float32, len(decoded.Data))
	for i, item := range decoded.Data {
		if item.Index != i {
			return nil, fmt.Errorf("embedding response index %d is out of sequence at position %d", item.Index, i)
		}
		if len(item.Embedding) != c.dimensions {
			return nil, fmt.Errorf("embedding %d has %d dimensions, want %d", i, len(item.Embedding), c.dimensions)
		}
		embeddings[i] = item.Embedding
	}
	return embeddings, nil
}
