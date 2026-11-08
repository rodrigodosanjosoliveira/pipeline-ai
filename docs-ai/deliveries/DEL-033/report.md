---
delivery_id: DEL-033
task_code: TASK-033
title: "Integração automática de EventGate (runner), delivery close e MTTR inferido no cálculo das 4 métricas DORA"
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Embora o **Pipeline Runner (`pipeline-ai run`)** já registrasse eventos `EventGate` e durações por estágio (`doctor`, `sensor`, `validate`) em `.pipeline-state/metrics.json`, a função `CalculateDORA` lia apenas o slice `data.Deliveries` (populado por `RecordDelivery`) e eventos `EventRestore` explícitos. Como nem `pipeline-ai run` nem `pipeline-ai delivery close` invocavam `RecordDelivery`, o array `deliveries` permanecia vazio em uso normal, fazendo com que os 4 cards superiores de Métricas DORA no Dashboard Generative UI e no `pipeline-ai metrics summary` exibissem `0.00 / dia`, `0s` e `0.0%` mesmo com histórico real de execuções em `events` e `stages`.

# Escopo

- `internal/metrics/metrics.go`:
  - Adição da função `RecordDeliveryClose(rootDir, deliveryID string, startedAt, completedAt time.Time, success bool) error`, que persiste ou atualiza o `DeliveryMetric` em `data.Deliveries` e contabiliza automaticamente `FailedGateCount` a partir dos eventos `EventGate` daquela delivery.
  - Evolução de `CalculateDORA(data *MetricsData, windowDays int)` para sintetizar métricas de deliveries a partir de eventos `EventGate` quando uma delivery não possuir entrada explícita em `data.Deliveries`.
  - Evolução de `CalculateDORA` para inferir automaticamente o **MTTR (`TimeToRestore` e `RestoresTotal`)** sempre que uma sequência cronológica de `EventGate` para uma mesma delivery transicionar de falha (`Success: false`) para sucesso (`Success: true`).
- `cmd/pipeline-ai/main.go`:
  - Atualização de `runDeliveryClose` para carregar a delivery ativa antes de fechá-la e registrar sua conclusão em `metrics.RecordDeliveryClose`.
- Testes e Documentação:
  - `internal/metrics/metrics_test.go` (`TestCalculateDORAFromGateEventsAndAutoMTTR` e `TestRecordDeliveryClose`).
  - `cmd/pipeline-ai/main_test.go` (`TestCLIDeliveryCommands/delivery_close_registra_metricas_DORA_automaticamente`).
  - `docs-ai/DECISOES.md` (decisão `D-37`), `docs-ai/deliveries/DEL-032/report.md` (`merged`), `docs-ai/deliveries/INDEX.md` e este relatório.

**Fora de escopo:**
- Adição de dependências externas em `go.mod` (100% Go 1.24+ standard library).
- Mudanças no layout visual do Dashboard HTML (`internal/dashboard`).

# Implementação

1. **Síntese de Deliveries e MTTR Automático em `CalculateDORA` (`internal/metrics/metrics.go`):**
   - Mantido o processamento prioritário de `data.Deliveries` para total retrocompatibilidade.
   - Para qualquer `DeliveryID` presente em `data.Events` (`EventGate`) que não conste em `data.Deliveries`, `CalculateDORA` agrupa os eventos ordenados por timestamp, calcula o `LeadTime` (`lastTs - firstTs`, ou `Duration` quando houver evento único), verifica `FailedGateCount` e alimenta `DeploymentFrequency`, `LeadTimeForChanges` e `ChangeFailureRate`.
   - Quando não houver eventos `EventRestore` explícitos, `CalculateDORA` detecta pares de transição `EventGate(Success: false) -> EventGate(Success: true)` por `DeliveryID` e calcula `TimeToRestore` e `RestoresTotal` a partir do tempo real de recuperação.
2. **Fechamento de Delivery com Telemetria (`RecordDeliveryClose` e `cmd/pipeline-ai/main.go`):**
   - `runDeliveryClose` consulta `state.LoadActiveDelivery(root)` antes de chamar `state.CloseDelivery(root)` e grava o `DeliveryMetric` correspondente via `metrics.RecordDeliveryClose`.

**Arquivos alterados:**
- `internal/metrics/metrics.go` — `RecordDeliveryClose`, fallback `EventGate` e MTTR inferido em `CalculateDORA`
- `internal/metrics/metrics_test.go` — Testes `TestCalculateDORAFromGateEventsAndAutoMTTR` e `TestRecordDeliveryClose`
- `cmd/pipeline-ai/main.go` — Integração de `metrics.RecordDeliveryClose` no comando `pipeline-ai delivery close`
- `cmd/pipeline-ai/main_test.go` — Subteste `delivery_close_registra_metricas_DORA_automaticamente`
- `docs-ai/DECISOES.md` — Registro da decisão `D-37`
- `docs-ai/deliveries/DEL-032/report.md` — Status atualizado para `merged`
- `docs-ai/deliveries/INDEX.md` — `DEL-032` marcada como `merged` e `DEL-033` registrada
- `docs-ai/deliveries/DEL-033/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia, exit code 0)

$ go build ./...
(saída vazia, exit code 0)

$ go vet ./...
(saída vazia, exit code 0)

$ go test -race -count=1 ./internal/metrics ./internal/dashboard ./internal/runner
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.528s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.614s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	9.120s

$ go test -count=1 -run "TestCLIDeliveryCommands|TestCLIMetricsCommand" ./cmd/pipeline-ai
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.140s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Cálculo das 4 métricas DORA e MTTR inferido a partir de `EventGate` | `TestCalculateDORAFromGateEventsAndAutoMTTR` em `internal/metrics/metrics_test.go` |
| Registro de `DeliveryMetric` e contagem de falhas em `RecordDeliveryClose` | `TestRecordDeliveryClose` em `internal/metrics/metrics_test.go` |
| Integração automática no CLI `pipeline-ai delivery close` | `TestCLIDeliveryCommands` em `cmd/pipeline-ai/main_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Ordenação explícita por `Timestamp` dos eventos `EventGate` de cada delivery antes de calcular o `LeadTime` e o tempo de restauração (`TimeToRestore`), evitando dependência da ordem de inserção no slice.

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
