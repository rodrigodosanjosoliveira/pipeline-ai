---
delivery_id: DEL-038
task_code: TASK-038
title: "Agent Mode, detecção de TTY, hardening não-interativo e cobertura JSON integral na CLI"
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

CLIs modernas voltadas para uma audiência dupla (humanos e agentes autônomos de IA) enfrentam desafios críticos de confiabilidade e ergonomia. Quando operadas por agentes de IA, utilitários que demandam entrada interativa em `stdin` sem verificar se há um terminal conectado (TTY) travam a execução indefinidamente ou dependem de timeouts de subprocessos. Além disso, a ausência de saída estruturada em JSON em determinados subcomandos força agentes a realizar parsing frágil de strings em texto puro.

Seguindo os princípios de Dual-Audience CLI Design estabelecidos em D-39 e D-40, esta delivery implementa o **Agent Mode** global via flags (`-agent`, `-no-agent`) e variáveis de ambiente explícitas (`PIPELINE_AGENT_MODE=1`, `AI_AGENT=1`), detecção determinística de terminal via `isTTY` (`os.ModeCharDevice`), hardening não-interativo no subcomando `pipeline-ai init` (com flags `-non-interactive` e `-json`, aplicando defaults seguros sem travar em `stdin` ou falhando imediatamente com `hint` acionável se `-name` estiver ausente) e cobertura JSON integral em 100% dos subcomandos da CLI (`version`, `delivery status`, `module list`, `plugin verify`, `plugin install`, `sync`, `init`) (D-40).

# Escopo

- **Agent Mode Global e Detecção de TTY (`cmd/pipeline-ai/main.go`):**
  - Flags globais `-agent` / `--agent` (ativação) e `-no-agent` / `--no-agent` (desativação explícita).
  - Reconhecimento automático das variáveis de ambiente `PIPELINE_AGENT_MODE=1` e `AI_AGENT=1`.
  - Função `isTTY` baseada em `os.ModeCharDevice` para identificar se descritores de entrada/saída são terminais reais.
- **Hardening Não-Interativo do Scaffolding (`pipeline-ai init`):**
  - Suporte às flags `-non-interactive` e `-json`.
  - Falha determinística imediata (exit code 1) quando executado em ambiente não-interativo ou com `-json` sem a flag obrigatória `-name`, retornando erro e `"hint"` explicativo sem bloquear o processo.
  - Aplicação automática de defaults seguros (`project_type: desktop`, `db: PostgreSQL`) quando `-name` for informado em modo não-interativo/agent.
- **Cobertura JSON Integral em 100% dos Subcomandos:**
  - `version` (`-json`, emite `{"name":"pipeline-ai","version":"..."}`).
  - `delivery status` (`-json`, emite estado da delivery ativa ou status inativo em formato estruturado).
  - `module list` (`-json`, emite catálogo e status dos módulos opcionais).
  - `plugin verify` e `plugin install` (`-json`, emite resultado estruturado com `ok`, `dir` e mensagens).
  - `sync` (`-json`, emite contagem de skills sincronizadas e status).
  - `init` (`-json`, emite payload estruturado de inicialização ou erro com hint).
- **Testes Abrangentes (`cmd/pipeline-ai/main_test.go`):**
  - Testes unitários para `extractAgentMode` e `isTTY` (`TestExtractAgentModeAndIsTTY`).
  - Testes de integração de CLI para todos os subcomandos com `-json` e `-agent` (`TestCLIAgentModeAndJSONSubcommands`).
  - Testes de hardening para `init` não-interativo (`TestCLIInitNonInteractiveHardening`).
- **Documentação e Governança:**
  - `README.md`: Documentação de Agent Mode, flags não-interativas e inclusão da seção Roadmap (`DEL-039`).
  - `docs-ai/DECISOES.md`: Registro formal da decisão D-40.
  - `docs-ai/deliveries/INDEX.md`: Registro de DEL-038 e atualização de DEL-037 para `merged`.
  - `docs-ai/deliveries/DEL-037/report.md`: Sincronização do status para `merged`.
  - `docs-ai/deliveries/DEL-038/report.md`: Relatório de entrega da delivery.

