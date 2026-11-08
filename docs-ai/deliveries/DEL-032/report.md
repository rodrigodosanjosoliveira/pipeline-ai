---
delivery_id: DEL-032
task_code: TASK-032
title: "Painel de telemetria semântica Jev (SemanticSummary) no Dashboard Generative UI e CLI metrics"
status: merged
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Após a consolidação das três primitivas do sensor semântico calibrado Jev (`EvalNoul`, `EvalChoice`, `EvalScore`) e da coleta de eventos `SemanticEvalEvent` na `DEL-029` / `DEL-030` (`v0.10.0`), faltava expor visualmente a telemetria semântica agregada (`SemanticSummary`) no cockpit HTML interativo (`internal/dashboard` — Generative UI) e unificar o contrato `DORAMetrics.Semantic` com o comando `pipeline-ai metrics summary` (tanto em modo texto quanto em `-json`).

# Escopo

- `internal/metrics/metrics.go`:
  - Adição da struct `SemanticSummary` (`TotalEvals`, `ApprovedRate`, `AvgConfidence`, `ByMode`) e do campo `Semantic SemanticSummary` (`json:"semantic"`) em `DORAMetrics`.
  - Cálculo de `SemanticSummary` dentro de `CalculateDORA(data *MetricsData, windowDays int)` a partir de `data.Events` (`EventSemanticEval`) e `data.SemanticEvals`.
- `internal/dashboard/dashboard.go`:
  - Adição dos helpers de template `mul100` e `modeCount`.
  - Adição da seção **"Sensor Semântico Calibrado (System 1 / Jev)"** na Aba 1 (`tab-dora`) com 4 cards responsivos: Total de Avaliações, Taxa de Aprovação, Confiança Média e Distribuição por Modo (`noul`, `choice`, `score`).
- `cmd/pipeline-ai/main.go`:
  - Atualização de `runMetrics` (`pipeline-ai metrics summary`) para exibir o bloco `Sensor Semântico (Jev):` quando `dora.Semantic.TotalEvals > 0` e serializar `"semantic"` no modo `-json`.
- Testes e Documentação:
  - `internal/metrics/metrics_test.go` (`TestCalculateDORASemanticSummary`).
  - `internal/dashboard/dashboard_test.go` (`TestCollectDashboardData` e `TestRenderHTML`).
  - `cmd/pipeline-ai/main_test.go` (`TestCLIMetricsCommand`).
  - `docs-ai/DECISOES.md` (decisão `D-36`), `docs-ai/deliveries/DEL-031/report.md` (`merged`), `docs-ai/deliveries/INDEX.md` e este relatório.

**Fora de escopo:**
- Adição de dependências externas em `go.mod` (100% Go 1.24+ standard library).
- Alteração no cliente HTTP do sensor semântico (`internal/sensor/semantic`).

# Implementação

1. **Agregação Unificada em `internal/metrics/metrics.go`:**
   - Adicionada `SemanticSummary` em `DORAMetrics.Semantic` com inicialização garantida de `ByMode: make(map[string]int)` para evitar mapa `nil` no JSON.
   - `CalculateDORA` consolida eventos `EventSemanticEval` e registros `SemanticEvals` filtrados pela janela temporal (`cutoff`), computando `TotalEvals`, `ApprovedRate`, `AvgConfidence` e contagem por modo (`noul`, `choice`, `score`).
2. **Painel Visual no Dashboard Generative UI (`internal/dashboard/dashboard.go`):**
   - Registradas as funções `mul100` e `modeCount` no `template.FuncMap`.
   - Incluída na Aba 1 (`tab-dora`) a seção `"Sensor Semântico Calibrado (System 1 / Jev)"` com 4 cards seguindo o mesmo grid CSS responsivo das métricas DORA e escape nativo de `html/template`.
3. **CLI `pipeline-ai metrics summary` (`cmd/pipeline-ai/main.go`):**
   - Exibição formatada das estatísticas do sensor semântico no terminal quando `dora.Semantic.TotalEvals > 0`.

**Arquivos alterados:**
- `internal/metrics/metrics.go` — `SemanticSummary`, `DORAMetrics.Semantic` e agregação em `CalculateDORA`
- `internal/metrics/metrics_test.go` — Teste unitário `TestCalculateDORASemanticSummary`
- `internal/dashboard/dashboard.go` — Seção visual do sensor semântico Jev na Aba 1 e helpers `mul100`/`modeCount`
- `internal/dashboard/dashboard_test.go` — Asserções de `DORA.Semantic` e renderização HTML dos cards semânticos
- `cmd/pipeline-ai/main.go` — Exibição de `dora.Semantic` em `pipeline-ai metrics summary`
- `cmd/pipeline-ai/main_test.go` — Asserções de telemetria semântica em `TestCLIMetricsCommand`
- `docs-ai/DECISOES.md` — Registro da decisão `D-36`
- `docs-ai/deliveries/DEL-031/report.md` — Status atualizado para `merged`
- `docs-ai/deliveries/INDEX.md` — `DEL-031` marcada como `merged` e `DEL-032` registrada
- `docs-ai/deliveries/DEL-032/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia, exit code 0)

$ go build ./...
(saída vazia, exit code 0)

$ go vet ./...
(saída vazia, exit code 0)

$ go test -race -count=1 ./internal/metrics ./internal/dashboard
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.542s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.618s

$ go test -count=1 -run TestCLIMetricsCommand ./cmd/pipeline-ai
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.812s

$ go test -run="^$" -fuzz=FuzzParseMetrics -fuzztime=3s ./internal/metrics
fuzz: elapsed: 0s, gathering baseline coverage: 0/13 completed
fuzz: elapsed: 0s, gathering baseline coverage: 13/13 completed, now fuzzing with 10 workers
fuzz: elapsed: 3s, execs: 923177 (307692/sec), new interesting: 0 (total: 13)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	3.195s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Cálculo de `SemanticSummary` em `CalculateDORA` | `TestCalculateDORASemanticSummary` e `TestRecordSemanticEval` em `internal/metrics/metrics_test.go` |
| Coleta e renderização dos 4 cards do sensor semântico no Dashboard HTML | `TestCollectDashboardData` e `TestRenderHTML` em `internal/dashboard/dashboard_test.go` |
| Exibição de `Sensor Semântico (Jev)` na CLI `metrics summary` e `-json` | `TestCLIMetricsCommand` em `cmd/pipeline-ai/main_test.go` |
| Resiliência do parser de métricas com Fuzzing | `FuzzParseMetrics` em `internal/metrics/metrics_fuzz_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Uso do helper `modeCount` no `template.FuncMap` do dashboard para leitura segura das chaves `"noul"`, `"choice"` e `"score"` no mapa `ByMode`.

**Sugestões recusadas:** nenhuma

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
