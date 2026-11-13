package semantic_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic"
)

func TestEvalNoul_HappyPath(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		req            semantic.EvalRequest
		wantDecision   bool
		wantConfidence float64
		wantReason     string
	}{
		{
			name:           "positive decision with high confidence",
			serverResponse: `{"decision": true, "confidence": 0.95, "reason": "All checks passed"}`,
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "Is this safe?",
				Threshold: 0.8,
			},
			wantDecision:   true,
			wantConfidence: 0.95,
			wantReason:     "All checks passed",
		},
		{
			name:           "negative decision with high confidence",
			serverResponse: `{"decision": false, "confidence": 0.89, "reason": "Safety violation detected"}`,
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "Does it violate rules?",
				Threshold: 0.7,
			},
			wantDecision:   false,
			wantConfidence: 0.89,
			wantReason:     "Safety violation detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedHeader http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedHeader = r.Header.Clone()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			apiKey := "test-secret-key-abc"
			client := semantic.NewClient(semantic.ClientConfig{
				Endpoint:     server.URL,
				APIKey:       apiKey,
				Timeout:      2 * time.Second,
				MaxBodyBytes: 1048576,
			})

			ctx := context.Background()
			decision, err := client.EvalNoul(ctx, tt.req)
			if err != nil {
				t.Fatalf("EvalNoul returned unexpected error: %v", err)
			}
			if decision == nil {
				t.Fatal("expected non-nil decision, got nil")
			}
			if decision.Decision != tt.wantDecision {
				t.Errorf("expected Decision=%v, got %v", tt.wantDecision, decision.Decision)
			}
			if decision.Confidence != tt.wantConfidence {
				t.Errorf("expected Confidence=%v, got %v", tt.wantConfidence, decision.Confidence)
			}
			if decision.Reason != tt.wantReason {
				t.Errorf("expected Reason=%q, got %q", tt.wantReason, decision.Reason)
			}

			// Validar envio de Authorization Header
			authHeader := capturedHeader.Get("Authorization")
			if !strings.Contains(authHeader, apiKey) {
				t.Errorf("expected Authorization header to contain API key, got: %q", authHeader)
			}
		})
	}
}

func TestEvalNoul_ConfidenceThreshold(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		req            semantic.EvalRequest
		wantErr        error
	}{
		{
			name:           "confidence equal to threshold succeeds",
			serverResponse: `{"decision": true, "confidence": 0.80, "reason": "Exact match"}`,
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "Valid?",
				Threshold: 0.80,
			},
			wantErr: nil,
		},
		{
			name:           "confidence above threshold succeeds",
			serverResponse: `{"decision": true, "confidence": 0.90, "reason": "High confidence"}`,
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "Valid?",
				Threshold: 0.80,
			},
			wantErr: nil,
		},
		{
			name:           "confidence below threshold fails with ErrInsufficientConfidence",
			serverResponse: `{"decision": true, "confidence": 0.65, "reason": "Low confidence"}`,
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "Valid?",
				Threshold: 0.80,
			},
			wantErr: semantic.ErrInsufficientConfidence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := semantic.NewClient(semantic.ClientConfig{
				Endpoint:     server.URL,
				APIKey:       "key-123",
				Timeout:      2 * time.Second,
				MaxBodyBytes: 1048576,
			})

			_, err := client.EvalNoul(context.Background(), tt.req)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestEvalNoul_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     semantic.EvalRequest
		wantErr bool
	}{
		{
			name: "empty state returns error",
			req: semantic.EvalRequest{
				State:     "",
				Question:  "Valid?",
				Threshold: 0.8,
			},
			wantErr: true,
		},
		{
			name: "empty question returns error",
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "",
				Threshold: 0.8,
			},
			wantErr: true,
		},
		{
			name: "both state and question empty returns error",
			req: semantic.EvalRequest{
				State:     "",
				Question:  "",
				Threshold: 0.8,
			},
			wantErr: true,
		},
		{
			name: "negative threshold returns error",
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "Valid?",
				Threshold: -0.1,
			},
			wantErr: true,
		},
		{
			name: "threshold above 1.0 returns error",
			req: semantic.EvalRequest{
				State:     "active",
				Question:  "Valid?",
				Threshold: 1.05,
			},
			wantErr: true,
		},
	}

	client := semantic.NewClient(semantic.ClientConfig{
		Endpoint:     "http://localhost:9999",
		APIKey:       "key-123",
		Timeout:      2 * time.Second,
		MaxBodyBytes: 1048576,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.EvalNoul(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvalNoul() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !errors.Is(err, semantic.ErrInvalidRequest) && !strings.Contains(err.Error(), "invalid request") {
				t.Errorf("expected ErrInvalidRequest, got: %v", err)
			}
		})
	}
}

