package dashboard_test

import (
	"testing"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics"
)

func FuzzRenderHTML(f *testing.F) {
	f.Add("pipeline-ai", "api", "DEL-001", "TASK-001", "feat", "Título Exemplo")
	f.Add("", "", "", "", "", "")
	f.Add("<script>alert(1)</script>", "\x00\xff", "DEL-999", "TASK-999", "bugfix", "Caractere especial: \n\r\t")

	f.Fuzz(func(t *testing.T, projName, projType, delivID, task, delivType, title string) {
		data := &dashboard.DashboardData{
			ProjectName: projName,
			ProjectType: projType,
			GeneratedAt: time.Now(),
			DORA: metrics.DORAMetrics{
				DeploymentFrequency: 1.0,
				DeploymentsTotal:    1,
			},
			Pillars: []dashboard.PillarInfo{
				{Name: "Pilar Fuzz", Description: "Desc", Status: "Ativo"},
			},
			Skills: []dashboard.SkillInfo{
				{Name: "skill-fuzz", Description: "Desc"},
			},
			Deliveries: []dashboard.DeliverySummary{
				{
					ID:     delivID,
					Task:   task,
					Title:  title,
					Type:   delivType,
					Date:   "2026-10-28",
					Status: "merged",
				},
			},
		}

		// A renderização nunca deve entrar em pânico
		_, _ = dashboard.RenderHTML(data)
	})
}
