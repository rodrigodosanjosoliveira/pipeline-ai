package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_Tolerance(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string)
		wantNilErr bool
		wantErr    error
		check      func(t *testing.T, data *MetricsData)
	}{
		{
			name: "arquivo ausente deve retornar estrutura inicializada sem erro",
			setup: func(t *testing.T, dir string) {
				// não cria nada
			},
			wantNilErr: true,
			check: func(t *testing.T, data *MetricsData) {
				if data == nil {
					t.Fatal("esperava estrutura de métricas não-nula para arquivo ausente")
				}
				if len(data.Events) != 0 || len(data.Deliveries) != 0 || len(data.Stages) != 0 {
					t.Errorf("esperava dados vazios, obteve %+v", data)
				}
			},
		},
		{
			name: "arquivo vazio deve retornar estrutura inicializada sem erro",
			setup: func(t *testing.T, dir string) {
				stateDir := filepath.Join(dir, DirName)
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatalf("erro ao criar dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(stateDir, FileName), []byte("   \n"), 0o644); err != nil {
					t.Fatalf("erro ao escrever arquivo vazio: %v", err)
				}
			},
			wantNilErr: true,
			check: func(t *testing.T, data *MetricsData) {
				if data == nil {
					t.Fatal("esperava estrutura não-nula para arquivo vazio")
				}
			},
		},
		{
			name: "arquivo corrompido deve retornar erro",
			setup: func(t *testing.T, dir string) {
				stateDir := filepath.Join(dir, DirName)
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatalf("erro ao criar dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(stateDir, FileName), []byte("{invalid-json"), 0o644); err != nil {
					t.Fatalf("erro ao escrever arquivo corrompido: %v", err)
				}
			},
			wantNilErr: false,
			check: func(t *testing.T, data *MetricsData) {
				if data != nil {
					t.Errorf("esperava data nil ao falhar por corrupção, obteve %+v", data)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)

			data, err := Load(dir)
			if tt.wantNilErr && err != nil {
				t.Fatalf("esperava erro nil, obteve: %v", err)
			}
			if !tt.wantNilErr && err == nil {
				t.Fatal("esperava erro, mas obteve nil")
			}
			if tt.check != nil {
				tt.check(t, data)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	now := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	compTime := now.Add(2 * time.Hour)

	tests := []struct {
		name string
		data *MetricsData
	}{
		{
			name: "dados vazios com versão",
			data: &MetricsData{
				Version:   1,
				UpdatedAt: now,
			},
		},
		{
			name: "dados completos com eventos, stages e deliveries",
			data: &MetricsData{
				Version:   1,
				UpdatedAt: now,
				Events: []Event{
					{
						ID:         "evt-1",
						Type:       EventDeployment,
						DeliveryID: "DEL-001",
						Timestamp:  now,
						Duration:   15 * time.Minute,
						Success:    true,
						Message:    "deploy sucesso",
					},
					{
						ID:         "evt-2",
						Type:       EventRestore,
						DeliveryID: "DEL-002",
						Timestamp:  now.Add(30 * time.Minute),
						Duration:   10 * time.Minute,
						Success:    true,
					},
				},
				Stages: []StageDuration{
					{
						Stage:      "doctor",
						DeliveryID: "DEL-001",
						Duration:   150 * time.Millisecond,
						Timestamp:  now,
						Success:    true,
					},
					{
						Stage:      "sensor",
						DeliveryID: "DEL-001",
						Duration:   2 * time.Second,
						Timestamp:  now,
						Success:    true,
					},
				},
				Deliveries: []DeliveryMetric{
					{
						DeliveryID:  "DEL-001",
						TaskCode:    "TASK-001",
						StartedAt:   now,
						CompletedAt: &compTime,
						Success:     true,
						LeadTime:    2 * time.Hour,
						Stages: map[string]time.Duration{
							"doctor": 150 * time.Millisecond,
							"sensor": 2 * time.Second,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if err := Save(dir, tt.data); err != nil {
				t.Fatalf("Save falhou: %v", err)
			}

			loaded, err := Load(dir)
			if err != nil {
				t.Fatalf("Load falhou: %v", err)
			}

			if loaded.Version != tt.data.Version {
				t.Errorf("Version: esperado %d, obtido %d", tt.data.Version, loaded.Version)
			}
			if len(loaded.Events) != len(tt.data.Events) {
				t.Errorf("Events: esperado %d, obtido %d", len(tt.data.Events), len(loaded.Events))
			}
			if len(loaded.Stages) != len(tt.data.Stages) {
				t.Errorf("Stages: esperado %d, obtido %d", len(tt.data.Stages), len(loaded.Stages))
			}
			if len(loaded.Deliveries) != len(tt.data.Deliveries) {
				t.Errorf("Deliveries: esperado %d, obtido %d", len(tt.data.Deliveries), len(loaded.Deliveries))
			}
		})
	}
}

func TestRecordEvent(t *testing.T) {
	tests := []struct {
		name       string
		events     []Event
		wantEvents int
	}{
		{
			name: "gravação de um evento",
			events: []Event{
				{
					Type:       EventDeployment,
					DeliveryID: "DEL-010",
					Timestamp:  time.Now().UTC(),
					Success:    true,
				},
			},
			wantEvents: 1,
		},
		{
			name: "gravação sequencial de múltiplos eventos",
			events: []Event{
				{
					Type:       EventGate,
					DeliveryID: "DEL-010",
					Stage:      "sensor",
					Success:    true,
				},
				{
					Type:       EventDeployment,
					DeliveryID: "DEL-010",
					Success:    false,
					Message:    "falha de validação",
				},
				{
					Type:       EventRestore,
					DeliveryID: "DEL-010",
					Success:    true,
				},
			},
			wantEvents: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for _, evt := range tt.events {
				if err := RecordEvent(dir, evt); err != nil {
					t.Fatalf("RecordEvent falhou: %v", err)
				}
			}

			loaded, err := Load(dir)
			if err != nil {
				t.Fatalf("Load falhou: %v", err)
			}
			if len(loaded.Events) != tt.wantEvents {
				t.Fatalf("esperado %d eventos, obtido %d", tt.wantEvents, len(loaded.Events))
			}
			lastEvt := loaded.Events[len(loaded.Events)-1]
			wantLast := tt.events[len(tt.events)-1]
			if lastEvt.Type != wantLast.Type {
				t.Errorf("Type do último evento: esperado %s, obtido %s", wantLast.Type, lastEvt.Type)
			}
			if lastEvt.Success != wantLast.Success {
				t.Errorf("Success do último evento: esperado %v, obtido %v", wantLast.Success, lastEvt.Success)
			}
		})
	}
}

func TestRecordStageDuration(t *testing.T) {
	tests := []struct {
		name   string
		stages []struct {
			deliveryID string
			stage      string
			duration   time.Duration
			success    bool
		}
		wantStages int
	}{
		{
			name: "gravação de stages individuais",
			stages: []struct {
				deliveryID string
				stage      string
				duration   time.Duration
				success    bool
			}{
				{deliveryID: "DEL-100", stage: "doctor", duration: 100 * time.Millisecond, success: true},
				{deliveryID: "DEL-100", stage: "sensor", duration: 2 * time.Second, success: true},
				{deliveryID: "DEL-100", stage: "validate", duration: 500 * time.Millisecond, success: false},
			},
			wantStages: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for _, s := range tt.stages {
				if err := RecordStageDuration(dir, s.deliveryID, s.stage, s.duration, s.success); err != nil {
					t.Fatalf("RecordStageDuration falhou: %v", err)
				}
			}

			loaded, err := Load(dir)
			if err != nil {
				t.Fatalf("Load falhou: %v", err)
			}
			if len(loaded.Stages) != tt.wantStages {
				t.Fatalf("esperado %d stages, obtido %d", tt.wantStages, len(loaded.Stages))
			}
			if loaded.Stages[0].Stage != "doctor" {
				t.Errorf("primeiro stage: esperado 'doctor', obtido %s", loaded.Stages[0].Stage)
			}
		})
	}
}

func TestCalculateDORA(t *testing.T) {
	baseTime := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	t1 := baseTime.Add(24 * time.Hour)
	t2 := baseTime.Add(48 * time.Hour)
	t3 := baseTime.Add(72 * time.Hour)
	t4 := baseTime.Add(96 * time.Hour)

	c1 := t1.Add(1 * time.Hour)
	c2 := t2.Add(2 * time.Hour)
	c3 := t3.Add(3 * time.Hour)
	c4 := t4.Add(4 * time.Hour)

	tests := []struct {
		name    string
		data    *MetricsData
		opts    []DORAOptions
		want    DORAMetrics
		wantErr bool
	}{
		{
			name: "histórico vazio não causa divisão por zero ou pânico",
			data: &MetricsData{},
			want: DORAMetrics{
				DeploymentFrequency: 0,
				DeploymentsTotal:    0,
				LeadTimeForChanges:  0,
				ChangeFailureRate:   0,
				FailedDeployments:   0,
				TimeToRestore:       0,
				RestoresTotal:       0,
			},
		},
		{
			name: "múltiplas entregas todas com sucesso",
			data: &MetricsData{
				Deliveries: []DeliveryMetric{
					{DeliveryID: "DEL-1", StartedAt: t1, CompletedAt: &c1, Success: true, LeadTime: 1 * time.Hour},
					{DeliveryID: "DEL-2", StartedAt: t2, CompletedAt: &c2, Success: true, LeadTime: 2 * time.Hour},
					{DeliveryID: "DEL-3", StartedAt: t3, CompletedAt: &c3, Success: true, LeadTime: 3 * time.Hour},
					{DeliveryID: "DEL-4", StartedAt: t4, CompletedAt: &c4, Success: true, LeadTime: 4 * time.Hour},
				},
				Events: []Event{
					{Type: EventDeployment, Timestamp: c1, Success: true},
					{Type: EventDeployment, Timestamp: c2, Success: true},
					{Type: EventDeployment, Timestamp: c3, Success: true},
					{Type: EventDeployment, Timestamp: c4, Success: true},
				},
			},
			opts: []DORAOptions{
				{Window: 4 * 24 * time.Hour}, // 4 dias
			},
			want: DORAMetrics{
				DeploymentFrequency: 1.0, // 4 deploys / 4 dias = 1 deploy/dia
				DeploymentsTotal:    4,
				LeadTimeForChanges:  2*time.Hour + 30*time.Minute, // (1h + 2h + 3h + 4h) / 4 = 2.5h
				ChangeFailureRate:   0.0,                          // 0 falhas em 4 deploys = 0%
				FailedDeployments:   0,
				TimeToRestore:       0,
				RestoresTotal:       0,
			},
		},
		{
			name: "histórico misto com falha e restauração (CFR e MTTR)",
			data: &MetricsData{
				Deliveries: []DeliveryMetric{
					{DeliveryID: "DEL-1", StartedAt: t1, CompletedAt: &c1, Success: true, LeadTime: 2 * time.Hour},
					{DeliveryID: "DEL-2", StartedAt: t2, CompletedAt: &c2, Success: true, LeadTime: 2 * time.Hour},
					{DeliveryID: "DEL-3", StartedAt: t3, CompletedAt: &c3, Success: true, LeadTime: 2 * time.Hour},
					{DeliveryID: "DEL-4", StartedAt: t4, CompletedAt: &c4, Success: false, LeadTime: 2 * time.Hour},
				},
				Events: []Event{
					{Type: EventDeployment, Timestamp: c1, Success: true},
					{Type: EventDeployment, Timestamp: c2, Success: true},
					{Type: EventDeployment, Timestamp: c3, Success: true},
					{Type: EventDeployment, Timestamp: c4, Success: false},
					// Incidente e restauração
					{Type: EventRestore, Timestamp: c4.Add(30 * time.Minute), Duration: 30 * time.Minute, Success: true},
				},
			},
			opts: []DORAOptions{
				{Window: 4 * 24 * time.Hour},
			},
			want: DORAMetrics{
				DeploymentFrequency: 1.0,
				DeploymentsTotal:    4,
				LeadTimeForChanges:  2 * time.Hour,
				ChangeFailureRate:   25.0, // 1 falha em 4 = 25%
				FailedDeployments:   1,
				TimeToRestore:       30 * time.Minute,
				RestoresTotal:       1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDORA(tt.data, tt.opts...)

			if got.DeploymentsTotal != tt.want.DeploymentsTotal {
				t.Errorf("DeploymentsTotal: esperado %d, obtido %d", tt.want.DeploymentsTotal, got.DeploymentsTotal)
			}
			if got.DeploymentFrequency != tt.want.DeploymentFrequency {
				t.Errorf("DeploymentFrequency: esperado %v, obtido %v", tt.want.DeploymentFrequency, got.DeploymentFrequency)
			}
			if got.LeadTimeForChanges != tt.want.LeadTimeForChanges {
				t.Errorf("LeadTimeForChanges: esperado %v, obtido %v", tt.want.LeadTimeForChanges, got.LeadTimeForChanges)
			}
			if got.ChangeFailureRate != tt.want.ChangeFailureRate {
				t.Errorf("ChangeFailureRate: esperado %v, obtido %v", tt.want.ChangeFailureRate, got.ChangeFailureRate)
			}
			if got.FailedDeployments != tt.want.FailedDeployments {
				t.Errorf("FailedDeployments: esperado %d, obtido %d", tt.want.FailedDeployments, got.FailedDeployments)
			}
			if got.TimeToRestore != tt.want.TimeToRestore {
				t.Errorf("TimeToRestore: esperado %v, obtido %v", tt.want.TimeToRestore, got.TimeToRestore)
			}
			if got.RestoresTotal != tt.want.RestoresTotal {
				t.Errorf("RestoresTotal: esperado %d, obtido %d", tt.want.RestoresTotal, got.RestoresTotal)
			}
		})
	}
}

func TestGetDORA(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	comp := now.Add(1 * time.Hour)

	data := &MetricsData{
		Version: 1,
		Deliveries: []DeliveryMetric{
			{DeliveryID: "DEL-1", StartedAt: now, CompletedAt: &comp, Success: true, LeadTime: 1 * time.Hour},
		},
		Events: []Event{
			{Type: EventDeployment, Timestamp: comp, Success: true},
		},
	}

	if err := Save(dir, data); err != nil {
		t.Fatalf("Save falhou: %v", err)
	}

	dora, err := GetDORA(dir)
	if err != nil {
		t.Fatalf("GetDORA falhou: %v", err)
	}
	if dora.DeploymentsTotal != 1 {
		t.Errorf("DeploymentsTotal: esperado 1, obtido %d", dora.DeploymentsTotal)
	}
}
