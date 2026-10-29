package semantic_test

import (
	"context"
	"errors"
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
