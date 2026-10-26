// Package state gerencia o ciclo de vida e persistência da delivery ativa do repositório.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DirName é o nome do diretório de estado na raiz do projeto.
const DirName = ".pipeline-state"

// FileName é o nome do arquivo que armazena os metadados da delivery ativa.
const FileName = "active-delivery.json"

// Sentinel errors do pacote state.
var (
	ErrNoActiveDelivery = errors.New("nenhuma delivery ativa")
	ErrExpired          = errors.New("delivery ativa expirada")
	ErrInvalidParams    = errors.New("parâmetros inválidos")
)

// ActiveDelivery representa os metadados da entrega em andamento.
type ActiveDelivery struct {
	DeliveryID string    `json:"delivery_id"`
	TaskCode   string    `json:"task_code,omitempty"`
	OpenedAt   time.Time `json:"opened_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Open registra uma nova delivery ativa com o TTL especificado.
func Open(rootDir, deliveryID, taskCode string, ttl time.Duration) error {
	if strings.TrimSpace(deliveryID) == "" {
		return fmt.Errorf("%w: delivery_id é obrigatório", ErrInvalidParams)
	}
	if ttl <= 0 {
		return fmt.Errorf("%w: ttl deve ser positivo", ErrInvalidParams)
	}

	now := time.Now().UTC()
	active := ActiveDelivery{
		DeliveryID: strings.TrimSpace(deliveryID),
		TaskCode:   strings.TrimSpace(taskCode),
		OpenedAt:   now,
		ExpiresAt:  now.Add(ttl),
	}

	stateDir := filepath.Join(rootDir, DirName)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("state: falha ao criar diretório %s: %w", stateDir, err)
	}

	data, err := json.MarshalIndent(active, "", "  ")
	if err != nil {
		return fmt.Errorf("state: falha ao serializar json: %w", err)
	}

	stateFile := filepath.Join(stateDir, FileName)
	if err := os.WriteFile(stateFile, data, 0o644); err != nil {
		return fmt.Errorf("state: falha ao escrever %s: %w", stateFile, err)
	}

	return nil
}

// Get lê e decodifica a delivery ativa em rootDir.
func Get(rootDir string) (*ActiveDelivery, error) {
	stateFile := filepath.Join(rootDir, DirName, FileName)
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoActiveDelivery
		}
		return nil, fmt.Errorf("state: falha ao ler %s: %w", stateFile, err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, ErrNoActiveDelivery
	}

	var active ActiveDelivery
	if err := json.Unmarshal(data, &active); err != nil {
		return nil, fmt.Errorf("state: json inválido em %s: %w", stateFile, err)
	}

	if active.DeliveryID == "" {
		return nil, ErrNoActiveDelivery
	}

	return &active, nil
}

// IsActive verifica se existe uma delivery aberta e dentro do prazo de validade.
func IsActive(rootDir string) (bool, error) {
	active, err := Get(rootDir)
	if err != nil {
		return false, err
	}

	if time.Now().UTC().After(active.ExpiresAt) {
		return false, ErrExpired
	}

	return true, nil
}

// Close encerra a delivery ativa removendo o arquivo de estado.
func Close(rootDir string) error {
	stateFile := filepath.Join(rootDir, DirName, FileName)
	err := os.Remove(stateFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("state: falha ao remover %s: %w", stateFile, err)
	}
	return nil
}