func TestEvalNoul_Security_TokenRedaction(t *testing.T) {
	// SEC-01: Erros de requisição NÃO devem vazar o token de autenticação
	secretKey := "super-secret-jev-api-token-987654321"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simula erro de servidor
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal server crash`))
	}))
	defer server.Close()

	client := semantic.NewClient(semantic.ClientConfig{
		Endpoint:     server.URL,
		APIKey:       secretKey,
		Timeout:      2 * time.Second,
		MaxBodyBytes: 1048576,
	})

	req := semantic.EvalRequest{
		State:     "active",
		Question:  "Is this safe?",
		Threshold: 0.8,
	}

	_, err := client.EvalNoul(context.Background(), req)
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}

	if strings.Contains(err.Error(), secretKey) {
		t.Fatalf("SEC-01 VIOLATION: error message leaked secret APIKey: %s", err.Error())
	}
}

func TestEvalNoul_Security_MaxBodyBytes(t *testing.T) {
	// SEC-02: Resposta maliciosa com payload gigante deve ser interrompida por MaxBodyBytes
	maxAllowed := int64(512)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Envia payload de 100KB, excedendo o limite de 512 bytes
		junk := strings.Repeat("A", 100*1024)
		_, _ = io.WriteString(w, `{"decision": true, "reason": "`+junk+`"}`)
	}))
	defer server.Close()

	client := semantic.NewClient(semantic.ClientConfig{
		Endpoint:     server.URL,
		APIKey:       "key-test",
		Timeout:      2 * time.Second,
		MaxBodyBytes: maxAllowed,
	})

	req := semantic.EvalRequest{
		State:     "active",
		Question:  "Is this safe?",
		Threshold: 0.8,
	}

	_, err := client.EvalNoul(context.Background(), req)
	if err == nil {
		t.Fatal("expected error due to response exceeding MaxBodyBytes, got nil")
	}

	if !errors.Is(err, semantic.ErrResponseTooLarge) && !strings.Contains(strings.ToLower(err.Error()), "body") && !strings.Contains(strings.ToLower(err.Error()), "exceed") {
		t.Errorf("expected ErrResponseTooLarge or payload size limit error, got: %v", err)
	}
}

func TestEvalNoul_Security_ContextTimeout(t *testing.T) {
	// SEC-03: Timeout e cancelamento de context
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"decision": true, "confidence": 0.9, "reason": "ok"}`))
	}))
	defer server.Close()

	client := semantic.NewClient(semantic.ClientConfig{
		Endpoint:     server.URL,
		APIKey:       "key-test",
		Timeout:      2 * time.Second,
		MaxBodyBytes: 1048576,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	req := semantic.EvalRequest{
		State:     "active",
		Question:  "Is this safe?",
		Threshold: 0.8,
	}

	_, err := client.EvalNoul(ctx, req)
	if err == nil {
		t.Fatal("expected context timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestEvalNoul_HTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{
			name:       "401 Unauthorized returns ErrUnauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error": "unauthorized"}`,
			wantErr:    semantic.ErrUnauthorized,
		},
		{
			name:       "500 Internal Server Error returns ErrServerFailure",
			statusCode: http.StatusInternalServerError,
			body:       `{"error": "internal error"}`,
			wantErr:    semantic.ErrServerFailure,
		},
		{
			name:       "200 OK with malformed json returns error",
			statusCode: http.StatusOK,
			body:       `{"decision": true, confidence: invalid-json`,
			wantErr:    nil, // Deve retornar erro de parse
		},
		{
			name:       "200 OK with empty body returns error",
			statusCode: http.StatusOK,
			body:       ``,
			wantErr:    nil, // Deve retornar erro de parse
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := semantic.NewClient(semantic.ClientConfig{
				Endpoint:     server.URL,
				APIKey:       "key-test",
				Timeout:      2 * time.Second,
				MaxBodyBytes: 1048576,
			})

			req := semantic.EvalRequest{
				State:     "active",
				Question:  "Is this safe?",
				Threshold: 0.8,
			}

			_, err := client.EvalNoul(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error for status %d, got nil", tt.statusCode)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestClient_Availability(t *testing.T) {
	t.Run("IsConfigured", func(t *testing.T) {
		tests := []struct {
			name     string
			cfg      semantic.ClientConfig
			wantConf bool
		}{
			{
				name:     "empty config returns false",
				cfg:      semantic.ClientConfig{},
				wantConf: false,
			},
			{
				name: "endpoint only returns false",
				cfg: semantic.ClientConfig{
					Endpoint: "https://api.jev.ai",
				},
				wantConf: false,
			},
			{
				name: "api key only returns false",
				cfg: semantic.ClientConfig{
					APIKey: "secret-key",
				},
				wantConf: false,
			},
			{
				name: "endpoint and api key returns true",
				cfg: semantic.ClientConfig{
					Endpoint: "https://api.jev.ai",
					APIKey:   "secret-key",
				},
				wantConf: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				c := semantic.NewClient(tt.cfg)
				if got := c.IsConfigured(); got != tt.wantConf {
					t.Errorf("IsConfigured() = %v, want %v", got, tt.wantConf)
				}
			})
		}
	})

	t.Run("CheckAvailable", func(t *testing.T) {
		// Servidor saudável
		serverOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer serverOK.Close()

		clientOK := semantic.NewClient(semantic.ClientConfig{
			Endpoint: serverOK.URL,
			APIKey:   "key",
			Timeout:  1 * time.Second,
		})
		if !clientOK.CheckAvailable(context.Background()) {
			t.Errorf("CheckAvailable() = false for healthy server, want true")
		}

		// Servidor indisponível / 503
		serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer serverFail.Close()

		clientFail := semantic.NewClient(semantic.ClientConfig{
			Endpoint: serverFail.URL,
			APIKey:   "key",
			Timeout:  1 * time.Second,
		})
		if clientFail.CheckAvailable(context.Background()) {
			t.Errorf("CheckAvailable() = true for 503 server, want false")
		}
	})
}

func TestParseNoulResponse(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		wantDecision   bool
		wantConfidence float64
		wantReason     string
		wantErr        bool
	}{
		{
			name:           "valid positive payload",
			input:          []byte(`{"decision": true, "confidence": 0.95, "reason": "Valid pattern"}`),
			wantDecision:   true,
			wantConfidence: 0.95,
			wantReason:     "Valid pattern",
			wantErr:        false,
		},
		{
			name:           "valid negative payload",
			input:          []byte(`{"decision": false, "confidence": 0.12, "reason": "Anomaly detected"}`),
			wantDecision:   false,
			wantConfidence: 0.12,
			wantReason:     "Anomaly detected",
			wantErr:        false,
		},
		{
			name:    "empty input returns error",
			input:   []byte(``),
			wantErr: true,
		},
		{
			name:    "malformed json returns error",
			input:   []byte(`{"decision": `),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := semantic.ParseNoulResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseNoulResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if dec == nil {
					t.Fatal("expected non-nil decision, got nil")
				}
				if dec.Decision != tt.wantDecision {
					t.Errorf("expected Decision=%v, got %v", tt.wantDecision, dec.Decision)
				}
				if dec.Confidence != tt.wantConfidence {
					t.Errorf("expected Confidence=%v, got %v", tt.wantConfidence, dec.Confidence)
				}
				if dec.Reason != tt.wantReason {
					t.Errorf("expected Reason=%q, got %q", tt.wantReason, dec.Reason)
				}
			}
		})
	}
}

func TestParseChoiceResponse(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		wantSelected   string
		wantConfidence float64
		wantReason     string
		wantErr        bool
	}{
		{
			name:           "valid choice payload with options and reason",
			input:          []byte(`{"selected": "bugfix", "confidence": 0.91, "options": {"feature": 0.05, "bugfix": 0.91, "refactor": 0.04}, "reason": "Fixes panic on nil pointer"}`),
			wantSelected:   "bugfix",
			wantConfidence: 0.91,
			wantReason:     "Fixes panic on nil pointer",
			wantErr:        false,
		},
		{
			name:    "empty input returns error",
			input:   []byte(``),
			wantErr: true,
		},
		{
			name:    "malformed json returns error",
			input:   []byte(`{"selected": "bugfix", "confidence": `),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := semantic.ParseChoiceResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseChoiceResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if dec == nil {
					t.Fatal("expected non-nil ChoiceDecision, got nil")
				}
				if dec.Selected != tt.wantSelected {
					t.Errorf("expected Selected=%q, got %q", tt.wantSelected, dec.Selected)
				}
				if dec.Confidence != tt.wantConfidence {
					t.Errorf("expected Confidence=%v, got %v", tt.wantConfidence, dec.Confidence)
				}
				if dec.Reason != tt.wantReason {
					t.Errorf("expected Reason=%q, got %q", tt.wantReason, dec.Reason)
				}
			}
		})
	}
}

func TestParseScoreResponse(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		wantScore    float64
		wantMaxScore float64
		wantReason   string
		wantErr      bool
	}{
		{
			name:         "valid score payload",
			input:        []byte(`{"score": 8.5, "max_score": 10.0, "reason": "Clean architecture and full test coverage"}`),
			wantScore:    8.5,
			wantMaxScore: 10.0,
			wantReason:   "Clean architecture and full test coverage",
			wantErr:      false,
		},
		{
			name:    "empty input returns error",
			input:   []byte(`   `),
			wantErr: true,
		},
		{
			name:    "malformed json returns error",
			input:   []byte(`{"score": "not-a-number"}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := semantic.ParseScoreResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseScoreResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if dec == nil {
					t.Fatal("expected non-nil ScoreDecision, got nil")
				}
				if dec.Score != tt.wantScore {
					t.Errorf("expected Score=%v, got %v", tt.wantScore, dec.Score)
				}
				if dec.MaxScore != tt.wantMaxScore {
					t.Errorf("expected MaxScore=%v, got %v", tt.wantMaxScore, dec.MaxScore)
				}
				if dec.Reason != tt.wantReason {
					t.Errorf("expected Reason=%q, got %q", tt.wantReason, dec.Reason)
				}
			}
		})
	}
}

func TestEvalChoice(t *testing.T) {
	tests := []struct {
		name           string
		serverStatus   int
		serverResponse string
		req            semantic.EvalRequest
		maxBodyBytes   int64
		wantSelected   string
		wantConfidence float64
		wantErr        error
	}{
		{
			name:           "happy path selects valid option above threshold",
			serverStatus:   http.StatusOK,
			serverResponse: `{"selected": "refactor", "confidence": 0.88, "options": {"feature": 0.05, "bugfix": 0.07, "refactor": 0.88}, "reason": "Code restructuring only"}`,
			req: semantic.EvalRequest{
				State:     "diff content",
				Question:  "Classify change type",
				Threshold: 0.80,
				Mode:      "choice",
				Choices:   []string{"feature", "bugfix", "refactor"},
			},
			wantSelected:   "refactor",
			wantConfidence: 0.88,
			wantErr:        nil,
		},
		{
			name:           "SEC-01 fewer than 2 choices returns ErrInvalidRequest",
			serverStatus:   http.StatusOK,
			serverResponse: `{"selected": "feature", "confidence": 0.99}`,
			req: semantic.EvalRequest{
				State:     "diff content",
				Question:  "Classify change type",
				Threshold: 0.80,
				Choices:   []string{"feature"},
			},
			wantErr: semantic.ErrInvalidRequest,
		},
		{
			name:           "SEC-01 empty choices returns ErrInvalidRequest",
			serverStatus:   http.StatusOK,
			serverResponse: `{"selected": "feature", "confidence": 0.99}`,
			req: semantic.EvalRequest{
				State:     "diff content",
				Question:  "Classify change type",
				Threshold: 0.80,
				Choices:   nil,
			},
			wantErr: semantic.ErrInvalidRequest,
		},
		{
			name:           "SEC-01 hallucinated option not in Choices returns ErrInvalidChoice",
			serverStatus:   http.StatusOK,
			serverResponse: `{"selected": "hotfix", "confidence": 0.95, "reason": "Invented category"}`,
			req: semantic.EvalRequest{
				State:     "diff content",
				Question:  "Classify change type",
				Threshold: 0.80,
				Choices:   []string{"feature", "bugfix", "refactor"},
			},
			wantErr: semantic.ErrInvalidChoice,
		},
		{
			name:           "SEC-01 empty selected option returns ErrInvalidChoice",
			serverStatus:   http.StatusOK,
			serverResponse: `{"selected": "", "confidence": 0.95, "reason": "No option selected"}`,
			req: semantic.EvalRequest{
				State:     "diff content",
				Question:  "Classify change type",
				Threshold: 0.80,
				Choices:   []string{"feature", "bugfix"},
			},
			wantErr: semantic.ErrInvalidChoice,
		},
		{
			name:           "SEC-01 confidence below threshold returns ErrInsufficientConfidence",
			serverStatus:   http.StatusOK,
			serverResponse: `{"selected": "feature", "confidence": 0.70, "reason": "Uncertain"}`,
			req: semantic.EvalRequest{
				State:     "diff content",
				Question:  "Classify change type",
				Threshold: 0.85,
				Choices:   []string{"feature", "bugfix"},
			},
			wantErr: semantic.ErrInsufficientConfidence,
		},
		{
			name:           "SEC-03 response exceeding MaxBodyBytes returns ErrResponseTooLarge",
			serverStatus:   http.StatusOK,
			serverResponse: `{"selected": "feature", "confidence": 0.95, "reason": "` + strings.Repeat("X", 1024) + `"}`,
			req: semantic.EvalRequest{
				State:     "diff content",
				Question:  "Classify change type",
				Threshold: 0.80,
				Choices:   []string{"feature", "bugfix"},
			},
			maxBodyBytes: 128,
			wantErr:      semantic.ErrResponseTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			maxBytes := tt.maxBodyBytes
			if maxBytes == 0 {
				maxBytes = 1048576
			}
			secretKey := "secret-choice-api-key-999"
			client := semantic.NewClient(semantic.ClientConfig{
				Endpoint:     server.URL,
				APIKey:       secretKey,
				Timeout:      2 * time.Second,
				MaxBodyBytes: maxBytes,
			})

			dec, err := client.EvalChoice(context.Background(), tt.req)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("EvalChoice unexpected error: %v", err)
			}
			if dec.Selected != tt.wantSelected {
				t.Errorf("Selected = %q, want %q", dec.Selected, tt.wantSelected)
			}
			if dec.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", dec.Confidence, tt.wantConfidence)
			}
		})
	}

	t.Run("SEC-03 token redaction on error in EvalChoice", func(t *testing.T) {
		secretKey := "choice-secret-token-777"
		client := semantic.NewClient(semantic.ClientConfig{
			Endpoint: "http://127.0.0.1:0/" + secretKey,
			APIKey:   secretKey,
			Timeout:  100 * time.Millisecond,
		})
		_, err := client.EvalChoice(context.Background(), semantic.EvalRequest{
			State:     "state",
			Question:  "question?",
			Threshold: 0.8,
			Choices:   []string{"a", "b"},
		})
		if err == nil {
			t.Fatal("expected network error, got nil")
		}
		if strings.Contains(err.Error(), secretKey) {
			t.Fatalf("SEC-03 violation: EvalChoice error leaked secret APIKey: %v", err)
		}
	})
}

