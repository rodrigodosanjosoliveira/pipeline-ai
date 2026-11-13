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

// effectiveProvider determina o provedor semântico em uso (gemini ou jev).
func (c *Client) effectiveProvider() string {
	p := strings.ToLower(strings.TrimSpace(c.cfg.Provider))
	if p == ProviderGemini {
		return ProviderGemini
	}
	if p == "" && strings.Contains(c.cfg.Endpoint, "generativelanguage.googleapis.com") {
		return ProviderGemini
	}
	return ProviderJev
}

// effectiveEndpoint retorna a URL de destino da requisição semântica.
func (c *Client) effectiveEndpoint() string {
	if c.effectiveProvider() == ProviderGemini {
		if strings.TrimSpace(c.cfg.Endpoint) != "" {
			return strings.TrimSpace(c.cfg.Endpoint)
		}
		model := strings.TrimSpace(c.cfg.Model)
		if model == "" {
			model = DefaultGeminiModel
		}
		return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	}
	return c.cfg.Endpoint
}

// IsConfigured verifica se o cliente possui endpoint e credenciais configuradas.
func (c *Client) IsConfigured() bool {
	if c.effectiveProvider() == ProviderGemini {
		return strings.TrimSpace(c.cfg.APIKey) != ""
	}
	return strings.TrimSpace(c.cfg.Endpoint) != "" && strings.TrimSpace(c.cfg.APIKey) != ""
}

// CheckAvailable verifica a disponibilidade e saúde da conexão com o sensor semântico.
func (c *Client) CheckAvailable(ctx context.Context) bool {
	if !c.IsConfigured() {
		return false
	}
	endpoint := c.effectiveEndpoint()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}
	if c.effectiveProvider() == ProviderGemini {
		if c.cfg.APIKey != "" {
			req.Header.Set("x-goog-api-key", c.cfg.APIKey)
		}
	} else {
		if c.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest
}

// doPost realiza a chamada HTTP POST autenticada para o endpoint do sensor semântico,
// aplicando limites de tamanho (MaxBodyBytes) e sanitização de erros.
func (c *Client) doPost(ctx context.Context, req EvalRequest) ([]byte, int64, error) {
	var payloadBytes []byte
	var err error

	if c.effectiveProvider() == ProviderGemini {
		var prompt string
		var schema map[string]any

		switch req.Mode {
		case "choice":
			prompt = fmt.Sprintf("Question: %s\nChoices: %s\nState:\n%s\nSelect the best choice from the allowed choices for the provided state. Return a JSON object with selected (must be one of the choices), confidence (0.0 to 1.0), options (probabilities map), and reason.",
				req.Question, strings.Join(req.Choices, ", "), req.State)
			schema = map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"selected":   map[string]any{"type": "STRING"},
					"confidence": map[string]any{"type": "NUMBER"},
					"options": map[string]any{
						"type":                 "OBJECT",
						"additionalProperties": map[string]any{"type": "NUMBER"},
					},
					"reason": map[string]any{"type": "STRING"},
				},
				"required": []string{"selected", "confidence", "options", "reason"},
			}
		case "score":
			prompt = fmt.Sprintf("Question: %s\nMax Score: %f\nState:\n%s\nEvaluate and rate the state on a numerical scale up to max_score. Return a JSON object with score, max_score, and reason.",
				req.Question, req.MaxScore, req.State)
			schema = map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"score":     map[string]any{"type": "NUMBER"},
					"max_score": map[string]any{"type": "NUMBER"},
					"reason":    map[string]any{"type": "STRING"},
				},
				"required": []string{"score", "max_score", "reason"},
			}
		default: // "noul"
			prompt = fmt.Sprintf("Question: %s\nState:\n%s\nEvaluate the question against the provided state and return a JSON object with boolean decision, confidence score (0.0 to 1.0), and reason.",
				req.Question, req.State)
			schema = map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"decision":   map[string]any{"type": "BOOLEAN"},
					"confidence": map[string]any{"type": "NUMBER"},
					"reason":     map[string]any{"type": "STRING"},
				},
				"required": []string{"decision", "confidence", "reason"},
			}
		}

		thinkingLevel := strings.ToLower(strings.TrimSpace(c.cfg.ThinkingLevel))
		if thinkingLevel == "" {
			thinkingLevel = DefaultGeminiThinkingLevel
		}

		geminiPayload := map[string]any{
			"contents": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]any{
						{"text": prompt},
					},
				},
			},
			"generationConfig": map[string]any{
				"responseMimeType": "application/json",
				"responseSchema":   schema,
				"thinkingConfig": map[string]any{
					"thinkingLevel": thinkingLevel,
				},
			},
		}
		payloadBytes, err = json.Marshal(geminiPayload)
	} else {
		payloadBytes, err = json.Marshal(req)
	}

	if err != nil {
		return nil, 0, sanitizeErr(err, c.cfg.APIKey)
	}

	endpoint := c.effectiveEndpoint()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, 0, sanitizeErr(err, c.cfg.APIKey)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.effectiveProvider() == ProviderGemini {
		if c.cfg.APIKey != "" {
			httpReq.Header.Set("x-goog-api-key", c.cfg.APIKey)
		}
	} else {
		if c.cfg.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		}
	}

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, sanitizeErr(err, c.cfg.APIKey)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, ErrUnauthorized
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, 0, ErrServerFailure
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, 0, sanitizeErr(fmt.Errorf("unexpected status code: %d", resp.StatusCode), c.cfg.APIKey)
	}

	maxLimit := c.cfg.MaxBodyBytes
	if maxLimit <= 0 {
		maxLimit = 10 * 1024 * 1024
	}

	limitReader := io.LimitReader(resp.Body, maxLimit+1)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, 0, sanitizeErr(err, c.cfg.APIKey)
	}
	if int64(len(body)) > maxLimit {
		return nil, 0, ErrResponseTooLarge
	}

	latencyMs := time.Since(start).Milliseconds()
	return body, latencyMs, nil
}

