---
delivery_id: DEL-030
task_code: TASK-030
title: "Release v0.10.0: Consolidação dos modos semânticos Jev (choice e score), telemetria e artigo técnico"
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Esta entrega formaliza o encerramento operacional do ciclo **v0.10.0** do framework Pipeline-AI, consolidando a expansão das primitivas de sensoriamento semântico e decisões calibradas com modelos "System One" (Jev) e a publicação do 4º artigo técnico oficial da série:
1. **Resolução Recursiva de Symlinks Ancestrais no macOS (DEL-028):** Correção de `Config.relToRoot` via `resolveSymlinkAncestors` para suportar múltiplos níveis de subdiretórios ainda inexistentes sob árvores symlinkadas (`/var -> /private/var`).
2. **Expansão Completa do Sensor Semântico Jev e Telemetria (DEL-029):** Implementação de `EvalChoice` (`/v1/choice` com trava anti-alucinação `SEC-01` / `ErrInvalidChoice`) e `EvalScore` (`/v1/score` com dupla trava de calibração e fronteiras `SEC-02` / `ErrInvalidScore`), flags `-mode noul|choice|score`, `-choices` e `-max-score` na CLI `pipeline-ai eval`, parâmetros equivalentes na tool MCP `pipeline_eval_semantic`, Fuzz testing dedicado (`FuzzParseChoiceResponse` e `FuzzParseScoreResponse`) e telemetria semântica (`SemanticEvalEvent` e `SemanticSummary`) acoplada ao motor de métricas DORA.
3. **Pacote Editorial Completo do 4º Artigo Técnico (`docs-ai/artigos/`):** Elaboração do artigo *"System 1 vs. System 2 em Engenharia de IA: Como Substituímos Julgamentos Ambíguos de LLMs por Sensores Semânticos Calibrados (Jev) em Go Puro"* em Markdown (PT-BR), HTML estilizado (LinkedIn Articles / Substack), Inglês (Dev.to) e texto de divulgação para feed do LinkedIn.

# Escopo

- Bump de versão sincronizado para `v0.10.0`:
  - `cmd/pipeline-ai/main.go`: constante `version = "v0.10.0"`.
  - `cmd/pipeline-ai/main_test.go`: asserção de versão `"v0.10.0"` em `TestCLIRouting`.
  - `internal/mcp/server.go`: handshake MCP `serverInfo.version = "v0.10.0"`.
  - `internal/mcp/server_test.go`: asserção de versão MCP `"v0.10.0"` em `TestMCPInitializeAndPing`.
  - `plugin/plugin.json`: campo `"version": "0.10.0"`.
  - `internal/plugin/plugin_test.go`: asserção `"0.10.0"` em `TestOfficialPluginBundleVersion`.
  - `README.md`: indicação de versão estável `v0.10.0`.
- Publicação do 4º Artigo Técnico sobre o Jev (`docs-ai/artigos/`):
  - `docs-ai/artigos/artigo-sensoriamento-semantico-jev.md`
  - `docs-ai/artigos/artigo-sensoriamento-semantico-jev.html`
  - `docs-ai/artigos/article-semantic-sensing-jev-go-devto.md`
  - `docs-ai/artigos/post-feed-linkedin-jev.txt`
  - `docs-ai/artigos/capa-artigo-sensoriamento-semantico-jev.jpg`
- Governança Arquitetural e de Deliveries:
  - `docs-ai/DECISOES.md`: atualização da seção 12 (`Consolidação Arquitetural da v0.9.0 e v0.10.0`).
  - `docs-ai/deliveries/DEL-029/report.md`: status sincronizado para `merged`.
  - `docs-ai/deliveries/INDEX.md`: atualização de `DEL-029` para `merged` e registro de `DEL-030` como `draft`.
  - `docs-ai/deliveries/DEL-030/report.md`: criação e validação do relatório da release `v0.10.0`.

**Fora de escopo:**
- Adição de dependências externas em `go.mod` (100% Go 1.24+ standard library mantido).
- Alteração de lógica funcional nos pacotes de domínio além do bump de versão.

# Implementação

1. **Versionamento Central e Sincronizado (`v0.10.0`):**
   - Constante `version` em `cmd/pipeline-ai/main.go` atualizada para `"v0.10.0"`.
   - Handshake MCP `initialize` em `internal/mcp/server.go` atualizado para `"v0.10.0"`.
   - Manifesto oficial `plugin/plugin.json` atualizado para `"0.10.0"`.
   - Testes unitários (`cmd/pipeline-ai/main_test.go`, `internal/mcp/server_test.go` e `internal/plugin/plugin_test.go`) sincronizados para `v0.10.0` / `0.10.0`.
