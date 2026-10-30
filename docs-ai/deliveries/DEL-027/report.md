---
delivery_id: DEL-027
task_code: TASK-027
title: "Release v0.9.0: Consolidação do sensoriamento semântico Jev (CLI eval e MCP pipeline_eval_semantic)"
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Esta entrega formaliza o encerramento operacional do ciclo **v0.9.0** do framework Pipeline-AI, consolidando o subsistema de sensoriamento semântico e decisões calibradas com modelos "System One" (Jev) introduzido nas duas entregas precedentes:
1. **Pacote de Sensoriamento Semântico em Go Puro (DEL-025):** Implementação do pacote `internal/sensor/semantic` utilizando exclusivamente a standard library do Go 1.24+, fornecendo tipos de decisão calibrada (`NoulDecision`, `ChoiceDecision`, `ScoreDecision`), política fail-closed segura, sanitização automática de credenciais (`SEC-01`), mitigação de DoS em respostas (`SEC-02`) e Fuzz testing contínuo (`FuzzParseNoulResponse`).
2. **Integração Operacional via CLI e Protocolo MCP (DEL-026):** Exposição das capacidades de avaliação semântica tanto para desenvolvedores humanos quanto para agentes autônomos por meio do subcomando CLI `pipeline-ai eval` (com suporte polimórfico a arquivo, string ou `stdin`) e da ferramenta MCP `pipeline_eval_semantic`, protegidos por limite de 512KB na entrada, redação de tokens e timeout mandatório via `context.Context`.

Esta delivery oficializa o bump de versão central para **v0.9.0**, consolida a seção 12 das decisões arquiteturais D-33 e D-34 em `docs-ai/DECISOES.md`, atualiza o `README.md`, adiciona cobertura automatizada de versão do bundle oficial do plugin e sincroniza os relatórios de delivery em `docs-ai/deliveries/INDEX.md`.

# Escopo

- Bump de versão para `v0.9.0`:
  - `cmd/pipeline-ai/main.go`: constante `version = "v0.9.0"`.
  - `cmd/pipeline-ai/main_test.go`: asserção de versão `"v0.9.0"` em `TestCLIRouting`.
  - `internal/mcp/server.go`: handshake MCP `serverInfo.version = "v0.9.0"`.
  - `internal/mcp/server_test.go`: asserção de versão MCP `"v0.9.0"` em `TestMCPInitializeAndPing`.
  - `plugin/plugin.json`: campo `"version": "0.9.0"`.
  - `internal/plugin/plugin_test.go`: novo teste `TestOfficialPluginBundleVersion` assegurando que `plugin/plugin.json` declare `"0.9.0"` e valide integralmente via `VerifyBundle`.
  - `README.md`: indicação de versão estável `v0.9.0`.
- Consolidação de Decisões Arquiteturais:
  - `docs-ai/DECISOES.md`: atualização da seção 12 (`Consolidação Arquitetural da v0.9.0`) documentando formalmente as decisões D-33 e D-34.
- Governança de Deliveries:
  - `docs-ai/deliveries/DEL-025/report.md` e `docs-ai/deliveries/DEL-026/report.md`: status sincronizado para `merged`.
  - `docs-ai/deliveries/INDEX.md`: atualização de DEL-025 e DEL-026 para `merged` e registro de DEL-027 como `draft`.
  - `docs-ai/deliveries/DEL-027/report.md`: criação e validação do relatório da release v0.9.0.

**Fora de escopo:**
- Adição de dependências externas (100% Go 1.24+ standard library mantido).
- Alterações nas regras de negócio do cliente semântico ou dos contratos CLI/MCP.

# Implementação

1. **Versionamento Central e Sincronizado:**
   - Constante `version` em `cmd/pipeline-ai/main.go` atualizada para `"v0.9.0"`.
   - Handshake MCP `initialize` em `internal/mcp/server.go` atualizado para `"v0.9.0"`.
   - Manifesto oficial `plugin/plugin.json` atualizado para `"0.9.0"`.
   - Testes unitários de versão (`cmd/pipeline-ai/main_test.go`, `internal/mcp/server_test.go` e `internal/plugin/plugin_test.go`) sincronizados.
