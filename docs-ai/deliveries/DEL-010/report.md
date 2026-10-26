---
delivery_id: DEL-010
task_code: TASK-010
title: Modelo híbrido de autorização com gate por delivery ativa, TTL e caminhos imutáveis
status: draft
require_security: true
require_migration: false
require_performance: false
---

# Contexto

O contraditório técnico do framework identificou que a autorização de escrita estritamente atrelada à identidade do agente (`agent_type` ou `subagentId`) apresenta fragilidades práticas:
1. No ecossistema Antigravity, o runtime não transmite o nome nominal da skill no payload de chamada de ferramenta, mas sim um identificador efêmero de sessão, inviabilizando checagens estáticas confiáveis contra `editor_agents`.
2. Em sessões principais de desenvolvimento pair programming ou em IDEs distintas (Claude Code vs Antigravity vs Cursor), amarrar a permissão a um nome fixo de subagente gera atrito ou bloqueios indevidos quando o desenvolvedor conduz a tarefa dentro de um escopo homologado.
3. Não havia mecanismo declarativo para proibir incondicionalmente a escrita em caminhos hipercríticos (como regras de governança, manifests de CI ou configurações centrais), independentemente do agente atuando.

Esta entrega implementa a solução de autorização híbrida (C3): um gate acoplado ao ciclo de vida da delivery ativa com prazo de expiração (TTL de 7 dias) persistido em `.pipeline-state/active-delivery.json`, comandos CLI de gestão de entregas, integração completa ao servidor MCP e suporte a caminhos imutáveis (`immutable_paths`).

# Escopo

- Implementação do pacote `internal/state` com o ciclo de vida completo de `ActiveDelivery` (`Open`, `Close`, `Get`, `IsActive`) e expiração automática por TTL.
- Suporte a `immutable_paths` no arquivo `.pipeline-ai.json` e no pacote `internal/config/config.go` com o método `IsImmutable`.
- Integração do modelo híbrido de autorização no `internal/hook/hook.go`:
  1. Bloqueio incondicional de escrita em caminhos `immutable_paths`.
  2. Liberação por identidade de agente (`IsEditorAgent`).
  3. Liberação pela existência de delivery ativa válida e não expirada em `.pipeline-state`.
  4. Bloqueio determinístico se expirada (`ErrExpired`) ou ausente, com mensagens orientativas claras.
- Subcomandos de linha de comando: `pipeline-ai delivery open`, `pipeline-ai delivery close` e `pipeline-ai delivery status`.
- Integração no servidor MCP: abertura automática no `pipeline_scaffold_delivery`, registro da tool `pipeline_close_delivery` e atualização do resource `pipeline://delivery/active`.
- Inclusão de `.pipeline-state/` no `.gitignore`.

**Fora de escopo:**
- Modificações na ordem do fluxo técnico (C8: Security no PRE), a ser implementada na sequência.
- Esquema de relatórios customizáveis (C10/C11).
- Dependências externas de terceiros (100% Go stdlib pura).

# Implementação

1. **`internal/state/state.go`:**
   - Pacote dedicado que gerencia o arquivo `.pipeline-state/active-delivery.json`.
   - Estrutura `ActiveDelivery` contendo `delivery_id`, `task_code`, `opened_at` e `expires_at`.
   - Função `Open` com validação de parâmetros e cálculo de expiração UTC baseado em TTL.
   - Função `IsActive` que valida a presença e integridade do arquivo, retornando `ErrExpired` caso `time.Now().UTC()` ultrapasse `ExpiresAt`.
   - Função `Close` idempotente que remove o arquivo de estado.
2. **`internal/config/config.go`:**
   - Campo `ImmutablePaths []string \`json:"immutable_paths"\`` mapeado em `Config`.
   - Método `IsImmutable(absPath string) bool` que resolve o caminho canônico com `relToRoot` e avalia padrões contra a lista de imutáveis.
3. **`internal/hook/hook.go`:**
   - Em `Run()`, após verificar que a ferramenta é de edição e o caminho é protegido:
     - Bloqueio imediato caso `cfg.IsImmutable(path)` seja verdadeiro.
     - Liberação caso `cfg.IsEditorAgent(in.agentType())` seja verdadeiro.
     - Liberação caso `state.IsActive(cfg.Root())` confirme delivery ativa válida.
     - Bloqueio com diagnóstico de expiração caso `state.ErrExpired` seja detectado.
