---
delivery_id: DEL-021
task_code: TASK-021
title: "Release v0.7.0: Consolidação de telemetria, métricas DORA, pipeline runner e orquestração autônoma"
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Esta entrega formaliza o encerramento operacional do ciclo de inovação das entregas DEL-016 a DEL-020, consolidando os pilares de **Orquestração Autônoma Contínua**, **Pipeline Runner Determinístico**, **Cross-Compilação Transparente para Desktop** e **Telemetria com Métricas DORA para Agentes de IA**.

Ao longo das entregas deste ciclo, o framework superou o paradigma da microgestão manual de agentes e alcançou maturidade de produção:
- **DEL-016:** Princípio da Não-Interrupção (Regra 7) e orquestração contínua de ponta a ponta sem pausas protocolares.
- **DEL-017:** Pipeline Runner unificado (`pipeline-ai run`) executando a cascata determinística `Doctor` → `Sensor` → `Validate` e tool MCP nativa.
- **DEL-018:** Parser e validador automatizado de contratos de handoff (`pipeline-ai handoff validate`) blindado por Fuzzing.
- **DEL-019:** Detecção e injeção transparente de cross-compilação Windows Desktop Wails v3 sem CGO (`CGO_ENABLED=0 GOOS=windows GOARCH=amd64`).
- **DEL-020:** Motor de telemetria local em Go puro com cálculo das quatro métricas DORA (`pipeline-ai metrics summary|list`) e tool MCP `pipeline_get_metrics`.

Esta delivery oficializa o bump de versão para **v0.7.0**, documenta formalmente as decisões arquiteturais D-26 a D-30 em `docs-ai/DECISOES.md`, atualiza o `README.md` e consolida os registros no `INDEX.md`.

# Escopo

- Bump de versão para `v0.7.0`:
  - `cmd/pipeline-ai/main.go`: constante `version = "v0.7.0"`.
  - `cmd/pipeline-ai/main_test.go`: asserção de versão `"v0.7.0"`.
  - `internal/mcp/server.go`: handshake MCP `serverInfo.version = "v0.7.0"`.
  - `internal/mcp/server_test.go`: asserção de versão MCP `"v0.7.0"`.
  - `README.md`: indicação de versão estável `v0.7.0`.
- Formalização de Decisões Arquiteturais:
  - `docs-ai/DECISOES.md`: inclusão da seção 10 registrando formalmente as decisões D-26 a D-30.
- Governança de Deliveries:
  - `docs-ai/deliveries/INDEX.md`: atualização das entregas DEL-016 a DEL-020 para o status `merged` e registro de DEL-021.
  - `docs-ai/deliveries/DEL-020/report.md`: status sincronizado para `merged`.
- Criação e validação do relatório da release DEL-021.

**Fora de escopo:**
- Adição de dependências externas (100% Go 1.24+ standard library).
- Alterações em regras de negócio ou contratos de segurança existentes.

# Implementação

1. **Versionamento Central:**
   - Constante `version` em `cmd/pipeline-ai/main.go` atualizada para `"v0.7.0"`.
   - Handshake MCP `initialize` em `internal/mcp/server.go` atualizado para `"v0.7.0"`.
   - Testes unitários de versão (`main_test.go` e `server_test.go`) sincronizados.
2. **Decisões Arquiteturais:**
   - Formalização das decisões D-26 a D-30 em `docs-ai/DECISOES.md`.
3. **Documentação e Índice:**
   - `README.md` atualizado com a versão estável `v0.7.0`.
   - `docs-ai/deliveries/INDEX.md` sincronizado com status consolidado das entregas.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Bump de versão para v0.7.0
- `cmd/pipeline-ai/main_test.go` — Teste unitário de versão v0.7.0
- `internal/mcp/server.go` — Versão do handshake MCP v0.7.0
- `internal/mcp/server_test.go` — Teste de handshake MCP v0.7.0
- `README.md` — Referência de versão estável v0.7.0
- `docs-ai/DECISOES.md` — Formalização das decisões D-26 a D-30
- `docs-ai/deliveries/INDEX.md` — Atualização de status e registro de DEL-021
- `docs-ai/deliveries/DEL-020/report.md` — Status merged
- `docs-ai/deliveries/DEL-021/report.md` — Relatório de release v0.7.0

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	3.125s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.125s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.540s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.210s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.082s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.015s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.180s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.078s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.038s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	5.420s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.950s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	2.980s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.115s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.580s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Bump de versão CLI para v0.7.0 | `TestVersionCommand` em `cmd/pipeline-ai/main_test.go` |
| Bump de versão MCP para v0.7.0 | `TestServer_Initialize` em `internal/mcp/server_test.go` |
| Verificação de compilação de todos os pacotes | `go build ./...` |
| Suíte de regressão completa | `go test -count=1 ./...` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** consolidação do changelog e formalização das decisões D-26 a D-30 antes da criação da tag.

**Sugestões recusadas:** nenhuma

# Rollback Plan

Esta entrega consiste exclusivamente em bump de versão, documentação de decisões e metadados de release. Em caso de necessidade de reversão:
1. Reverter o commit com `git revert <commit_hash>`.
2. Remover a tag Git `v0.7.0` (caso criada) local e remotamente via `git tag -d v0.7.0 && git push --delete origin v0.7.0`.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
