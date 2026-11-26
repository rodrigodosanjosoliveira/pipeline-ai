---
delivery_id: DEL-042
task_code: TASK-042
title: "Release v0.12.0: Consolidação do Dual-Audience CLI Design, Gemini 4 Hydrogen e Helpful Errors"
status: merged
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Esta entrega formaliza o lançamento da versão estável **`v0.12.0`** do framework **Pipeline-AI**, consolidando todas as entregas desde a `v0.11.0` (`DEL-036` a `DEL-042`):
1. **Ajustes no 5º Artigo Técnico (`DEL-036`):** Ajustes de formatação e legibilidade no pacote do 5º artigo técnico (`post-feed-linkedin-yagni.txt`).
2. **Bounded Output, Smart Test Filtering & Partial Results (`DEL-037`, D-39):** Teto padrão de 8 KB por sensor (`-max-output-bytes`, `-verbose-output`), supressão de blocos `--- PASS`/`--- SKIP` em testes aprovados, preservação integral de `--- FAIL` e metadados explícitos (`truncated`, `total_bytes`, `omitted_bytes`, `failed_tests`) na CLI e MCP (`pipeline_run_sensors`, `pipeline_run_pipeline`).
3. **Agent Mode, Detecção de TTY & Cobertura JSON Integral (`DEL-038`, D-40):** Flags globais `-agent` / `-no-agent` e variáveis `PIPELINE_AGENT_MODE` / `AI_AGENT`, detecção de terminal via `isTTY` (`os.ModeCharDevice`), hardening não-interativo no `init` (`-non-interactive`, `-json`) e saída JSON estruturada em 100% dos subcomandos (`version`, `delivery status`, `module list`, `plugin verify`, `plugin install`, `sync`, `init`).
4. **Gemini 4 Hydrogen Routing & Native Semantic Provider (`DEL-039`, D-41):** Matriz `model_routing` em `.pipeline-ai.json` (`DefaultModelRouting` e `ResolveSkillModel`) exposta em `pipeline://catalog`, e provider nativo `"gemini"` em `internal/sensor/semantic`, `pipeline-ai eval` e `pipeline_eval_semantic` usando `generateContent`, `responseSchema`, `thinkingConfig.thinkingLevel` e header `x-goog-api-key` em 100% Go stdlib.
5. **Discovery & Schema Introspection (`DEL-040`, D-42):** Subcomando `pipeline-ai schema [comando]` e flag global `-schema` / `--schema` com catálogo declarativo (`cmd/pipeline-ai/schema.go`) de flags globais, variáveis de ambiente e todos os 17 comandos sem efeitos colaterais.
6. **Helpful Errors & Guided Recovery (`DEL-041`, D-43):** Sugestão automática de comandos por distância de Levenshtein (`suggestCommand`), respostas estruturadas (`cause`, `hint`, `fix_command`) na CLI (`writeCLIError`) e no servidor MCP (`sendToolError`).

# Escopo

- Bump de versão sincronizado para `v0.12.0`:
  - `cmd/pipeline-ai/main.go`: constante `version = "v0.12.0"`.
  - `cmd/pipeline-ai/main_test.go`: asserção de versão `"v0.12.0"` em `TestCLIRouting`.
  - `internal/mcp/server.go`: handshake MCP `serverInfo.version = "v0.12.0"`.
  - `internal/mcp/server_test.go`: asserção de versão MCP `"v0.12.0"` em `TestMCPInitializeAndPing`.
  - `plugin/plugin.json`: campo `"version": "0.12.0"`.
  - `internal/plugin/plugin_test.go`: asserção `"0.12.0"` em `TestOfficialPluginBundleVersion`.
- Documentação e Governança:
  - `CHANGELOG.md`: seção `## [0.12.0] - 2026-11-26` consolidando `DEL-036`–`DEL-041`.
  - `README.md`: atualização de referências de versão `v0.11.0` para `v0.12.0`.
  - `docs-ai/deliveries/DEL-041/report.md`: sincronização de `status: draft` para `status: merged`.
  - `docs-ai/deliveries/INDEX.md`: atualização de `DEL-041` para `merged` e registro de `DEL-042`.
  - `docs-ai/deliveries/DEL-042/report.md`: relatório completo desta delivery.

**Fora de escopo:**
- Adição de dependências externas em `go.mod` (100% Go 1.24+ standard library mantido).
- Alteração de lógica funcional nos pacotes de domínio além do bump de versão e documentação da release.

# Implementação

