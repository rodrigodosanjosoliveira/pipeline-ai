---
delivery_id: DEL-024
task_code: TASK-024
title: "Release v0.8.0: Dashboard visual interativo Generative UI e Plugin oficial do Google Antigravity"
status: merged
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Esta entrega formaliza o encerramento operacional do ciclo **v0.8.0** do framework Pipeline-AI, consolidando duas grandes inovações:
1. **Dashboard Generative UI (DEL-022):** Interface visual interativa e autocontida para acompanhamento das métricas DORA, esteira de 16 skills e histórico de entregas, acessível via terminal (`pipeline-ai dashboard`) e embed inline (`pipeline_get_dashboard_html`).
2. **Plugin Oficial do Google Antigravity (DEL-023):** Empacotamento completo do toolkit conforme o schema oficial de plugins (`plugin/`), com manifesto `plugin.json`, servidor MCP nativo sobre stdio, hooks de ciclo de vida e subcomando de instalação em 1 comando (`pipeline-ai plugin install`).

Esta delivery oficializa o bump de versão central para **v0.8.0**, documenta formalmente as decisões arquiteturais D-31 e D-32 em `docs-ai/DECISOES.md`, atualiza o `README.md` e consolida os registros no `INDEX.md`.

# Escopo

- Bump de versão para `v0.8.0`:
  - `cmd/pipeline-ai/main.go`: constante `version = "v0.8.0"`.
  - `cmd/pipeline-ai/main_test.go`: asserção de versão `"v0.8.0"`.
  - `internal/mcp/server.go`: handshake MCP `serverInfo.version = "v0.8.0"`.
  - `internal/mcp/server_test.go`: asserção de versão MCP `"v0.8.0"`.
  - `plugin/plugin.json`: campo `"version": "0.8.0"`.
  - `README.md`: indicação de versão estável `v0.8.0`.
- Formalização de Decisões Arquiteturais:
  - `docs-ai/DECISOES.md`: inclusão da seção 11 registrando formalmente as decisões D-31 e D-32.
- Governança de Deliveries:
  - `docs-ai/deliveries/INDEX.md`: atualização de DEL-023 para `merged` e registro de DEL-024 como `draft`.
  - `docs-ai/deliveries/DEL-023/report.md`: status sincronizado para `merged`.
- Criação e validação do relatório da release DEL-024.

**Fora de escopo:**
- Adição de dependências externas (100% Go 1.24+ standard library mantido).
- Alterações em regras de negócio prévias.

# Implementação

1. **Versionamento Central:**
   - Constante `version` em `cmd/pipeline-ai/main.go` atualizada para `"v0.8.0"`.
   - Handshake MCP `initialize` em `internal/mcp/server.go` atualizado para `"v0.8.0"`.
   - Manifest `plugin/plugin.json` atualizado para `"0.8.0"`.
   - Testes unitários de versão (`main_test.go` e `server_test.go`) sincronizados.
2. **Decisões Arquiteturais:**
   - Formalização das decisões D-31 e D-32 em `docs-ai/DECISOES.md`.
3. **Documentação e Índice:**
   - `README.md` atualizado com a versão estável `v0.8.0`.
   - `docs-ai/deliveries/INDEX.md` sincronizado com status consolidado das entregas.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Bump de versão para v0.8.0
- `cmd/pipeline-ai/main_test.go` — Teste unitário de versão v0.8.0
- `internal/mcp/server.go` — Versão do handshake MCP v0.8.0
- `internal/mcp/server_test.go` — Teste de handshake MCP v0.8.0
- `plugin/plugin.json` — Versão do plugin v0.8.0
- `README.md` — Referência de versão estável v0.8.0
- `docs-ai/DECISOES.md` — Formalização das decisões D-31 e D-32
- `docs-ai/deliveries/INDEX.md` — Atualização de status e registro de DEL-024
- `docs-ai/deliveries/DEL-023/report.md` — Status merged
- `docs-ai/deliveries/DEL-024/report.md` — Relatório de release v0.8.0

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	7.408s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.312s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	2.991s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.547s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.136s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.263s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.029s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	4.511s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.174s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.451s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.070s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	13.672s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.864s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	6.624s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.288s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	1.238s
```

**Verificação do Plugin:**
```
$ pipeline-ai plugin verify -dir plugin
Bundle do plugin em "plugin" é válido [OK].
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Bump de versão CLI para v0.8.0 | `TestVersionCommand` em `cmd/pipeline-ai/main_test.go` |
| Bump de versão MCP para v0.8.0 | `TestServer_Initialize` em `internal/mcp/server_test.go` |
| Validação de manifesto do plugin v0.8.0 | `TestValidateManifest` em `internal/plugin/plugin_test.go` |
| Verificação de compilação de todos os pacotes | `go build ./...` |
| Suíte completa de testes de regressão | `go test -count=1 ./...` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** consolidação simultânea da versão em `plugin/plugin.json`, `README.md`, CLI e servidor MCP.

**Sugestões recusadas:** nenhuma

# Rollback Plan

Esta entrega consiste exclusivamente em bump de versão, formalização de decisões arquiteturais e metadados de release. Em caso de necessidade de reversão:
1. Reverter o commit com `git revert <commit_hash>`.
2. Remover a tag Git `v0.8.0` (caso criada) local e remotamente via `git tag -d v0.8.0 && git push --delete origin v0.8.0`.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
