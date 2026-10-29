package semantic

import (
	"errors"
	"time"
)

var (
	// ErrInvalidRequest indica que os parâmetros da requisição são inválidos.
	ErrInvalidRequest = errors.New("invalid request: state and question must not be empty, threshold must be between 0.0 and 1.0")

	// ErrInsufficientConfidence indica que o score de confiança retornou abaixo do threshold requerido.
	ErrInsufficientConfidence = errors.New("insufficient confidence score")

	// ErrResponseTooLarge indica que o corpo da resposta excedeu MaxBodyBytes.
	ErrResponseTooLarge = errors.New("response body exceeds max allowed bytes")

	// ErrUnauthorized indica falha de autenticação com a API do sensor.
	ErrUnauthorized = errors.New("unauthorized: invalid or missing api key")

	// ErrServerFailure indica falha interna (5xx) no servidor remoto do sensor.
	ErrServerFailure = errors.New("server failure")
)

// NoulDecision representa uma decisão booleana com confiança e justificativa do sensor semântico.
type NoulDecision struct {
	Decision   bool    `json:"decision"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	LatencyMs  int64   `json:"latency_ms,omitempty"`
}

// ChoiceDecision representa uma decisão de múltipla escolha com distribuição de probabilidades.
type ChoiceDecision struct {
	Selected   string             `json:"selected"`
	Confidence float64            `json:"confidence"`
	Options    map[string]float64 `json:"options"`
}

// ScoreDecision representa uma avaliação com pontuação numérica contínua.
type ScoreDecision struct {
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Reason   string  `json:"reason"`
}

// EvalRequest define os parâmetros para avaliação de um estado semântico.
type EvalRequest struct {
	State     string  `json:"state"`
	Question  string  `json:"question"`
	Threshold float64 `json:"threshold"`
}

// ClientConfig armazena as configurações de conexão e restrições de segurança do cliente semântico.
type ClientConfig struct {
	Endpoint     string
	APIKey       string
	Timeout      time.Duration
	MaxBodyBytes int64
}