4. **`cmd/pipeline-ai/main.go`:**
   - Subcomando `delivery` registrado no menu de ajuda e roteado para `runDeliveryOpen`, `runDeliveryClose` e `runDeliveryStatus`.
5. **`internal/mcp/scaffold.go` e `internal/mcp/server.go`:**
   - `ScaffoldDelivery` dispara `state.Open` automaticamente na raiz do projeto ao criar o esqueleto da entrega.
   - Tool `pipeline_close_delivery` registrada e implementada para encerramento de entregas via MCP.
   - Resource `pipeline://delivery/active` atualizado para retornar status e metadados da delivery em JSON.
6. **`.gitignore`:**
   - Adicionado `.pipeline-state/` para isolar dados locais de sessão.

**Arquivos alterados:**
- `.gitignore` — Isolamento de `.pipeline-state/`
- `cmd/pipeline-ai/main.go` — Adição dos subcomandos delivery open, close e status
- `cmd/pipeline-ai/main_test.go` — Testes dos comandos CLI de delivery
- `cmd/pipeline-hook/main_test.go` — Testes de autorização híbrida, expiração de TTL e caminhos imutáveis
- `internal/config/config.go` — Suporte a immutable_paths e método IsImmutable
- `internal/config/config_test.go` — Testes unitários de IsImmutable
- `internal/hook/hook.go` — Orquestração de salvaguardas e modelo híbrido no hook
- `internal/mcp/scaffold.go` — Abertura automática de delivery ativa no scaffold
- `internal/mcp/server.go` — Tool pipeline_close_delivery e resource pipeline://delivery/active
- `internal/mcp/server_test.go` — Testes de MCP para scaffold, close e resource de delivery
- `internal/state/state.go` — Pacote de estado da delivery ativa com TTL
- `internal/state/state_test.go` — Testes unitários e alvo de fuzzing do pacote state
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-010
- `docs-ai/deliveries/DEL-010/report.md` — Relatório de entrega da delivery

# Testes

**Gates:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso)

$ go vet ./...
(análise estática sem avisos)

$ go test ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.195s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.121s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.111s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.134s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.059s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.147s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.016s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.523s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.113s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.348s

$ go test -fuzz=FuzzActiveDeliveryJSON -fuzztime=5s ./internal/state
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	6.096s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Ciclo de vida da delivery ativa | `TestOpenAndGet`, `TestIsActive`, `TestClose` em `internal/state/state_test.go` |
| Resiliência a JSON corrompido | `FuzzActiveDeliveryJSON` em `internal/state/state_test.go` |
| Suporte a `immutable_paths` | `TestIsImmutable` e `TestLoadImmutablePaths` em `internal/config/config_test.go` |
| Gate híbrido e expiração de TTL | `TestHybridGateActiveDelivery` em `cmd/pipeline-hook/main_test.go` |
| Bloqueio incondicional de imutáveis | `TestImmutablePathsHookBlocking` em `cmd/pipeline-hook/main_test.go` |
| MCP Tools e Resource de delivery | `TestMCPScaffoldOpensActiveDelivery`, `TestMCPCloseDelivery`, `TestMCPResourceDeliveryActive` em `internal/mcp/server_test.go` |
| CLI `pipeline-ai delivery` | `TestCLIDeliveryCommands` em `cmd/pipeline-ai/main_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Precedência rigorosa de segurança no hook avaliando `IsImmutable` antes de qualquer verificação de autorização de agente ou delivery.

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- `internal/state/state.go` — Gerenciamento seguro de estado local com sanitização e fuzzing
- `internal/config/config.go` — Proteção perimetral inviolável via `immutable_paths`
- `internal/hook/hook.go` — Modelo de autorização híbrido, expiração automática por TTL e fail-closed
- `.gitignore` — Isolamento de runtime local prevenindo exposição de estado em commits

**Achados:**
- Nenhum achado de vulnerabilidade residual (0 CRITICO, 0 ALTO, 0 MEDIO, 0 BAIXO).
- Zero dependências externas adicionadas ao projeto.
- Zero credenciais, chaves ou segredos em código.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
