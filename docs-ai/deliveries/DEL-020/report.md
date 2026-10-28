---
delivery_id: DEL-020
task_code: TASK-020
title: Métricas DORA e telemetria de agentes de IA no runner e servidor MCP
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A governança do pipeline com múltiplos agentes exige visibilidade sobre eficiência, estabilidade e fluxo de entrega contínua. Sem medição quantitativa, é inviável identificar gargalos em estágios do pipeline (como tempo gasto em testes versus implementação) ou rastrear a frequência e confiabilidade de entregas autônomas.

Esta delivery introduz um motor nativo de telemetria e o cálculo determinístico das quatro métricas DORA (Deployment Frequency, Lead Time for Changes, Change Failure Rate e Time to Restore Service / MTTR) adaptadas para agentes autônomos, instrumentando o runner e expondo os resultados via CLI (`pipeline-ai metrics`) e servidor MCP (`pipeline_get_metrics`).

# Escopo

- Implementação do pacote `internal/metrics` em Go 1.24 puro (stdlib) para registro de eventos e estágios.
- Cálculo determinístico das quatro métricas-chave DORA:
  - Deployment Frequency (taxa diária e total de deploys).
  - Lead Time for Changes (média e mediana de duração das entregas).
  - Change Failure Rate (percentual de falhas em gates e pipelines).
  - Time to Restore Service / MTTR (tempo médio de recuperação entre falhas e restaurações).
- Persistência e carregamento com gravação atômica e tolerância fail-safe a arquivos inexistentes, vazios ou corrompidos.
- Instrumentação do Pipeline Runner (`internal/runner`) com medição de durações de cada estágio (`doctor`, `sensor`, `validate`) e evento global de gate.
- Criação do subcomando CLI `pipeline-ai metrics [summary|list]` com suporte a `-json` e janelas temporais configuráveis.
- Adição da ferramenta MCP `pipeline_get_metrics` no servidor `internal/mcp`.
- Alvo de Fuzzing contínuo (`FuzzParseMetrics`) para proteção de parsing contra panics.

**Fora de escopo:**
- Exportação de telemetria para servidores SaaS externos (manter privacidade zero-trust local).
- Dependências externas no `go.mod`.
- Alteração nos contratos existentes de hooks ou regras de handoff.

# Implementação

A implementação foi estruturada no novo pacote `internal/metrics`:
1. `internal/metrics/metrics.go`:
   - Estruturas de dados `MetricsData`, `Event`, `StageDuration`, `DeliveryMetric` e `DORAMetrics`.
   - Persistência atômica via arquivo temporário com rename seguro em `.pipeline-state/metrics.json`.
   - Motor matemático de métricas DORA com suporte a janelas temporais (`Window`, `Since`, `Until`).
2. `internal/runner/runner.go`:
   - Instrumentação transparente com `metrics.RecordStageDuration` nos estágios `doctor`, `sensor` e `validate`.
   - Gravação de evento global `EventGate` com status de aprovação geral.
3. `cmd/pipeline-ai/main.go`:
   - Subcomando `metrics` tratando os subcomandos `summary` e `list`, com saída formatada para humanos e flag `-json`.
4. `internal/mcp/server.go`:
   - Nova tool MCP `pipeline_get_metrics` declarada em `tools/list` e despachada em `tools/call`.

**Arquivos alterados:**
- `internal/metrics/metrics.go` — implementação do motor de telemetria e métricas DORA
- `internal/metrics/metrics_test.go` — testes unitários e de tolerância
- `internal/metrics/metrics_fuzz_test.go` — alvo de fuzzing contra payloads malformados
- `internal/runner/runner.go` — instrumentação do runner para registro de estágios e eventos
- `internal/runner/runner_metrics_test.go` — testes de integração entre runner e métricas
- `cmd/pipeline-ai/main.go` — comando CLI metrics summary e metrics list
- `cmd/pipeline-ai/main_test.go` — testes unitários do comando metrics na CLI
- `internal/mcp/server.go` — tool MCP pipeline_get_metrics
- `internal/mcp/server_test.go` — testes da tool MCP pipeline_get_metrics
- `docs-ai/deliveries/INDEX.md` — registro da delivery DEL-020

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	3.332s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.131s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.579s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.224s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.079s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.012s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.207s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.075s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.037s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	5.625s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.941s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	3.104s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.119s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.560s
```

**Alvo de Fuzzing:**
```
$ go test -run=^FuzzParseMetrics$ -fuzz=FuzzParseMetrics -fuzztime=10s ./internal/metrics
fuzz: elapsed: 0s, gathering baseline coverage: 367/367 completed, now fuzzing with 18 workers
fuzz: elapsed: 11s, execs: 1132783 (48838/sec), new interesting: 91 (total: 458)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	11.068s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Persistência e tolerância fail-safe | `TestLoad_Tolerance` em `internal/metrics/metrics_test.go` |
| Salvamento e carregamento atômico | `TestSaveAndLoad` em `internal/metrics/metrics_test.go` |
| Registro de eventos e estágios | `TestRecordEvent` e `TestRecordStageDuration` em `internal/metrics/metrics_test.go` |
| Cálculo matemático das 4 métricas DORA | `TestCalculateDORA` em `internal/metrics/metrics_test.go` |
| Fuzzing contra entradas corrompidas | `FuzzParseMetrics` em `internal/metrics/metrics_fuzz_test.go` |
| Instrumentação do Pipeline Runner | `TestRunner_RecordsMetrics` em `internal/runner/runner_metrics_test.go` |
| Subcomando CLI pipeline-ai metrics | `TestMetricsCommand` em `cmd/pipeline-ai/main_test.go` |
| Tool MCP pipeline_get_metrics | `TestServer_GetMetricsTool` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** implementação fail-safe em `ParseMetrics` retornando estrutura vazia inicializada em vez de erro para payloads vazios, permitindo bootstrap suave em repositórios novos.

**Sugestões recusadas:** nenhuma

# Rollback Plan

Esta entrega adiciona o pacote `internal/metrics` sem alterar contratos de dados destrutivos ou introduzir migrações de banco. Em caso de necessidade de reversão:
1. Reverter os commits com `git revert -m 1 <commit_hash>`.
2. Remover o diretório `.pipeline-state/metrics.json` (se presente).
3. Recompilar o binário unificado via `go build -o /tmp/pipeline-ai ./cmd/pipeline-ai`.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
