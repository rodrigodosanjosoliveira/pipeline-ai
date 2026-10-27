---
delivery_id: DEL-015
task_code: TASK-015
title: "Release v0.6.0: Consolidação de versão, changelog e encerramento operacional do contraditório técnico"
type: chore
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Esta entrega encerra formalmente o ciclo de resposta ao documento de contraditório técnico (*"Contraditório ao pipeline-ai"*), que comparou a arquitetura do framework contra pipelines de produção do Claude Code (`Guard-IA` e `ris-backend`).

Ao longo das entregas DEL-008 a DEL-014, todas as fragilidades críticas (P0), melhorias de hardening (P1) e evoluções funcionais da v0.6.0 foram tratadas com rigor:
- **DEL-008:** Correção de vulnerabilidades de path traversal e falhas no hook e validador (C18, C2b, C9, C7).
- **DEL-009:** Hardening do hook fail-closed, resolução canônica de symlinks e MCP scaffold dinâmico (C1, C2, C17).
- **DEL-010:** Modelo híbrido de autorização por delivery ativa, TTL de 7 dias e caminhos imutáveis (C3).
- **DEL-011:** Security Shift-Left bidirecional no PRE e POST com modelagem de ameaças e testes adversariais (C8).
- **DEL-012:** Flexibilização declarativa de IDs de entrega/tarefas, seções customizadas e índice opcional (C10, C11).
- **DEL-013:** Sensores de qualidade determinísticos em JSON e autoteste negativo contra falsos positivos (C13, C15).
- **DEL-014:** Checkpoints de invariantes, critérios formais de reversibilidade/rollback e gestão dinâmica de módulos (C14, C16).

Esta entrega formaliza o fechamento operacional do ciclo com o bump de versão para **v0.6.0**, registro das decisões D-21 a D-25 em `docs-ai/DECISOES.md`, atualização da documentação e encerramento das entregas no `INDEX.md`.

# Escopo

- Bump de versão para `v0.6.0`:
  - `cmd/pipeline-ai/main.go`: constante `version = "v0.6.0"`.
  - `internal/mcp/server.go`: handshake MCP `serverInfo.version = "v0.6.0"`.
- Documentação e Decisões:
  - `README.md`: documentação completa dos novos comandos da v0.6.0 (`delivery`, `sensor`, `module`, flags `-json` e tools MCP).
  - `docs-ai/DECISOES.md`: inclusão da seção 9 registrando as decisões arquiteturais D-21 a D-25.
  - `docs-ai/deliveries/INDEX.md`: atualização dos status de DEL-008 a DEL-014 para `merged`.
  - `docs-ai/deliveries/DEL-008` a `DEL-014/report.md`: frontmatter atualizado para `status: merged`.
- Criação e validação do relatório da release DEL-015.

**Fora de escopo:**
- Alterações em regras de negócio ou novas funcionalidades.
- Adição de dependências externas (100% Go 1.24+ standard library).

# Implementação

1. **Versionamento Central:**
   - Constante `version` em `cmd/pipeline-ai/main.go` atualizada para `"v0.6.0"`.
   - Handshake MCP `initialize` em `internal/mcp/server.go` atualizado para `"v0.6.0"`.
2. **Documentação do Ecossistema:**
   - Atualizado o `README.md` com instruções de uso e instalação para a v0.6.0, tabela de comandos e detalhes das ferramentas MCP.
   - Formalizadas as decisões arquiteturais D-21 a D-25 em `docs-ai/DECISOES.md`.
3. **Governança de Deliveries:**
   - Registros do `INDEX.md` sincronizados com o encerramento das entregas DEL-008 a DEL-014.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — Bump de versão para v0.6.0
- `cmd/pipeline-ai/main_test.go` — Teste de saída da versão v0.6.0
- `internal/mcp/server.go` — Handshake MCP na v0.6.0
- `internal/mcp/server_test.go` — Asserção de versão MCP v0.6.0
- `README.md` — Documentação dos comandos da v0.6.0
- `docs-ai/DECISOES.md` — Registro das decisões D-21 a D-25
- `docs-ai/deliveries/INDEX.md` — Encerramento das entregas anteriores
- `docs-ai/deliveries/DEL-008/report.md` a `DEL-014/report.md` — Status merged
- `docs-ai/deliveries/DEL-015/report.md` — Relatório da release v0.6.0

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.485s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.111s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.615s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.205s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.090s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.426s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.036s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.996s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	2.904s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.104s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.704s
PASS

$ go test -race ./...
PASS
```

**Execução de Sensores (`pipeline-ai sensor -json`):**

```json
{
  "overall": "PASS",
  "ok": true,
  "tests_run": 388,
  "sensors": [
    {
      "name": "fmt",
      "status": "PASS",
      "message": "Todos os arquivos estão formatados segundo o padrão gofmt."
    },
    {
      "name": "vet",
      "status": "PASS",
      "message": "go vet executado com sucesso sem avisos ou erros."
    },
    {
      "name": "test",
      "status": "PASS",
      "message": "Todos os testes passaram (388 testes executados, 0 pulados)."
    }
  ],
  "message": "Todos os sensores passaram com sucesso (388 testes executados)."
}
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Bump da versão da CLI para v0.6.0 | `TestCLIRouting` ("comando version") em `cmd/pipeline-ai/main_test.go` |
| Bump da versão MCP para v0.6.0 | `TestMCPInitializeAndPing` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Atualização simultânea dos frontmatters de relatórios anteriores em `docs-ai/deliveries/` garantindo total consistência no repositório.

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