2. **Pacote do 4º Artigo Técnico (`docs-ai/artigos/`):**
   - Criados os 4 artefatos de publicação detalhando a arquitetura System 1 vs. System 2 com Jev em Go puro (`EvalNoul`, `EvalChoice`, `EvalScore`), hardening `SEC-01`/`SEC-02`/`SEC-03`, Fuzz Testing e telemetria DORA.
3. **Documentação e Governança:**
   - `README.md` e `docs-ai/DECISOES.md` atualizados para `v0.10.0`.
   - `docs-ai/deliveries/DEL-029/report.md` atualizado para `status: merged`.
   - `docs-ai/deliveries/INDEX.md` atualizado com `DEL-029` como `merged` e `DEL-030` registrada no topo.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Bump de versão para `v0.10.0`
- `cmd/pipeline-ai/main_test.go` — Asserção de versão `v0.10.0` em `TestCLIRouting`
- `internal/mcp/server.go` — Versão do handshake MCP `v0.10.0`
- `internal/mcp/server_test.go` — Asserção de handshake MCP `v0.10.0` em `TestMCPInitializeAndPing`
- `internal/plugin/plugin_test.go` — Asserção de versão `0.10.0` em `TestOfficialPluginBundleVersion`
- `plugin/plugin.json` — Versão do manifesto `0.10.0`
- `README.md` — Referência de versão estável `v0.10.0`
- `docs-ai/DECISOES.md` — Consolidação da seção 12 para `v0.9.0 e v0.10.0`
- `docs-ai/artigos/artigo-sensoriamento-semantico-jev.md` — Artigo técnico completo em PT-BR
- `docs-ai/artigos/artigo-sensoriamento-semantico-jev.html` — Versão HTML estilizada para LinkedIn Articles / Substack
- `docs-ai/artigos/article-semantic-sensing-jev-go-devto.md` — Artigo técnico em Inglês para Dev.to
- `docs-ai/artigos/post-feed-linkedin-jev.txt` — Post de divulgação para o feed do LinkedIn
- `docs-ai/artigos/capa-artigo-sensoriamento-semantico-jev.jpg` — Imagem de capa 16:9 do artigo sobre o Jev
- `docs-ai/deliveries/DEL-029/report.md` — Status atualizado para `merged`
- `docs-ai/deliveries/INDEX.md` — `DEL-029` marcada como `merged` e `DEL-030` registrada
- `docs-ai/deliveries/DEL-030/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa, exit code 0)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -race -count=1 ./cmd/pipeline-ai ./internal/mcp ./internal/plugin ./internal/sensor/semantic
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	3.251s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.740s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.612s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	1.680s

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	2.110s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.415s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.820s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.210s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.240s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.490s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.520s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.640s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.230s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.280s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.215s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	8.410s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.510s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	4.320s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	0.310s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.195s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.890s
```

**Verificação do Plugin e CLI:**
```
$ go run ./cmd/pipeline-ai version
pipeline-ai v0.10.0

$ go run ./cmd/pipeline-ai plugin verify -dir plugin
Bundle do plugin em "plugin" é válido [OK].
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Bump de versão CLI para `v0.10.0` | `TestCLIRouting` em `cmd/pipeline-ai/main_test.go` |
| Bump de versão MCP para `v0.10.0` | `TestMCPInitializeAndPing` em `internal/mcp/server_test.go` |
| Validação de versão `0.10.0` e integridade do bundle oficial do plugin | `TestOfficialPluginBundleVersion` em `internal/plugin/plugin_test.go` |
| Suíte completa de regressão | `go test -count=1 ./...` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** nenhuma

**Sugestões recusadas:** nenhuma

# Rollback Plan

Esta entrega consiste exclusivamente em bump de versão (`v0.10.0`), publicação do pacote editorial do 4º artigo técnico e sincronização de metadados de release. Em caso de necessidade de reversão:
1. Reverter o commit com `git revert <commit_hash>`.
2. Remover a tag Git `v0.10.0` (caso criada) local e remotamente via `git tag -d v0.10.0 && git push --delete origin v0.10.0`.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
