package runner

import (
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics"
)

func TestRunner_RecordsMetrics(t *testing.T) {
	tests := []struct {
		name       string
		deliveryID string
		taskCode   string
		wantStages []string
	}{
		{
			name:       "execução do runner deve persistir métricas de execução e estágios",
			deliveryID: "DEL-020",
			taskCode:   "TASK-020",
			wantStages: []string{"doctor", "sensor", "validate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupBaseWorkspace(t)
			setupValidDelivery(t, dir, tt.deliveryID, tt.taskCode)

			opts := Options{
				Dir:        dir,
				DeliveryID: tt.deliveryID,
				NoGit:      true,
			}

			res, err := Run(opts)
			if err != nil {
				t.Fatalf("Run falhou inesperadamente: %v", err)
			}
			if res == nil || !res.OK {
				t.Fatalf("esperava execução com sucesso, obteve: %+v", res)
			}

			// Carrega as métricas geradas na raiz do workspace
			data, err := metrics.Load(dir)
			if err != nil {
				t.Fatalf("falha ao carregar métricas persistidas: %v", err)
			}
			if data == nil {
				t.Fatal("esperava métricas persistidas não-nulas após execução do runner")
			}

			// Deve conter ao menos um evento registrado
			if len(data.Events) == 0 {
				t.Error("esperava eventos registrados em metrics.Events após Run, obteve 0")
			}

			// Deve conter as métricas dos estágios executados
			recordedStages := make(map[string]bool)
			for _, st := range data.Stages {
				recordedStages[st.Stage] = true
			}

			for _, want := range tt.wantStages {
				if !recordedStages[want] {
					t.Errorf("estágio obrigatório %q não foi registrado em metrics.Stages", want)
				}
			}
		})
	}
}
