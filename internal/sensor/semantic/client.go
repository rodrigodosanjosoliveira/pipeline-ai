package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type sanitizedError struct {
	err error
	msg string
}

func (s *sanitizedError) Error() string {
	return s.msg
}

func (s *sanitizedError) Unwrap() error {
	return s.err
}

func sanitizeErr(err error, apiKey string) error {
	if err == nil {
		return nil
	}
	if apiKey == "" || !strings.Contains(err.Error(), apiKey) {
		return err
	}
	return &sanitizedError{
		err: err,
		msg: strings.ReplaceAll(err.Error(), apiKey, "[REDACTED]"),
	}
}

// Client é o cliente para o sensor semântico calibrado com Jev.
type Client struct {
	cfg        ClientConfig
	httpClient *http.Client
}

// NewClient cria uma nova instância de Client com a configuração especificada.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// IsConfigured verifica se o cliente possui endpoint e credenciais configuradas.
func (c *Client) IsConfigured() bool {
	return strings.TrimSpace(c.cfg.Endpoint) != "" && strings.TrimSpace(c.cfg.APIKey) != ""
}

// CheckAvailable verifica a disponibilidade e saúde da conexão com o sensor semântico.
func (c *Client) CheckAvailable(ctx context.Context) bool {
	if !c.IsConfigured() {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.Endpoint, nil)
	if err != nil {
		return false
	}
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest
}

// EvalNoul submete uma requisição de avaliação booleana ao sensor semântico.
func (c *Client) EvalNoul(ctx context.Context, req EvalRequest) (*NoulDecision, error) {
	if req.State == "" || req.Question == "" || req.Threshold < 0.0 || req.Threshold > 1.0 {
		return nil, ErrInvalidRequest
	}

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, ErrServerFailure
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, sanitizeErr(fmt.Errorf("unexpected status code: %d", resp.StatusCode), c.cfg.APIKey)
	}

	maxLimit := c.cfg.MaxBodyBytes
	if maxLimit <= 0 {
		maxLimit = 10 * 1024 * 1024
	}

	limitReader := io.LimitReader(resp.Body, maxLimit+1)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}
	if int64(len(body)) > maxLimit {
		return nil, ErrResponseTooLarge
	}

	decision, err := ParseNoulResponse(body)
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}

	if decision.LatencyMs == 0 {
		decision.LatencyMs = time.Since(start).Milliseconds()
	}

	if decision.Confidence < req.Threshold {
		return nil, ErrInsufficientConfidence
	}

	return decision, nil
}

// ParseNoulResponse desserializa e valida a resposta JSON do sensor semântico.
func ParseNoulResponse(data []byte) (*NoulDecision, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("empty response body")
	}

	var decision NoulDecision
	if err := json.Unmarshal(trimmed, &decision); err != nil {
		return nil, err
	}

	return &decision, nil
}