func TestEvalScore(t *testing.T) {
	tests := []struct {
		name           string
		serverStatus   int
		serverResponse string
		req            semantic.EvalRequest
		maxBodyBytes   int64
		wantScore      float64
		wantMaxScore   float64
		wantErr        error
	}{
		{
			name:           "happy path normalized ratio meets threshold",
			serverStatus:   http.StatusOK,
			serverResponse: `{"score": 8.5, "max_score": 10.0, "reason": "Well tested"}`,
			req: semantic.EvalRequest{
				State:     "code",
				Question:  "Rate quality",
				Threshold: 0.80,
				Mode:      "score",
				MaxScore:  10.0,
			},
			wantScore:    8.5,
			wantMaxScore: 10.0,
			wantErr:      nil,
		},
		{
			name:           "SEC-02 negative score returns ErrInvalidScore",
			serverStatus:   http.StatusOK,
			serverResponse: `{"score": -1.0, "max_score": 10.0, "reason": "Negative"}`,
			req: semantic.EvalRequest{
				State:     "code",
				Question:  "Rate quality",
				Threshold: 0.50,
				MaxScore:  10.0,
			},
			wantErr: semantic.ErrInvalidScore,
		},
		{
			name:           "SEC-02 non-positive max_score returns ErrInvalidScore",
			serverStatus:   http.StatusOK,
			serverResponse: `{"score": 5.0, "max_score": 0.0, "reason": "Zero max"}`,
			req: semantic.EvalRequest{
				State:     "code",
				Question:  "Rate quality",
				Threshold: 0.50,
				MaxScore:  0.0,
			},
			wantErr: semantic.ErrInvalidScore,
		},
		{
			name:           "SEC-02 score greater than max_score returns ErrInvalidScore",
			serverStatus:   http.StatusOK,
			serverResponse: `{"score": 12.0, "max_score": 10.0, "reason": "Overflow"}`,
			req: semantic.EvalRequest{
				State:     "code",
				Question:  "Rate quality",
				Threshold: 0.50,
				MaxScore:  10.0,
			},
			wantErr: semantic.ErrInvalidScore,
		},
		{
			name:           "SEC-02 normalized score below threshold returns ErrInsufficientConfidence",
			serverStatus:   http.StatusOK,
			serverResponse: `{"score": 7.0, "max_score": 10.0, "reason": "Below 0.8 threshold"}`,
			req: semantic.EvalRequest{
				State:     "code",
				Question:  "Rate quality",
				Threshold: 0.80,
				MaxScore:  10.0,
			},
			wantErr: semantic.ErrInsufficientConfidence,
		},
		{
			name:           "SEC-03 response exceeding MaxBodyBytes returns ErrResponseTooLarge",
			serverStatus:   http.StatusOK,
			serverResponse: `{"score": 9.0, "max_score": 10.0, "reason": "` + strings.Repeat("Y", 1024) + `"}`,
			req: semantic.EvalRequest{
				State:     "code",
				Question:  "Rate quality",
				Threshold: 0.80,
				MaxScore:  10.0,
			},
			maxBodyBytes: 128,
			wantErr:      semantic.ErrResponseTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			maxBytes := tt.maxBodyBytes
			if maxBytes == 0 {
				maxBytes = 1048576
			}
			client := semantic.NewClient(semantic.ClientConfig{
				Endpoint:     server.URL,
				APIKey:       "score-key-123",
				Timeout:      2 * time.Second,
				MaxBodyBytes: maxBytes,
			})

			dec, err := client.EvalScore(context.Background(), tt.req)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("EvalScore unexpected error: %v", err)
			}
			if dec.Score != tt.wantScore {
				t.Errorf("Score = %v, want %v", dec.Score, tt.wantScore)
			}
			if dec.MaxScore != tt.wantMaxScore {
				t.Errorf("MaxScore = %v, want %v", dec.MaxScore, tt.wantMaxScore)
			}
		})
	}

	t.Run("SEC-03 token redaction on error in EvalScore", func(t *testing.T) {
		secretKey := "score-secret-token-888"
		client := semantic.NewClient(semantic.ClientConfig{
			Endpoint: "http://127.0.0.1:0/" + secretKey,
			APIKey:   secretKey,
			Timeout:  100 * time.Millisecond,
		})
		_, err := client.EvalScore(context.Background(), semantic.EvalRequest{
			State:     "state",
			Question:  "question?",
			Threshold: 0.8,
			MaxScore:  10.0,
		})
		if err == nil {
			t.Fatal("expected network error, got nil")
		}
		if strings.Contains(err.Error(), secretKey) {
			t.Fatalf("SEC-03 violation: EvalScore error leaked secret APIKey: %v", err)
		}
	})
}

