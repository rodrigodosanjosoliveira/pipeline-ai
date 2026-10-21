---
delivery_id: DEL-006
task_code: TASK-006
title: Release v0.5.0: Bump de versão, MCP completo e encerramento operacional
type: chore
status: draft
require_security: false
require_migration: false
require_performance: false
date: 2026-10-21
---

# Contexto

Com a implementação das entregas DEL-003 (suporte à flag `--json` no `doctor`), DEL-004 (servidor MCP nativo sobre `stdio` com `pipeline_doctor` e `pipeline_validate`) e DEL-005 (expansão MCP com Resources `pipeline://` e Tools `pipeline_scaffold_delivery` e `pipeline_sync_skills`), o toolkit atinge um marco de maturidade relevante, consolidando a governança como um protocolo interoperável de primeira classe para agentes de IA e IDEs como o Antigravity.

Esta entrega formaliza o fechamento operacional do ciclo, realiza o bump de versão para `v0.5.0`, atualiza a documentação e prepara a base para a criação da tag Git e GitHub Release `v0.5.0`.

# Escopo

- Bump de versão para `v0.5.0` em `cmd/pipeline-ai/main.go` e `internal/mcp/server.go`.
- Atualização do status de DEL-005 para `merged` em `docs-ai/deliveries/INDEX.md` e `DEL-005/report.md`.
- Atualização do `README.md` com a versão estável `v0.5.0` e os novos comandos de instalação.
- Criação e validação do relatório DEL-006 e registro no `INDEX.md`.
- Publicação da tag Git `v0.5.0` e GitHub Release oficial.

**Fora de escopo:**
- Novas funcionalidades ou alterações estruturais além do versionamento e governança.
- Adição de dependências externas.

# Implementação

- `cmd/pipeline-ai/main.go`: constante `version` atualizada de `v0.4.0` para `v0.5.0`.
- `internal/mcp/server.go`: handshake de inicialização MCP (`serverInfo.version`) atualizado para `v0.5.0`.
- `README.md`: indicação da versão estável `v0.5.0` e instrução `go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai@v0.5.0`.
- `docs-ai/deliveries/INDEX.md`: DEL-005 marcada como `merged` e DEL-006 registrada.
- `docs-ai/deliveries/DEL-005/report.md`: cabeçalho atualizado para `status: merged`.

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia)

$ go build ./...
(saída vazia)

$ go vet ./...
(saída vazia)

$ go test -race ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.196s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.090s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	(cached)
```

# Code Review

**Bloqueios:** Nenhum.

**Sugestões acatadas:**
- Sincronizar a versão exposta pelo handshake MCP (`initialize`) com a constante central da CLI para manter consistência semântica.
- Manter o registro das ferramentas legadas no README para compatibilidade total.

**Sugestões recusadas:** Nenhuma.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