1. **Versionamento Sincronizado (`v0.12.0`):**
   - Atualizada a versão da CLI (`cmd/pipeline-ai/main.go`), do servidor MCP (`internal/mcp/server.go`) e do manifesto do plugin oficial (`plugin/plugin.json`) para `v0.12.0` / `0.12.0`, com atualização correspondente nos testes unitários.
2. **Documentação de Release (`CHANGELOG.md` e `README.md`):**
   - Consolidado o histórico da `v0.12.0` no `CHANGELOG.md` cobrindo `DEL-036` a `DEL-041`.
   - Atualizadas as referências de versão no `README.md` para `v0.12.0`.
3. **Sincronização de Governança (`docs-ai/deliveries/`):**
   - Atualizado o status de `DEL-041` para `merged` em `docs-ai/deliveries/DEL-041/report.md` e `docs-ai/deliveries/INDEX.md`, e registrada a `DEL-042` em `docs-ai/deliveries/INDEX.md`.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Bump de versão para `v0.12.0`
- `cmd/pipeline-ai/main_test.go` — Asserção de versão `v0.12.0` em `TestCLIRouting`
- `internal/mcp/server.go` — Versão do handshake MCP `v0.12.0`
- `internal/mcp/server_test.go` — Asserção de handshake MCP `v0.12.0` em `TestMCPInitializeAndPing`
- `internal/plugin/plugin_test.go` — Asserção de versão `0.12.0` em `TestOfficialPluginBundleVersion`
- `plugin/plugin.json` — Versão do manifesto `0.12.0`
- `README.md` — Referências de versão atualizadas para `v0.12.0`
- `CHANGELOG.md` — Notas de release da versão `0.12.0`
- `docs-ai/deliveries/INDEX.md` — Registro de `DEL-042` e atualização de `DEL-041` para `merged`
- `docs-ai/deliveries/DEL-041/report.md` — Atualização de `status: draft` para `status: merged`
- `docs-ai/deliveries/DEL-042/report.md` — Relatório de entrega desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia, exit code 0)

$ go build ./...
(saída vazia, exit code 0)

$ go vet ./...
(saída vazia, exit code 0)

$ go test -v -run "TestCLIRouting" ./cmd/pipeline-ai
=== RUN   TestCLIRouting
=== RUN   TestCLIRouting/ajuda_sem_argumentos
=== RUN   TestCLIRouting/comando_help
=== RUN   TestCLIRouting/comando_version
=== RUN   TestCLIRouting/comando_desconhecido
--- PASS: TestCLIRouting (0.00s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.261s

$ go test -v -run "TestMCPInitializeAndPing" ./internal/mcp
=== RUN   TestMCPInitializeAndPing
--- PASS: TestMCPInitializeAndPing (0.00s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.272s

$ go test -v -run "TestOfficialPluginBundleVersion" ./internal/plugin
=== RUN   TestOfficialPluginBundleVersion
--- PASS: TestOfficialPluginBundleVersion (0.00s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.209s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Bump de versão CLI para `v0.12.0` | `TestCLIRouting` em `cmd/pipeline-ai/main_test.go` |
| Bump de versão MCP para `v0.12.0` | `TestMCPInitializeAndPing` em `internal/mcp/server_test.go` |
| Validação de versão `0.12.0` no bundle oficial do plugin | `TestOfficialPluginBundleVersion` em `internal/plugin/plugin_test.go` |
| Validação estrutural do relatório da release | `go run ./cmd/pipeline-validate docs-ai/deliveries/DEL-042/report.md` |

**Validação (QA):**
`gofmt -l .`, `go build ./...`, `go vet ./...`, `TestCLIRouting`, `TestMCPInitializeAndPing` e `TestOfficialPluginBundleVersion` aprovados.

**Segurança:**
Sinal `SECURITY: nao`. Nenhuma alteração em criptografia, segredos ou controle de acesso.

**Supply Chain:**
Sinal `SUPPLY_CHAIN: nao`. Nenhuma nova dependência externa adicionada em `go.mod`.

**Pendências:**
Nenhuma pendência técnica identificada.

**Handoffs:**
Esteira executada continuamente através dos papéis `pm-triage` → `architecture-guardrails` → `test-writer` → `dev-implementation` → `code-reviewer` → `qa-validation` → `docs-update`.

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** sincronização integral de versões (`v0.12.0` / `0.12.0`) entre CLI, servidor MCP, plugin manifesto e documentação.

**Sugestões recusadas:** nenhuma

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde (validado nos alvos de versão `TestCLIRouting`, `TestMCPInitializeAndPing` e `TestOfficialPluginBundleVersion`)
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