**Fora de escopo:**
- Detecção de variáveis de ambiente genéricas de IDEs (como `TERM_PROGRAM=vscode`) para forçar JSON automaticamente.
- Adição de dependências externas ao `go.mod` (preservando 100% Go puro e stdlib).

# Implementação

A arquitetura de execução da CLI foi enriquecida em `cmd/pipeline-ai/main.go`:

1. **Extração de Agent Mode e Detecção de Terminal:**
   - A função `extractAgentMode(args)` extrai as flags globais `-agent` e `-no-agent`, consultando alternativamente `PIPELINE_AGENT_MODE` e `AI_AGENT`.
   - A função `isTTY(v any)` valida se um reader/stream é um dispositivo de caractere (`os.ModeCharDevice`). Quando `stdin` não é TTY, a CLI desativa requisições interativas.
2. **Hardening do Subcomando `init`:**
   - `runInit` avalia se está em modo não-interativo (`nonInteractive || agentMode || !isTTY(stdin)`). Se a flag `-name` estiver ausente, encerra com exit code 1 emitindo dica acionável no `stderr` ou no payload JSON (`"hint": "informe o nome do projeto via flag -name <nome> para execuções não-interativas"`). Havendo nome, aplica automaticamente os padrões do ecossistema.
3. **Cobertura Universal de Saída JSON:**
   - Implementada a flag `-json` nos subcomandos restantes: `version`, `delivery status`, `module list`, `plugin verify`, `plugin install`, `sync` e `init`. Sob Agent Mode ativo, o modo JSON é acionado automaticamente em todos os subcomandos.
4. **Governança e Roadmap:**
   - Registro da decisão arquitetural D-40 em `docs-ai/DECISOES.md`.
   - Seção de Roadmap em `README.md` detalhando a próxima delivery planejada `DEL-039` (Gemini 4 Hydrogen Routing & Native Semantic Provider).

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Agent Mode global, isTTY, hardening do init e flags `-json` universais
- `cmd/pipeline-ai/main_test.go` — Testes unitários e de integração de CLI para Agent Mode, detecção de TTY e JSON
- `README.md` — Atualização da documentação da CLI (Agent Mode, init não-interativo) e Roadmap (DEL-039)
- `docs-ai/DECISOES.md` — Registro formal da decisão D-40 (Seção 14)
- `docs-ai/deliveries/INDEX.md` — Registro da delivery DEL-038 e atualização da DEL-037
- `docs-ai/deliveries/DEL-037/report.md` — Sincronização do status para merged
- `docs-ai/deliveries/DEL-038/report.md` — Relatório de entrega desta delivery

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	2.052s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/debt	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.219s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	(cached)
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| `cmd/pipeline-ai/main.go` (`extractAgentMode` e `isTTY`) | `TestExtractAgentModeAndIsTTY` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (`version -json` e `-agent version`) | `TestCLIAgentModeAndJSONSubcommands` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (`delivery status -json`) | `TestCLIAgentModeAndJSONSubcommands` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (`module list -json` e `-no-agent`) | `TestCLIAgentModeAndJSONSubcommands` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (`plugin verify` e `plugin install` `-json`) | `TestCLIAgentModeAndJSONSubcommands` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (`sync -json`) | `TestCLIAgentModeAndJSONSubcommands` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (`init -non-interactive` e `-json` hardening) | `TestCLIInitNonInteractiveHardening` (`cmd/pipeline-ai/main_test.go`) |

**Validação (QA):**
Suíte completa de testes aprovada sem regressões em todos os 18 pacotes.

**Segurança:**
Sinal `SECURITY: nao`. Nenhuma alteração em criptografia, segredos ou controle de acesso.

**Supply Chain:**
Sinal `SUPPLY_CHAIN: nao`. Nenhuma nova dependência externa adicionada em `go.mod`.

**Pendências:**
Nenhuma pendência técnica identificada.

**Handoffs:**
Esteira executada continuamente através dos papéis `pm-triage` → `architecture-guardrails` → `test-writer` → `dev-implementation` → `code-reviewer` → `qa-validation` → `release-ops` → `docs-update`.

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** desativação de detecção genérica de IDEs como VS Code para preservar ergonomia humana no terminal integrado; inclusão de campo `hint` no erro JSON do `init`.

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
