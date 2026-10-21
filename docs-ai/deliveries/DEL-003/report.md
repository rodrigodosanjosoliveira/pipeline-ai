---
delivery_id: DEL-003
task_code: TASK-003
title: Suporte a saída estruturada JSON na ferramenta doctor (--json)
type: feat
status: merged
require_security: false
require_migration: false
require_performance: false
date: 2026-10-21
---

# Contexto

O subcomando `pipeline-ai doctor` foi concebido para validar o ambiente local e configurações de projetos derivados. No entanto, para integração em pipelines de CI, observabilidade e scripts automatizados de checagem prévia, era necessária uma forma legível por máquina de consumir o status dos diagnósticos.

Esta entrega adiciona o suporte à flag `--json` ao `pipeline-ai doctor`, emitindo um payload JSON estruturado contendo status booleano global (`ok`) e detalhes individuais de cada checagem (`name`, `status`, `message`).

# Escopo

- Adição de tags de serialização JSON (`json:"..."`) no modelo `CheckResult`.
- Criação do tipo `Report` agrupando status booleano global (`ok`) e coleção de resultados (`results`).
- Criação das funções `Collect(rootDir string)` e `RunJSON(rootDir string, stdout io.Writer)` em `internal/doctor`.
- Suporte à flag `-json` / `--json` no roteamento de `cmd/pipeline-ai/main.go`.
- Testes unitários para `RunJSON` e teste de integração na CLI.
- Atualização do `README.md`.

**Fora de escopo:**
- Modificações em outros subcomandos (`init`, `sync`, `hook`, `validate`).
- Adição de dependências externas ao `go.mod`.

# Implementação

A funcionalidade foi construída estritamente com `encoding/json` da stdlib:
- `internal/doctor/doctor.go`:
  - `CheckResult` recebeu tags JSON `name`, `status` e `message`.
  - Adicionado o tipo `Report` com campos `OK bool` e `Results []CheckResult`.
  - Refatoração determinística separando a coleta de métricas (`Collect`) da renderização em texto (`Run`) ou JSON (`RunJSON`).
  - `RunJSON` utiliza indentação de 2 espaços (`enc.SetIndent("", "  ")`) para clareza em logs e stdout.
  - O código de saída permanece padronizado: `0` para sucesso (ou apenas warnings) e `1` para falha crítica.
- `cmd/pipeline-ai/main.go`:
  - `runDoctor` registra `fs.Bool("json", false, ...)` e delega para `doctor.RunJSON`.
- `README.md`:
  - Documentação da flag `--json` no catálogo de comandos.

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.287s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.247s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	1.182s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.089s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Serialização e integridade do JSON | `TestRunDoctorJSON` em `internal/doctor/doctor_test.go` |
| Chamada da CLI com flag `-json` | `TestCLISyncAndDoctor` em `cmd/pipeline-ai/main_test.go` |

# Code Review

**Bloqueios:** Nenhum.

**Sugestões acatadas:**
- Refatoração da função `Collect` para garantir que o mesmo pipeline de diagnósticos seja executado indistintamente entre a renderização textual e JSON.
- Manutenção da saída JSON no `stdout` mantendo eventuais erros operacionais no `stderr`.

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