func TestGeminiProvider_EvalModesAndSecurity(t *testing.T) {
	apiKey := "gem-secret-123"

	t.Run("Headers e payload estruturado para REST Gemini generateContent", func(t *testing.T) {
		var capturedReq *http.Request
		var capturedBody []byte

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedReq = r
			bodyBytes, _ := io.ReadAll(r.Body)
			capturedBody = bodyBytes
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"decision\":true,\"confidence\":0.92,\"reason\":\"ok\"}"}]}}]}`))
		}))
		defer srv.Close()

		client := semantic.NewClient(semantic.ClientConfig{
			Provider:      semantic.ProviderGemini,
			Endpoint:      srv.URL,
			APIKey:        apiKey,
			Model:         semantic.DefaultGeminiModel,
			ThinkingLevel: semantic.DefaultGeminiThinkingLevel,
			Timeout:       2 * time.Second,
		})

		ctx := context.Background()
		_, err := client.EvalNoul(ctx, semantic.EvalRequest{
			State:     "func test() {}",
			Question:  "isValid?",
			Threshold: 0.8,
		})
		if err != nil {
			t.Fatalf("EvalNoul retornou erro inesperado: %v", err)
		}

		if capturedReq.Method != http.MethodPost {
			t.Errorf("Método HTTP = %q, esperado %q", capturedReq.Method, http.MethodPost)
		}
		if capturedReq.Header.Get("x-goog-api-key") != apiKey {
			t.Errorf("Header x-goog-api-key = %q, esperado %q", capturedReq.Header.Get("x-goog-api-key"), apiKey)
		}
		if authHeader := capturedReq.Header.Get("Authorization"); authHeader != "" {
			t.Errorf("Header Authorization não deve estar presente no provider Gemini, obteve %q", authHeader)
		}
		if strings.Contains(capturedReq.URL.RawQuery, "key=") {
			t.Errorf("Query params na URL não devem conter 'key=', obteve %q", capturedReq.URL.RawQuery)
		}

		var payload map[string]any
		if err := json.Unmarshal(capturedBody, &payload); err != nil {
			t.Fatalf("Body enviado não é JSON válido: %v", err)
		}

		if _, ok := payload["contents"]; !ok {
			t.Errorf("Payload enviado não contém campo 'contents': %s", string(capturedBody))
		}

		genCfg, ok := payload["generationConfig"].(map[string]any)
		if !ok {
			t.Fatalf("Payload enviado não contém 'generationConfig': %s", string(capturedBody))
		}

		if genCfg["responseMimeType"] != "application/json" {
			t.Errorf("generationConfig.responseMimeType = %v, esperado 'application/json'", genCfg["responseMimeType"])
		}
		if _, ok := genCfg["responseSchema"]; !ok {
			t.Errorf("generationConfig.responseSchema ausente no payload: %v", genCfg)
		}

		thinkingCfg, ok := genCfg["thinkingConfig"].(map[string]any)
		if !ok {
			t.Fatalf("generationConfig.thinkingConfig ausente no payload: %v", genCfg)
		}
		if thinkingCfg["thinkingLevel"] != "low" {
			t.Errorf("thinkingConfig.thinkingLevel = %v, esperado 'low'", thinkingCfg["thinkingLevel"])
		}
	})

	t.Run("EvalNoul com decodificação de campos", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"decision\":true,\"confidence\":0.92,\"reason\":\"ok\"}"}]}}]}`))
		}))
		defer srv.Close()

		client := semantic.NewClient(semantic.ClientConfig{
			Provider:      semantic.ProviderGemini,
			Endpoint:      srv.URL,
			APIKey:        apiKey,
			Model:         semantic.DefaultGeminiModel,
			ThinkingLevel: semantic.DefaultGeminiThinkingLevel,
			Timeout:       2 * time.Second,
		})

		dec, err := client.EvalNoul(context.Background(), semantic.EvalRequest{
			State:     "active",
			Question:  "ok?",
			Threshold: 0.8,
		})
		if err != nil {
			t.Fatalf("EvalNoul retornou erro: %v", err)
		}
		if !dec.Decision {
			t.Errorf("Decision = %v, quer true", dec.Decision)
		}
		if dec.Confidence != 0.92 {
			t.Errorf("Confidence = %v, quer 0.92", dec.Confidence)
		}
		if dec.Reason != "ok" {
			t.Errorf("Reason = %q, quer 'ok'", dec.Reason)
		}
	})

	t.Run("EvalChoice com sucesso e rejeição SEC-01 de opção alucinada", func(t *testing.T) {
		var respText string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":%q}]}}]}`, respText)))
		}))
		defer srv.Close()

		client := semantic.NewClient(semantic.ClientConfig{
			Provider:      semantic.ProviderGemini,
			Endpoint:      srv.URL,
			APIKey:        apiKey,
			Model:         semantic.DefaultGeminiModel,
			ThinkingLevel: semantic.DefaultGeminiThinkingLevel,
			Timeout:       2 * time.Second,
		})

		// 1. Sucesso com opção válida
		respText = `{"selected":"feat","confidence":0.95,"options":{"feat":0.95,"fix":0.05},"reason":"feature detectada"}`
		dec, err := client.EvalChoice(context.Background(), semantic.EvalRequest{
			State:     "add new function",
			Question:  "type?",
			Threshold: 0.8,
			Choices:   []string{"feat", "fix"},
		})
		if err != nil {
			t.Fatalf("EvalChoice retornou erro inesperado: %v", err)
		}
		if dec.Selected != "feat" {
			t.Errorf("Selected = %q, quer 'feat'", dec.Selected)
		}

		// 2. Opção alucinada fora de Choices rejeitada com ErrInvalidChoice (SEC-01)
		respText = `{"selected":"refactor","confidence":0.98,"options":{"refactor":0.98},"reason":"refactor detectado"}`
		_, err = client.EvalChoice(context.Background(), semantic.EvalRequest{
			State:     "add new function",
			Question:  "type?",
			Threshold: 0.8,
			Choices:   []string{"feat", "fix"},
		})
		if err == nil {
			t.Fatal("esperava erro ao retornar opção alucinada fora de choices, obteve nil")
		}
		if !errors.Is(err, semantic.ErrInvalidChoice) {
			t.Errorf("esperava ErrInvalidChoice, obteve: %v", err)
		}
	})

	t.Run("EvalScore com sucesso e rejeição SEC-02 de score fora dos limites", func(t *testing.T) {
		var respText string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":%q}]}}]}`, respText)))
		}))
		defer srv.Close()

		client := semantic.NewClient(semantic.ClientConfig{
			Provider:      semantic.ProviderGemini,
			Endpoint:      srv.URL,
			APIKey:        apiKey,
			Model:         semantic.DefaultGeminiModel,
			ThinkingLevel: semantic.DefaultGeminiThinkingLevel,
			Timeout:       2 * time.Second,
		})

		// 1. Sucesso com score válido
		respText = `{"score":8.5,"max_score":10.0,"reason":"boa qualidade"}`
		dec, err := client.EvalScore(context.Background(), semantic.EvalRequest{
			State:     "clean code",
			Question:  "score?",
			Threshold: 0.8,
			MaxScore:  10.0,
		})
		if err != nil {
			t.Fatalf("EvalScore retornou erro: %v", err)
		}
		if dec.Score != 8.5 || dec.MaxScore != 10.0 {
			t.Errorf("Score/MaxScore = %v/%v, quer 8.5/10.0", dec.Score, dec.MaxScore)
		}

		// 2. Score fora dos limites (maior que MaxScore) rejeitado com ErrInvalidScore (SEC-02)
		respText = `{"score":15.0,"max_score":10.0,"reason":"score estourado"}`
		_, err = client.EvalScore(context.Background(), semantic.EvalRequest{
			State:     "code",
			Question:  "score?",
			Threshold: 0.8,
			MaxScore:  10.0,
		})
		if err == nil {
			t.Fatal("esperava erro com score > max_score, obteve nil")
		}
		if !errors.Is(err, semantic.ErrInvalidScore) {
			t.Errorf("esperava ErrInvalidScore, obteve: %v", err)
		}

		// 3. Score negativo rejeitado com ErrInvalidScore (SEC-02)
		respText = `{"score":-2.0,"max_score":10.0,"reason":"score negativo"}`
		_, err = client.EvalScore(context.Background(), semantic.EvalRequest{
			State:     "code",
			Question:  "score?",
			Threshold: 0.8,
			MaxScore:  10.0,
		})
		if err == nil {
			t.Fatal("esperava erro com score negativo, obteve nil")
		}
		if !errors.Is(err, semantic.ErrInvalidScore) {
			t.Errorf("esperava ErrInvalidScore, obteve: %v", err)
		}
	})

	t.Run("SEC-03: Redação de apiKey em erro e MaxBodyBytes", func(t *testing.T) {
		// Redação de erro HTTP contendo APIKey
		srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error leaking " + apiKey))
		}))
		defer srvErr.Close()

		clientErr := semantic.NewClient(semantic.ClientConfig{
			Provider: semantic.ProviderGemini,
			Endpoint: srvErr.URL,
			APIKey:   apiKey,
			Timeout:  2 * time.Second,
		})

		_, err := clientErr.EvalNoul(context.Background(), semantic.EvalRequest{
			State:     "test",
			Question:  "ok?",
			Threshold: 0.8,
		})
		if err == nil {
			t.Fatal("esperava erro do servidor com status 500")
		}
		if strings.Contains(err.Error(), apiKey) {
			t.Fatalf("SEC-03 violação: erro expôs APIKey secreta: %v", err)
		}

		// MaxBodyBytes excedido retorna ErrResponseTooLarge
		srvLarge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"` + strings.Repeat("A", 1000) + `"}]}}]}`))
		}))
		defer srvLarge.Close()

		clientLarge := semantic.NewClient(semantic.ClientConfig{
			Provider:     semantic.ProviderGemini,
			Endpoint:     srvLarge.URL,
			APIKey:       apiKey,
			MaxBodyBytes: 100,
			Timeout:      2 * time.Second,
		})

		_, err = clientLarge.EvalNoul(context.Background(), semantic.EvalRequest{
			State:     "test",
			Question:  "ok?",
			Threshold: 0.8,
		})
		if err == nil {
			t.Fatal("esperava ErrResponseTooLarge, obteve nil")
		}
		if !errors.Is(err, semantic.ErrResponseTooLarge) {
			t.Errorf("esperava ErrResponseTooLarge, obteve: %v", err)
		}
	})

	t.Run("ParseGeminiResponse extrai candidates e rejeita payloads invalidos", func(t *testing.T) {
		t.Run("ParseGeminiResponse válido", func(t *testing.T) {
			validJSON := []byte(`{"candidates":[{"content":{"parts":[{"text":"{\"decision\":true}"}]}}]}`)
			partText, err := semantic.ParseGeminiResponse(validJSON)
			if err != nil {
				t.Fatalf("ParseGeminiResponse retornou erro inesperado: %v", err)
			}
			if string(partText) != `{"decision":true}` {
				t.Errorf("ParseGeminiResponse extraiu %q, quer %q", string(partText), `{"decision":true}`)
			}
		})

		t.Run("ParseGeminiResponse com JSON inválido ou candidates vazio", func(t *testing.T) {
			invalidInputs := [][]byte{
				[]byte(`invalid json`),
				[]byte(`{}`),
				[]byte(`{"candidates":[]}`),
				[]byte(`{"candidates":[{"content":{"parts":[]}}]}`),
			}
			for _, in := range invalidInputs {
				if _, err := semantic.ParseGeminiResponse(in); err == nil {
					t.Errorf("ParseGeminiResponse(%q) esperava erro, obteve nil", string(in))
				}
			}
		})
	})
}
