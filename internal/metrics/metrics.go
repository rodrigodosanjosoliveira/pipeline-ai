// Package metrics coleta, persiste e calcula métricas de engenharia e governança (DORA).
package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Constantes de diretório e arquivo de métricas.
const (
	DirName  = ".pipeline-state"
	FileName = "metrics.json"
)

// Sentinel errors do pacote metrics.
var (
	ErrCorruptedMetrics = errors.New("métricas corrompidas")
	ErrNotFound         = errors.New("arquivo de métricas não encontrado")
	ErrNotImplemented   = errors.New("não implementado")
)

var mu sync.Mutex

// EventType categoriza os eventos observados no pipeline.
type EventType string

const (
	EventDeployment EventType = "deployment"
	EventRestore    EventType = "restore"
	EventGate       EventType = "gate"
	EventStage      EventType = "stage"
	EventDelivery   EventType = "delivery"
)

// Event registra uma ocorrência pontual no ciclo de vida do pipeline.
type Event struct {
	ID         string            `json:"id,omitempty"`
	Type       EventType         `json:"type"`
	DeliveryID string            `json:"delivery_id,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Duration   time.Duration     `json:"duration,omitempty"`
	Success    bool              `json:"success"`
	Stage      string            `json:"stage,omitempty"`
	Message    string            `json:"message,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// StageDuration registra o tempo gasto em uma etapa específica.
type StageDuration struct {
	Stage      string        `json:"stage"`
	DeliveryID string        `json:"delivery_id,omitempty"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`
	Success    bool          `json:"success"`
}

// DeliveryMetric consolida o ciclo de uma delivery.
type DeliveryMetric struct {
	DeliveryID  string                   `json:"delivery_id"`
	TaskCode    string                   `json:"task_code,omitempty"`
	StartedAt   time.Time                `json:"started_at"`
	CompletedAt *time.Time               `json:"completed_at,omitempty"`
	Success     bool                     `json:"success"`
	LeadTime    time.Duration            `json:"lead_time,omitempty"`
	Stages      map[string]time.Duration `json:"stages,omitempty"`
}

// MetricsData representa a base persistida de métricas.
type MetricsData struct {
	Version    int              `json:"version"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Events     []Event          `json:"events,omitempty"`
	Stages     []StageDuration  `json:"stages,omitempty"`
	Deliveries []DeliveryMetric `json:"deliveries,omitempty"`
}

// DORAMetrics consolida as quatro métricas-chave DORA.
type DORAMetrics struct {
	DeploymentFrequency float64       `json:"deployment_frequency"` // Deploys por dia
	DeploymentsTotal    int           `json:"deployments_total"`
	LeadTimeForChanges  time.Duration `json:"lead_time_for_changes"` // Lead time médio
	ChangeFailureRate   float64       `json:"change_failure_rate"`   // Taxa percentual de falha (0.0 a 100.0)
	FailedDeployments   int           `json:"failed_deployments"`
	TimeToRestore       time.Duration `json:"time_to_restore"` // MTTR médio
	RestoresTotal       int           `json:"restores_total"`
}

// DORAOptions configura parâmetros de janela temporal para cálculo DORA.
type DORAOptions struct {
	Window time.Duration
	Since  time.Time
	Until  time.Time
}

// Aliases para compatibilidade de nomenclatura.
type (
	MetricEvent     = Event
	StageMetric     = StageDuration
	DeliveryMetrics = DeliveryMetric
	DORASummary     = DORAMetrics
)

// Store encapsula operações de persistência e cálculo de métricas para um workspace.
type Store struct {
	rootDir string
}

// NewStore cria uma nova instância de Store para rootDir.
func NewStore(rootDir string) *Store {
	if rootDir == "" {
		rootDir = "."
	}
	return &Store{rootDir: rootDir}
}

// Load carrega as métricas da Store.
func (s *Store) Load() (*MetricsData, error) {
	return Load(s.rootDir)
}

// Save persiste as métricas na Store.
func (s *Store) Save(data *MetricsData) error {
	return Save(s.rootDir, data)
}

// RecordEvent adiciona um evento na Store.
func (s *Store) RecordEvent(event Event) error {
	return RecordEvent(s.rootDir, event)
}

// RecordStageDuration registra a duração de um estágio na Store.
func (s *Store) RecordStageDuration(deliveryID, stage string, duration time.Duration, success ...bool) error {
	return RecordStageDuration(s.rootDir, deliveryID, stage, duration, success...)
}

// CalculateDORA calcula as métricas DORA sobre os dados da Store.
func (s *Store) CalculateDORA(opts ...DORAOptions) (*DORAMetrics, error) {
	return GetDORA(s.rootDir, opts...)
}

// ParseMetrics converte bytes JSON na estrutura MetricsData com tolerância fail-safe.
func ParseMetrics(data []byte) (*MetricsData, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return &MetricsData{
			Version:    1,
			Events:     []Event{},
			Stages:     []StageDuration{},
			Deliveries: []DeliveryMetric{},
		}, nil
	}

	var md MetricsData
	if err := json.Unmarshal(trimmed, &md); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptedMetrics, err)
	}

	if md.Events == nil {
		md.Events = []Event{}
	}
	if md.Stages == nil {
		md.Stages = []StageDuration{}
	}
	if md.Deliveries == nil {
		md.Deliveries = []DeliveryMetric{}
	}
	if md.Version == 0 {
		md.Version = 1
	}

	return &md, nil
}

// Load carrega as métricas persistidas em rootDir.
func Load(rootDir string) (*MetricsData, error) {
	stateDir := filepath.Join(rootDir, DirName)
	metricsFile := filepath.Join(stateDir, FileName)

	data, err := os.ReadFile(metricsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &MetricsData{
				Version:    1,
				Events:     []Event{},
				Stages:     []StageDuration{},
				Deliveries: []DeliveryMetric{},
			}, nil
		}
		return nil, err
	}

	return ParseMetrics(data)
}

// Save persiste as métricas no arquivo padrão dentro de rootDir.
func Save(rootDir string, data *MetricsData) error {
	mu.Lock()
	defer mu.Unlock()

	if data == nil {
		data = &MetricsData{Version: 1}
	}
	if data.Version == 0 {
		data.Version = 1
	}
	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = time.Now().UTC()
	}
	if data.Events == nil {
		data.Events = []Event{}
	}
	if data.Stages == nil {
		data.Stages = []StageDuration{}
	}
	if data.Deliveries == nil {
		data.Deliveries = []DeliveryMetric{}
	}

	stateDir := filepath.Join(rootDir, DirName)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("erro ao criar diretório %s: %w", stateDir, err)
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar métricas: %w", err)
	}

	tmpFile := filepath.Join(stateDir, fmt.Sprintf("%s.tmp.%d", FileName, time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, raw, 0o644); err != nil {
		return fmt.Errorf("erro ao escrever arquivo temporário de métricas: %w", err)
	}

	targetFile := filepath.Join(stateDir, FileName)
	if err := os.Rename(tmpFile, targetFile); err != nil {
		_ = os.Remove(tmpFile)
		if writeErr := os.WriteFile(targetFile, raw, 0o644); writeErr != nil {
			return fmt.Errorf("erro ao persistir métricas: %w", writeErr)
		}
	}

	return nil
}

// RecordEvent adiciona um evento ao registro de métricas de rootDir.
func RecordEvent(rootDir string, event Event) error {
	data, err := Load(rootDir)
	if err != nil {
		return err
	}

	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", len(data.Events)+1)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	data.Events = append(data.Events, event)
	data.UpdatedAt = time.Now().UTC()

	return Save(rootDir, data)
}

// RecordStageDuration registra a duração de uma etapa em rootDir.
func RecordStageDuration(rootDir string, deliveryID, stage string, duration time.Duration, success ...bool) error {
	succ := true
	if len(success) > 0 {
		succ = success[0]
	}

	data, err := Load(rootDir)
	if err != nil {
		return err
	}

	sd := StageDuration{
		Stage:      stage,
		DeliveryID: deliveryID,
		Duration:   duration,
		Timestamp:  time.Now().UTC(),
		Success:    succ,
	}

	data.Stages = append(data.Stages, sd)
	data.UpdatedAt = time.Now().UTC()

	return Save(rootDir, data)
}

// CalculateDORA calcula as 4 métricas DORA sobre o conjunto de dados fornecido.
func CalculateDORA(data *MetricsData, opts ...DORAOptions) DORAMetrics {
	if data == nil {
		return DORAMetrics{}
	}

	var opt DORAOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	filterEvent := func(t time.Time) bool {
		if !opt.Since.IsZero() && t.Before(opt.Since) {
			return false
		}
		if !opt.Until.IsZero() && t.After(opt.Until) {
			return false
		}
		return true
	}

	// 1. Deployments & CFR
	var deploymentsTotal int
	var failedDeployments int
	var hasDeploymentEvents bool

	for _, e := range data.Events {
		if e.Type == EventDeployment {
			if !filterEvent(e.Timestamp) {
				continue
			}
			hasDeploymentEvents = true
			deploymentsTotal++
			if !e.Success {
				failedDeployments++
			}
		}
	}

	if !hasDeploymentEvents && len(data.Deliveries) > 0 {
		for _, d := range data.Deliveries {
			t := d.StartedAt
			if d.CompletedAt != nil {
				t = *d.CompletedAt
			}
			if !filterEvent(t) {
				continue
			}
			deploymentsTotal++
			if !d.Success {
				failedDeployments++
			}
		}
	}

	var cfr float64
	if deploymentsTotal > 0 {
		cfr = (float64(failedDeployments) / float64(deploymentsTotal)) * 100.0
	}

	// 2. Deployment Frequency
	var window time.Duration
	if opt.Window > 0 {
		window = opt.Window
	} else if !opt.Since.IsZero() && !opt.Until.IsZero() {
		window = opt.Until.Sub(opt.Since)
	}

	var deployFreq float64
	if deploymentsTotal > 0 {
		if window > 0 {
			days := window.Hours() / 24.0
			if days > 0 {
				deployFreq = float64(deploymentsTotal) / days
			} else {
				deployFreq = float64(deploymentsTotal)
			}
		} else {
			var minT, maxT time.Time
			for _, e := range data.Events {
				if e.Type == EventDeployment {
					if minT.IsZero() || e.Timestamp.Before(minT) {
						minT = e.Timestamp
					}
					if maxT.IsZero() || e.Timestamp.After(maxT) {
						maxT = e.Timestamp
					}
				}
			}
			if minT.IsZero() {
				for _, d := range data.Deliveries {
					t := d.StartedAt
					if d.CompletedAt != nil {
						t = *d.CompletedAt
					}
					if minT.IsZero() || t.Before(minT) {
						minT = t
					}
					if maxT.IsZero() || t.After(maxT) {
						maxT = t
					}
				}
			}
			span := maxT.Sub(minT)
			days := span.Hours() / 24.0
			if days < 1.0 {
				days = 1.0
			}
			deployFreq = float64(deploymentsTotal) / days
		}
	}

	// 3. Lead Time For Changes (tempo médio das deliveries concluídas com sucesso)
	var totalLeadTime time.Duration
	var leadTimeCount int
	for _, d := range data.Deliveries {
		t := d.StartedAt
		if d.CompletedAt != nil {
			t = *d.CompletedAt
		}
		if !filterEvent(t) {
			continue
		}
		if !d.Success {
			continue
		}
		lt := d.LeadTime
		if lt == 0 && d.CompletedAt != nil && !d.StartedAt.IsZero() {
			lt = d.CompletedAt.Sub(d.StartedAt)
		}
		if lt > 0 {
			totalLeadTime += lt
			leadTimeCount++
		}
	}

	var avgLeadTime time.Duration
	if leadTimeCount > 0 {
		avgLeadTime = time.Duration(int64(totalLeadTime) / int64(leadTimeCount))
	} else if deploymentsTotal > 0 {
		var evtDur time.Duration
		var evtCount int
		for _, e := range data.Events {
			if e.Type == EventDeployment && e.Success && e.Duration > 0 && filterEvent(e.Timestamp) {
				evtDur += e.Duration
				evtCount++
			}
		}
		if evtCount > 0 {
			avgLeadTime = time.Duration(int64(evtDur) / int64(evtCount))
		}
	}

	// 4. Time To Restore (MTTR)
	var totalRestoreDuration time.Duration
	var restoresTotal int
	for _, e := range data.Events {
		if e.Type == EventRestore {
			if !filterEvent(e.Timestamp) {
				continue
			}
			restoresTotal++
			if e.Duration > 0 {
				totalRestoreDuration += e.Duration
			}
		}
	}

	var mttr time.Duration
	if restoresTotal > 0 {
		mttr = time.Duration(int64(totalRestoreDuration) / int64(restoresTotal))
	}

	return DORAMetrics{
		DeploymentFrequency: deployFreq,
		DeploymentsTotal:    deploymentsTotal,
		LeadTimeForChanges:  avgLeadTime,
		ChangeFailureRate:   cfr,
		FailedDeployments:   failedDeployments,
		TimeToRestore:       mttr,
		RestoresTotal:       restoresTotal,
	}
}

// GetDORA carrega os dados de rootDir e calcula as métricas DORA.
func GetDORA(rootDir string, opts ...DORAOptions) (*DORAMetrics, error) {
	data, err := Load(rootDir)
	if err != nil {
		return nil, err
	}
	dora := CalculateDORA(data, opts...)
	return &dora, nil
}