2. **Decisões Arquiteturais:**
   - Consolidação da seção 12 (decisões D-33 e D-34) em `docs-ai/DECISOES.md`.
3. **Documentação e Índice:**
   - `README.md` atualizado com a versão estável `v0.9.0`.
   - `docs-ai/deliveries/DEL-025/report.md` e `docs-ai/deliveries/DEL-026/report.md` atualizados para `status: merged`.
   - `docs-ai/deliveries/INDEX.md` sincronizado com status consolidado das entregas e registro de `DEL-027`.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Bump de versão para v0.9.0
- `cmd/pipeline-ai/main_test.go` — Teste unitário de versão v0.9.0 em `TestCLIRouting`
- `internal/mcp/server.go` — Versão do handshake MCP v0.9.0
- `internal/mcp/server_test.go` — Teste de handshake MCP v0.9.0 em `TestMCPInitializeAndPing`
- `internal/plugin/plugin_test.go` — Teste `TestOfficialPluginBundleVersion` validando versão 0.9.0 do bundle oficial
- `plugin/plugin.json` — Versão do plugin 0.9.0
- `README.md` — Referência de versão estável v0.9.0
- `docs-ai/DECISOES.md` — Consolidação da seção 12 (D-33 e D-34)
- `docs-ai/deliveries/DEL-025/report.md` — Status merged
- `docs-ai/deliveries/DEL-026/report.md` — Status merged
- `docs-ai/deliveries/INDEX.md` — Atualização de status de DEL-025/DEL-026 e registro de DEL-027
- `docs-ai/deliveries/DEL-027/report.md` — Relatório de release v0.9.0

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -race -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	7.367s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	1.119s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	3.806s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.692s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.659s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	1.007s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	1.464s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	3.816s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.752s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	0.840s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.618s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	15.654s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	2.878s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	8.067s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	0.630s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.528s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	1.749s

$ go test -fuzz=FuzzValidateManifest -fuzztime=5s ./internal/plugin/...
fuzz: elapsed: 0s, gathering baseline coverage: 0/6 completed
fuzz: elapsed: 0s, gathering baseline coverage: 6/6 completed, now fuzzing with 10 workers
fuzz: elapsed: 3s, execs: 232784 (77593/sec), new interesting: 1 (total: 7)
fuzz: elapsed: 5s, execs: 363241 (65224/sec), new interesting: 1 (total: 7)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	5.588s
```

**Verificação do Plugin e CLI:**
```
$ go run ./cmd/pipeline-ai version
pipeline-ai v0.9.0

$ go run ./cmd/pipeline-ai plugin verify -dir plugin
Bundle do plugin em "plugin" é válido [OK].
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Bump de versão CLI para v0.9.0 | `TestCLIRouting` em `cmd/pipeline-ai/main_test.go` |
| Bump de versão MCP para v0.9.0 | `TestMCPInitializeAndPing` em `internal/mcp/server_test.go` |
| Validação de versão 0.9.0 e integridade do bundle oficial do plugin | `TestOfficialPluginBundleVersion` em `internal/plugin/plugin_test.go` |
| Resiliência de parsing do manifesto do plugin | `FuzzValidateManifest` em `internal/plugin/plugin_test.go` |
| Suíte completa de regressão com detector de corrida | `go test -race -count=1 ./...` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** inclusão do teste `TestOfficialPluginBundleVersion` em `internal/plugin/plugin_test.go` para garantir o sincronismo permanente de versão do manifesto oficial `plugin/plugin.json` junto ao CLI e ao servidor MCP.

**Sugestões recusadas:** nenhuma

# Rollback Plan

Esta entrega consiste exclusivamente em bump de versão, consolidação de documentação arquitetural e metadados de release. Em caso de necessidade de reversão:
1. Reverter o commit com `git revert <commit_hash>`.
2. Remover a tag Git `v0.9.0` (caso criada) local e remotamente via `git tag -d v0.9.0 && git push --delete origin v0.9.0`.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md

