---
delivery_id: DEL-002
task_code: TASK-002
title: Release v0.4.0 e encerramento operacional de DEL-001
type: chore
status: draft
require_security: false
require_migration: false
require_performance: false
date: 2026-10-21
---

# Contexto

Com a implementação e unificação das ferramentas do pipeline-ai em um comando único (`cmd/pipeline-ai`), foram incorporados os subcomandos `sync` e `doctor` através da entrega DEL-001. Para fechar o ciclo de release da versão estável `v0.4.0`, é necessário atualizar as referências de documentação no `README.md`, adicionar regras de exclusão no `.gitignore`, atualizar o status de DEL-001 para `merged` e preparar o repositório para o corte da tag `v0.4.0`.

# Escopo

- Atualização de status de DEL-001 para `merged` em `docs-ai/deliveries/INDEX.md` e `docs-ai/deliveries/DEL-001/report.md`.
- Adição da entrada `.idea/` no `.gitignore`.
- Atualização do `README.md`:
  - Versão estável indicada como `v0.4.0`.
  - Exemplos de instalação atualizados para o binário unificado `cmd/pipeline-ai@v0.4.0`.
- Registro e validação da entrega DEL-002.
- Preparação para lançamento da tag Git `v0.4.0`.

**Fora de escopo:**
- Modificações em código Go de `cmd/` ou `internal/`.
- Adição de novas dependências externas.

# Implementação

Foram aplicadas melhorias pontuais de documentação e configuração:
- `.gitignore`: adicionado `.idea/` para ignorar metadados do GoLand / IntelliJ.
- `docs-ai/deliveries/INDEX.md`: DEL-001 marcada como `merged` e DEL-002 registrada.
- `docs-ai/deliveries/DEL-001/report.md`: cabeçalho de status atualizado de `draft` para `merged`.
- `README.md`: documentação revisada com indicação da versão `v0.4.0` e instrução de instalação `go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai@v0.4.0`.

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.150s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.013s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	0.302s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.014s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.185s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.012s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.106s
```

# Code Review

**Bloqueios:** Nenhum.

**Sugestões acatadas:**
- Manter menção aos binários individuais legados no README para preservar retrocompatibilidade de documentação.
- Incluir regra `.idea/` no `.gitignore` para manter workspace limpo em ambientes JetBrains.

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
