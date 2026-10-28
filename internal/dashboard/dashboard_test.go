package dashboard_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics"
)

func setupTestWorkspace(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Cria .pipeline-ai.json
	cfgJSON := `{
		"version": 1,
		"project_type": "api",
		"protected_paths": ["internal/", "cmd/"],
		"editor_agents": ["dev-implementation", "test-writer"],
		"enabled_modules": ["release-ops"]
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("falha ao criar .pipeline-ai.json: %v", err)
	}

	// Cria diretório de métricas e metrics.json
	stateDir := filepath.Join(tmpDir, ".pipeline-state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("falha ao criar .pipeline-state: %v", err)
	}
	now := time.Now()
	mStore := metrics.NewStore(tmpDir)
	_ = mStore.RecordEvent(metrics.Event{
		Type:       metrics.EventDeployment,
		DeliveryID: "DEL-001",
		Timestamp:  now.Add(-2 * time.Hour),
		Success:    true,
	})
	_ = mStore.RecordEvent(metrics.Event{
		Type:       metrics.EventDeployment,
		DeliveryID: "DEL-002",
		Timestamp:  now.Add(-1 * time.Hour),
		Success:    false,
	})
	_ = mStore.RecordEvent(metrics.Event{
		Type:       metrics.EventRestore,
		DeliveryID: "DEL-002",
		Timestamp:  now.Add(-30 * time.Minute),
		Duration:   30 * time.Minute,
		Success:    true,
	})

	// Cria docs-ai/deliveries/INDEX.md
	delivDir := filepath.Join(tmpDir, "docs-ai", "deliveries")
	if err := os.MkdirAll(delivDir, 0755); err != nil {
		t.Fatalf("falha ao criar docs-ai/deliveries: %v", err)
	}
	indexContent := `# Índice de Deliveries

| ID | Task | Título | Tipo | Data | Status |
|---|---|---|---|---|---|
| DEL-002 | TASK-002 | Correção crítica no sensor | bugfix | 2026-10-28 | merged |
| DEL-001 | TASK-001 | Implementação de telemetria DORA | feat | 2026-10-27 | merged |
`
	if err := os.WriteFile(filepath.Join(delivDir, "INDEX.md"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("falha ao criar INDEX.md: %v", err)
	}

	return tmpDir
}

func TestCollectDashboardData(t *testing.T) {
	tests := []struct {
		name      string
		setupFn   func(t *testing.T) string
		wantErr   bool
		checkData func(t *testing.T, data *dashboard.DashboardData)
	}{
		{
			name:    "workspace completo com métricas e INDEX.md",
			setupFn: setupTestWorkspace,
			wantErr: false,
			checkData: func(t *testing.T, data *dashboard.DashboardData) {
				if data == nil {
					t.Fatalf("esperava DashboardData não nil")
				}
				if len(data.Deliveries) != 2 {
					t.Errorf("len(Deliveries) = %d, esperado 2", len(data.Deliveries))
				}
				if data.Deliveries[0].ID != "DEL-002" {
					t.Errorf("Deliveries[0].ID = %q, esperado DEL-002", data.Deliveries[0].ID)
				}
				if len(data.Pillars) < 4 {
					t.Errorf("len(Pillars) = %d, esperado ao menos 4", len(data.Pillars))
				}
				if len(data.Skills) == 0 {
					t.Errorf("esperava esteira de skills populada")
				}
				if data.DORA.DeploymentsTotal != 2 {
					t.Errorf("DORA.DeploymentsTotal = %d, esperado 2", data.DORA.DeploymentsTotal)
				}
			},
		},
		{
			name: "workspace sem metrics.json",
			setupFn: func(t *testing.T) string {
				dir := t.TempDir()
				delivDir := filepath.Join(dir, "docs-ai", "deliveries")
				_ = os.MkdirAll(delivDir, 0755)
				_ = os.WriteFile(filepath.Join(delivDir, "INDEX.md"), []byte("| ID | Task | Título | Tipo | Data | Status |\n|---|---|---|---|---|---|\n| DEL-001 | TASK-001 | Demo | feat | 2026-10-28 | draft |\n"), 0644)
				return dir
			},
			wantErr: false,
			checkData: func(t *testing.T, data *dashboard.DashboardData) {
				if data == nil {
					t.Fatalf("esperava DashboardData não nil")
				}
				if len(data.Deliveries) != 1 {
					t.Errorf("len(Deliveries) = %d, esperado 1", len(data.Deliveries))
				}
				if data.DORA.DeploymentsTotal != 0 {
					t.Errorf("DORA.DeploymentsTotal = %d, esperado 0", data.DORA.DeploymentsTotal)
				}
			},
		},
		{
			name: "workspace sem INDEX.md",
			setupFn: func(t *testing.T) string {
				dir := t.TempDir()
				stateDir := filepath.Join(dir, ".pipeline-state")
				_ = os.MkdirAll(stateDir, 0755)
				return dir
			},
			wantErr: false,
			checkData: func(t *testing.T, data *dashboard.DashboardData) {
				if data == nil {
					t.Fatalf("esperava DashboardData não nil")
				}
				if len(data.Deliveries) != 0 {
					t.Errorf("len(Deliveries) = %d, esperado 0", len(data.Deliveries))
				}
			},
		},
		{
			name: "workspace vazio e limpo (diretório temporário vazio)",
			setupFn: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
			checkData: func(t *testing.T, data *dashboard.DashboardData) {
				if data == nil {
					t.Fatalf("esperava DashboardData não nil mesmo em diretório vazio")
				}
				if len(data.Deliveries) != 0 {
					t.Errorf("len(Deliveries) = %d, esperado 0", len(data.Deliveries))
				}
				if len(data.Pillars) < 4 {
					t.Errorf("len(Pillars) = %d, esperado 4 pilares padrão", len(data.Pillars))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupFn(t)
			data, err := dashboard.CollectDashboardData(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CollectDashboardData() erro = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.checkData != nil {
				tt.checkData(t, data)
			}
		})
	}
}

func TestRenderHTML(t *testing.T) {
	tests := []struct {
		name         string
		data         *dashboard.DashboardData
		wantErr      bool
		wantContains []string
	}{
		{
			name: "renderização de dashboard completo com métricas e entregas",
			data: &dashboard.DashboardData{
				ProjectName: "pipeline-ai",
				ProjectType: "api",
				GeneratedAt: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC),
				DORA: metrics.DORAMetrics{
					DeploymentFrequency: 1.5,
					DeploymentsTotal:    3,
					LeadTimeForChanges:  45 * time.Minute,
					ChangeFailureRate:   33.3,
					FailedDeployments:   1,
					TimeToRestore:       15 * time.Minute,
					RestoresTotal:       1,
				},
				Pillars: []dashboard.PillarInfo{
					{Name: "PreToolUse Fail-Closed", Description: "Bloqueio determinístico de edição direta", Status: "Ativo"},
					{Name: "Invariantes Globais", Description: "Go 1.24+, stdlib testing, zero CGO desktop", Status: "Ativo"},
					{Name: "Gates Determinísticos", Description: "Suíte integrada de validação doctor/sensor", Status: "Ativo"},
					{Name: "Rastreabilidade e Segurança", Description: "Contratos de handoff e shift-left security", Status: "Ativo"},
				},
				Skills: []dashboard.SkillInfo{
					{Name: "pm-triage", Description: "Entrada e classificação"},
					{Name: "architecture-guardrails", Description: "Limites de arquitetura"},
					{Name: "test-writer", Description: "Testes Red phase"},
					{Name: "dev-implementation", Description: "Implementação Green phase"},
				},
				Deliveries: []dashboard.DeliverySummary{
					{
						ID:     "DEL-022",
						Task:   "TASK-022",
						Title:  "Dashboard HTML interativo de engenharia",
						Type:   "feat",
						Date:   "2026-10-28",
						Status: "draft",
					},
				},
				TotalDeploys: 3,
			},
			wantErr: false,
			wantContains: []string{
				"<!DOCTYPE html>",
				"<html",
				"tailwindcss",
				"pipeline-ai",
				"Deployment Frequency",
				"Lead Time for Changes",
				"Change Failure Rate",
				"Time to Restore",
				"PreToolUse Fail-Closed",
				"Invariantes Globais",
				"Gates Determinísticos",
				"Rastreabilidade e Segurança",
				"pm-triage",
				"test-writer",
				"dev-implementation",
				"DEL-022",
				"TASK-022",
				"Dashboard HTML interativo de engenharia",
				"draft",
			},
		},
		{
			name: "renderização tolerante com dados vazios (placeholders limpos)",
			data: &dashboard.DashboardData{
				ProjectName: "projeto-novo",
				ProjectType: "desktop",
				Deliveries:  []dashboard.DeliverySummary{},
			},
			wantErr: false,
			wantContains: []string{
				"<!DOCTYPE html>",
				"<html",
				"tailwindcss",
				"projeto-novo",
			},
		},
		{
			name:    "erro quando dados são nil",
			data:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := dashboard.RenderHTML(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RenderHTML() erro = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				for _, expected := range tt.wantContains {
					if !strings.Contains(html, expected) {
						t.Errorf("HTML renderizado não contém %q", expected)
					}
				}
			}
		})
	}
}