// EvalNoul submete uma requisição de avaliação booleana ao sensor semântico.
func (c *Client) EvalNoul(ctx context.Context, req EvalRequest) (*NoulDecision, error) {
	if req.State == "" || req.Question == "" || req.Threshold < 0.0 || req.Threshold > 1.0 {
		return nil, ErrInvalidRequest
	}
	if req.Mode == "" {
		req.Mode = "noul"
	}

	body, latencyMs, err := c.doPost(ctx, req)
	if err != nil {
		return nil, err
	}

	if c.effectiveProvider() == ProviderGemini {
		parsed, err := ParseGeminiResponse(body)
		if err != nil {
			return nil, sanitizeErr(err, c.cfg.APIKey)
		}
		body = parsed
	}

	decision, err := ParseNoulResponse(body)
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}

	if decision.LatencyMs == 0 {
		decision.LatencyMs = latencyMs
	}

	if decision.Confidence < req.Threshold {
		return nil, ErrInsufficientConfidence
	}

	return decision, nil
}

// EvalChoice submete uma requisição de avaliação de múltipla escolha ao sensor semântico.
func (c *Client) EvalChoice(ctx context.Context, req EvalRequest) (*ChoiceDecision, error) {
	if req.State == "" || req.Question == "" || req.Threshold < 0.0 || req.Threshold > 1.0 || len(req.Choices) < 2 {
		return nil, ErrInvalidRequest
	}
	if req.Mode == "" {
		req.Mode = "choice"
	}

	body, latencyMs, err := c.doPost(ctx, req)
	if err != nil {
		return nil, err
	}

	if c.effectiveProvider() == ProviderGemini {
		parsed, err := ParseGeminiResponse(body)
		if err != nil {
			return nil, sanitizeErr(err, c.cfg.APIKey)
		}
		body = parsed
	}

	decision, err := ParseChoiceResponse(body)
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}

	if decision.LatencyMs == 0 {
		decision.LatencyMs = latencyMs
	}

	selected := strings.TrimSpace(decision.Selected)
	if selected == "" {
		return nil, ErrInvalidChoice
	}

	allowed := false
	for _, ch := range req.Choices {
		trimmedChoice := strings.TrimSpace(ch)
		if trimmedChoice != "" && (selected == trimmedChoice || strings.EqualFold(selected, trimmedChoice)) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrInvalidChoice
	}

	if decision.Confidence < req.Threshold {
		return nil, ErrInsufficientConfidence
	}

	return decision, nil
}

// EvalScore submete uma requisição de avaliação numérica contínua ao sensor semântico.
func (c *Client) EvalScore(ctx context.Context, req EvalRequest) (*ScoreDecision, error) {
	if req.State == "" || req.Question == "" || req.Threshold < 0.0 || req.Threshold > 1.0 {
		return nil, ErrInvalidRequest
	}
	if req.Mode == "" {
		req.Mode = "score"
	}

	body, latencyMs, err := c.doPost(ctx, req)
	if err != nil {
		return nil, err
	}

	if c.effectiveProvider() == ProviderGemini {
		parsed, err := ParseGeminiResponse(body)
		if err != nil {
			return nil, sanitizeErr(err, c.cfg.APIKey)
		}
		body = parsed
	}

	decision, err := ParseScoreResponse(body)
	if err != nil {
		return nil, sanitizeErr(err, c.cfg.APIKey)
	}

	if decision.LatencyMs == 0 {
		decision.LatencyMs = latencyMs
	}

	if decision.MaxScore == 0 && req.MaxScore > 0 {
		decision.MaxScore = req.MaxScore
	}

	if decision.MaxScore <= 0 || decision.Score < 0 || decision.Score > decision.MaxScore {
		return nil, ErrInvalidScore
	}

	if (decision.Score / decision.MaxScore) < req.Threshold {
		return nil, ErrInsufficientConfidence
	}

	return decision, nil
}

type geminiCandidatePart struct {
	Text string `json:"text"`
}

type geminiCandidateContent struct {
	Parts []geminiCandidatePart `json:"parts"`
	Role  string                `json:"role,omitempty"`
}

type geminiCandidate struct {
	Content geminiCandidateContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

// ParseGeminiResponse desserializa o envelope oficial do Gemini e extrai o texto JSON do primeiro candidato.
func ParseGeminiResponse(data []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("empty response body")
	}

	var resp geminiResponse
	if err := json.Unmarshal(trimmed, &resp); err != nil {
		return nil, fmt.Errorf("gemini response json invalid: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, errors.New("gemini response has no candidates")
	}

	parts := resp.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return nil, errors.New("gemini candidate has no content parts")
	}

	text := strings.TrimSpace(parts[0].Text)
	if text == "" {
		return nil, errors.New("gemini response text part is empty")
	}

	return []byte(text), nil
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

// ParseChoiceResponse desserializa e valida a resposta JSON de múltipla escolha do sensor semântico.
func ParseChoiceResponse(data []byte) (*ChoiceDecision, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("empty response body")
	}

	var decision ChoiceDecision
	if err := json.Unmarshal(trimmed, &decision); err != nil {
		return nil, err
	}

	return &decision, nil
}

// ParseScoreResponse desserializa e valida a resposta JSON de score contínuo do sensor semântico.
func ParseScoreResponse(data []byte) (*ScoreDecision, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("empty response body")
	}

	var decision ScoreDecision
	if err := json.Unmarshal(trimmed, &decision); err != nil {
		return nil, err
	}

	return &decision, nil
}
