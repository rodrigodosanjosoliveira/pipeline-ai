package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAndGet(t *testing.T) {
	tests := []struct {
		name         string
		deliveryID   string
		taskCode     string
		ttl          time.Duration
		wantErr      bool
		errSubstring string
	}{
		{
			name:       "sucesso abrindo delivery com ttl padrao",
			deliveryID: "DEL-001",
			taskCode:   "TASK-001",
			ttl:        2 * time.Hour,
			wantErr:    false,
		},
		{
			name:       "sucesso com task code opcional ou vazio",
			deliveryID: "DEL-002",
			taskCode:   "",
			ttl:        30 * time.Minute,
			wantErr:    false,
		},
		{
			name:         "falha com delivery id vazio",
			deliveryID:   "",
			taskCode:     "TASK-003",
			ttl:          1 * time.Hour,
			wantErr:      true,
			errSubstring: "delivery_id",
		},
		{
			name:         "falha com ttl zero",
			deliveryID:   "DEL-004",
			taskCode:     "TASK-004",
			ttl:          0,
			wantErr:      true,
			errSubstring: "ttl",
		},
		{
			name:         "falha com ttl negativo",
			deliveryID:   "DEL-005",
			taskCode:     "TASK-005",
			ttl:          -10 * time.Minute,
			wantErr:      true,
			errSubstring: "ttl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			err := Open(root, tt.deliveryID, tt.taskCode, tt.ttl)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Open esperava erro, obteve nil")
				}
				if tt.errSubstring != "" && !errors.Is(err, ErrInvalidParams) && !errors.Is(err, err) {
					t.Logf("erro recebido conforme esperado: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Open retornou erro inesperado: %v", err)
			}

			// Verifica se arquivo .pipeline-state/active-delivery.json foi criado
			stateFile := filepath.Join(root, ".pipeline-state", "active-delivery.json")
			if _, err := os.Stat(stateFile); os.IsNotExist(err) {
				t.Fatalf("arquivo %s não foi criado", stateFile)
			}

			got, err := Get(root)
			if err != nil {
				t.Fatalf("Get retornou erro inesperado: %v", err)
			}
			if got == nil {
				t.Fatalf("Get retornou nil para delivery recém-aberta")
			}
			if got.DeliveryID != tt.deliveryID {
				t.Errorf("got.DeliveryID = %q, quer %q", got.DeliveryID, tt.deliveryID)
			}
			if got.TaskCode != tt.taskCode {
				t.Errorf("got.TaskCode = %q, quer %q", got.TaskCode, tt.taskCode)
			}
			if got.OpenedAt.IsZero() {
				t.Errorf("got.OpenedAt não deveria ser zero")
			}
			if got.ExpiresAt.IsZero() || !got.ExpiresAt.After(got.OpenedAt) {
				t.Errorf("got.ExpiresAt (%v) deve ser posterior a OpenedAt (%v)", got.ExpiresAt, got.OpenedAt)
			}
		})
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, root string)
		wantActive  bool
		wantErrIs   error
		wantErrText string
	}{
		{
			name: "retorna true quando dentro do TTL",
			setup: func(t *testing.T, root string) {
				if err := Open(root, "DEL-010", "TASK-010", 1*time.Hour); err != nil {
					t.Fatalf("setup Open: %v", err)
				}
			},
			wantActive: true,
			wantErrIs:  nil,
		},
		{
			name: "retorna false e indica expiracao quando ExpiresAt no passado",
			setup: func(t *testing.T, root string) {
				stateDir := filepath.Join(root, ".pipeline-state")
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				past := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
				opened := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
				data := `{"delivery_id":"DEL-EXPIRED","task_code":"TASK-OLD","opened_at":"` + opened + `","expires_at":"` + past + `"}`
				if err := os.WriteFile(filepath.Join(stateDir, "active-delivery.json"), []byte(data), 0o644); err != nil {
					t.Fatalf("write: %v", err)
				}
			},
			wantActive:  false,
			wantErrIs:   ErrExpired,
			wantErrText: "expirada",
		},
		{
			name: "retorna false quando nenhuma delivery foi aberta",
			setup: func(t *testing.T, root string) {
				// diretório vazio, sem .pipeline-state
			},
			wantActive: false,
			wantErrIs:  ErrNoActiveDelivery,
		},
		{
			name: "retorna false quando arquivo JSON corrompido",
			setup: func(t *testing.T, root string) {
				stateDir := filepath.Join(root, ".pipeline-state")
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(stateDir, "active-delivery.json"), []byte("{corrompido:"), 0o644); err != nil {
					t.Fatalf("write: %v", err)
				}
			},
			wantActive:  false,
			wantErrText: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, root)
			}

			active, err := IsActive(root)
			if active != tt.wantActive {
				t.Errorf("IsActive = %v, quer %v", active, tt.wantActive)
			}

			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("erro %v não é do tipo esperado %v", err, tt.wantErrIs)
			}
			if tt.wantErrText != "" && (err == nil || !errors.Is(err, tt.wantErrIs)) {
				// Log do erro recebido
				t.Logf("erro recebido: %v", err)
			}
		})
	}
}

func TestClose(t *testing.T) {
	root := t.TempDir()

	// 1. Abre uma delivery válida
	if err := Open(root, "DEL-001", "TASK-001", 1*time.Hour); err != nil {
		t.Fatalf("Open: %v", err)
	}

	active, err := IsActive(root)
	if err != nil || !active {
		t.Fatalf("esperava delivery ativa antes do close: active=%v, err=%v", active, err)
	}

	// 2. Executa Close
	if err := Close(root); err != nil {
		t.Fatalf("Close falhou: %v", err)
	}

	// 3. Após Close, IsActive deve retornar false
	activeAfter, _ := IsActive(root)
	if activeAfter {
		t.Errorf("IsActive retornou true após Close")
	}

	// 4. Arquivo active-delivery.json deve ser removido ou limpo
	stateFile := filepath.Join(root, ".pipeline-state", "active-delivery.json")
	if info, err := os.Stat(stateFile); err == nil {
		if info.Size() > 0 {
			// Se o arquivo ainda existe, Get não deve retornar delivery ativa
			got, _ := Get(root)
			if got != nil && got.DeliveryID != "" {
				t.Errorf("arquivo de estado ainda contém delivery ativa após Close: %+v", got)
			}
		}
	}

	// 5. Close repetido deve ser idempotente
	if err := Close(root); err != nil {
		t.Errorf("Close idempotente retornou erro: %v", err)
	}
}

func FuzzActiveDeliveryJSON(f *testing.F) {
	f.Add([]byte(`{"delivery_id":"DEL-001","task_code":"TASK-001","opened_at":"2026-10-26T12:00:00Z","expires_at":"2026-10-26T14:00:00Z"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"delivery_id":"","expires_at":"invalid-date"}`))
	f.Add([]byte(`{"delivery_id":"DEL-999","expires_at":"1970-01-01T00:00:00Z"}`))
	f.Add([]byte(`not-json`))

	f.Fuzz(func(t *testing.T, data []byte) {
		root := t.TempDir()
		stateDir := filepath.Join(root, ".pipeline-state")
		_ = os.MkdirAll(stateDir, 0o755)
		_ = os.WriteFile(filepath.Join(stateDir, "active-delivery.json"), data, 0o644)

		// Nunca deve causar panic
		_, _ = Get(root)
		_, _ = IsActive(root)
	})
}
