---
delivery_id: DEL-022
task_code: TASK-022
title: Dashboard visual interativo de handoffs, esteira e métricas DORA (Generative UI e CLI)
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A observabilidade da esteira de múltiplos agentes exige uma interface visual rica, interativa e clara, tanto para desenvolvedores humanos quanto para a própria IA em sessões de pair programming. Com a consolidação das métricas DORA e dos contratos de handoff, faltava um meio determinístico de renderizar a visão unificada da esteira em formato visual moderno e responsivo sem dependência de frameworks externos pesados ou servidores daemon.

Esta delivery introduz o gerador de Dashboard Generative UI (`internal/dashboard`), o subcomando CLI `pipeline-ai dashboard` (com suporte a exportação em arquivo HTML e saída estruturada `-json`) e a ferramenta MCP `pipeline_get_dashboard_html` no servidor `internal/mcp`.

# Escopo

- Implementação do pacote `internal/dashboard`:
  - `CollectDashboardData`: agregação resiliente de métricas DORA (`metrics.GetDORA`), entregas registradas (`docs-ai/deliveries/INDEX.md`), configuração do projeto e status das 16 skills.
  - `RenderHTML`: renderização HTML5 self-contained, responsiva e moderna com Tailwind CSS oficial (`https://www.gstatic.com/antigravity/web/dev/tailwindcss.min.js`), em conformidade estrita com as diretrizes de Generative UI (sem CDNs não-autorizadas e tipografia com piso mínimo em 12px/text-xs).
  - Componentes visuais com abas interativas (Vanilla JS):
    - Métricas DORA (4 cards com velocímetros/indicadores).
    - Esteira visual das 16 skills (fluxo sequencial com status).
    - Tabela de histórico e status das entregas.
    - Pilares fundamentais de governança determinística.
- Subcomando CLI `pipeline-ai dashboard`:
  - Suporte a `-output <arquivo.html>` para salvar o dashboard em disco.
  - Suporte a `-json` para emissão do payload estruturado.
  - Saída padrão via `stdout` para pipelines e pipes Unix.
- Ferramenta MCP `pipeline_get_dashboard_html`:
  - Invocável via JSON-RPC 2.0 (`tools/call`) para gerar o HTML do dashboard dinamicamente.
- Alvo de Fuzzing (`FuzzRenderHTML`) garantindo robustez contra panics.

**Fora de escopo:**
- Servidores HTTP daemon de longa duração no terminal (o HTML gerado é autocontido e estático/reativo).
- Bibliotecas externas de UI/NPM no `go.mod`.
- Alterações em regras de negócio dos sensores ou hooks.

# Implementação

1. `internal/dashboard/dashboard.go`:
   - Estruturas de dados `DashboardData`, `DeliverySummary`, `PillarInfo`, `SkillInfo`.
   - Coletor tolerante `CollectDashboardData` com fallbacks para workspaces sem histórico prévio.
   - Template `html/template` compilado estaticamente com design moderno, abas interativas e estilização Tailwind permitida pela CSP do Antigravity.
2. `cmd/pipeline-ai/main.go`:
   - Subcomando `dashboard` tratando flags `-dir`, `-output` e `-json`.
   - Atualização da mensagem de ajuda (`printUsage`).
3. `internal/mcp/server.go`:
   - Tool `pipeline_get_dashboard_html` declarada no catálogo de ferramentas e despachada em `tools/call`.

**Arquivos alterados:**
- `internal/dashboard/dashboard.go` — implementação do agregador e renderizador HTML
- `internal/dashboard/dashboard_test.go` — testes unitários table-driven
- `internal/dashboard/dashboard_fuzz_test.go` — alvo de fuzzing da renderização
- `cmd/pipeline-ai/main.go` — subcomando CLI dashboard
- `cmd/pipeline-ai/main_test.go` — testes unitários da CLI
- `internal/mcp/server.go` — ferramenta MCP pipeline_get_dashboard_html
- `internal/mcp/server_test.go` — testes da ferramenta MCP
- `docs-ai/deliveries/INDEX.md` — registro da delivery DEL-022

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	3.372s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.105s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.566s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.222s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.028s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.070s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.006s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.202s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.074s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.027s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	5.767s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.918s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	3.124s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.106s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.529s
```

**Alvo de Fuzzing:**
```
$ go test -run=^FuzzRenderHTML$ -fuzz=FuzzRenderHTML -fuzztime=5s ./internal/dashboard
fuzz: elapsed: 0s, gathering baseline coverage: 51/51 completed, now fuzzing with 18 workers
fuzz: elapsed: 6s, execs: 19569 (0/sec), new interesting: 3 (total: 54)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	6.036s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Coleta de dados do dashboard em workspace completo | `TestCollectDashboardData` em `internal/dashboard/dashboard_test.go` |
| Coleta resiliente em workspace sem métricas ou sem INDEX.md | `TestCollectDashboardData` em `internal/dashboard/dashboard_test.go` |
| Renderização HTML com Tailwind e seções obrigatórias | `TestRenderHTML` em `internal/dashboard/dashboard_test.go` |
| Fuzzing da renderização contra panics | `FuzzRenderHTML` em `internal/dashboard/dashboard_fuzz_test.go` |
| Subcomando CLI pipeline-ai dashboard com stdout e -output | `TestCLIDashboard` em `cmd/pipeline-ai/main_test.go` |
| Tool MCP pipeline_get_dashboard_html | `TestMCPToolsGetDashboardHTML` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** adequação da tipografia para respeitar a diretriz de Generative UI do Antigravity (substituição de `text-[10px]` por `text-xs`).

**Sugestões recusadas:** nenhuma

# Rollback Plan

Esta entrega adiciona o pacote `internal/dashboard` sem alterar estruturas de dados ou modificar regras de segurança preexistentes. Em caso de reversão:
1. Reverter o commit com `git revert <commit_hash>`.
2. Recompilar o binário unificado via `go build -o /tmp/pipeline-ai ./cmd/pipeline-ai`.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
