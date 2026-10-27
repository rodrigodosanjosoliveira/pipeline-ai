---
delivery_id: DEL-019
task_code: TASK-019
title: "Suporte automático a cross-compilação desktop (CGO_ENABLED=0 GOOS=windows) no sensor e runner"
type: feat
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

O Invariante 3 do framework estabelece: *"Sem CGO em projeto desktop para manter cross-compilação WSL2 -> Windows"*.

Em projetos desktop (`project_type: "desktop"`, baseados em Go + Wails v3 + React), a execução padrão de `go vet` e `go test` em hosts Linux ou WSL2 sem flags explícitas tentava invocar os alvos nativos de Linux de bibliotecas gráficas (como WebKitGTK e GTK3), exigindo `pkg-config` e dependências CGO que quebravam a validação local.

Esta entrega atualiza o pacote `internal/sensor`:
- O sensor detecta automaticamente quando o projeto é do tipo `"desktop"`.
- Injeta as variáveis de ambiente de cross-compilação Windows sem CGO (`CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64`) para os comandos `go vet` e `go test`.
- Mantém o comportamento inalterado para projetos do tipo `"api"` ou `"cli"`.

# Escopo

- Em `internal/sensor/sensor.go`:
  - Carregamento da configuração do projeto via `config.Load(absDir)`.
  - Injeção condicional de `CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64` no ambiente dos processos `vetCmd` e `testCmd` quando `cfg.ProjectType == "desktop"`.
- Em `internal/sensor/sensor_test.go`:
  - Testes unitários com mock de toolchain verificando a injeção estrita das variáveis para projetos desktop e a preservação do ambiente padrão para projetos API e CLI.
- Validação real no projeto desktop consumidor `sistema-estoque`, alcançando 100% de aprovação (327 testes executados com status geral PASS no `pipeline-ai run`).

**Fora de escopo:**
- Alterações nos comandos de compilação de projetos API ou CLI.
- Adição de dependências externas.

# Implementação

1. **`internal/sensor/sensor.go`:**
   - Adicionado carregamento de `cfg` no início da função `Run`.
   - Criada a fatia `cmdEnv` preservando as variáveis do sistema operacional e adicionando `CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64` caso `cfg.ProjectType == "desktop"`.
   - Atribuído `cmdEnv` a `vetCmd.Env` e `testCmd.Env`.
2. **`internal/sensor/sensor_test.go`:**
   - Adicionado o teste `TestRunSensors_DesktopEnvironment` cobrindo projetos desktop, api e cli.

**Arquivos alterados:**
- `internal/sensor/sensor.go` — Injeção de cross-compilação para projetos desktop
- `internal/sensor/sensor_test.go` — Testes unitários de injeção de ambiente
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-019
- `docs-ai/deliveries/DEL-019/report.md` — Relatório de entrega da delivery

# Testes

**Gates:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso)

$ go vet ./...
(análise estática sem avisos)

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	6.988s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.215s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	3.318s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.414s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.160s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.010s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	4.375s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.052s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	10.095s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.738s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	6.471s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.202s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	1.036s
PASS
```

**Execução Integrada no Consumidor Desktop (`sistema-estoque`):**

```
=== pipeline-ai runner ===

 [PASS] doctor     : Ambiente validado com sucesso
 [PASS] sensor     : sensores aprovados (327 testes executados)
 [SKIP] validate   : Nenhuma delivery ativa ou relatório informado; validação ignorada

Status geral: PASS
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Injeção de cross-compilação Windows sem CGO para desktop | `TestRunSensors_DesktopEnvironment` em `internal/sensor/sensor_test.go` |
| Preservação de ambiente padrão para API e CLI | `TestRunSensors_DesktopEnvironment` em `internal/sensor/sensor_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Preservar todo o ambiente do sistema operacional (`os.Environ()`) adicionando as variáveis de cross-compilação ao final para que `PATH` e outras configurações de toolchain continuem operacionais.

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
