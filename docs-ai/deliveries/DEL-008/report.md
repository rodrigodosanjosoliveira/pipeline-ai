---
delivery_id: DEL-008
task_code: TASK-008
title: Correção de vulnerabilidades de path traversal e falhas no hook e validador
status: merged
require_security: true
require_migration: false
require_performance: false
---

# Contexto

O contraditório técnico apontou quatro fragilidades críticas no pipeline-ai (v0.5.0):
1. **Path Traversal (C18):** No scaffolding do tipo `cli`, nomes contendo sequências relativas (como `../../escape`) escapavam da raiz do projeto destino.
2. **Bypass de Hook (C2b):** Ferramentas de edição invocadas com caminhos vazios ou sem identificação de arquivo passavam silenciosamente com permissão de escrita (`allow`), além do payload de `NotebookEdit` ignorar o campo `notebook_path`.
3. **Cegueira do Validador de Diff (C9):** A função `changedFiles` utilizava estritamente `git diff baseRef...HEAD`, deixando de enxergar alterações no working tree (unstaged), modificações em staging e novos arquivos untracked, permitindo que alterações sensíveis não commitadas fossem aprovadas sem checagem de superfície de segurança (D-14).
4. **Violação do Princípio do Menor Privilégio (C7):** O template `pipeline-ai.api.json` continha `query-performance` em `editor_agents`, apesar da skill ser estritamente de análise e revisão (D-04).

Esta entrega sana integralmente essas quatro falhas prioritárias (pacote P0), blindando o framework com salvaguardas rigorosas e testes automatizados.

# Escopo

- C18: Validação estrita de nome de projeto via regex e checagem de confinamento de diretórios em `internal/scaffold/scaffold.go`.
- C2b: Suporte a `tool_input.notebook_path` e bloqueio estrito (`replyDeny`) para ferramentas de edição com caminho vazio ou ausente em `internal/hook/hook.go`.
- C9: Agregação de `git diff` (commits, working tree unstaged, staged) e `git ls-files` (untracked) em `internal/validate/validate.go`.
- C7: Remoção de `query-performance` de `editor_agents` no template `internal/scaffold/templates/pipeline-ai.api.json`.

**Fora de escopo:**
- Reformulação total do hook para fail-closed genérico (C1) e modelo de gate por delivery efêmera com TTL (C3), mantidos para a release v0.6.0.
- Adicionar dependências externas ao projeto (permanece 100% Go stdlib pura).

# Implementação

1. **`internal/scaffold/scaffold.go`:**
   - Adicionada validação de regex `^[a-zA-Z0-9][a-zA-Z0-9_-]*$` no método `Normalize()` da struct `Options`, rejeitando caracteres de escape, caminhos absolutos e separadores.
   - Na função `createDirSkeleton()`, implementada verificação com `filepath.Clean` e `filepath.Rel` garantindo que todo caminho gerado permaneça confinado dentro da raiz do projeto.
2. **`internal/hook/hook.go`:**
   - Adicionado o campo `NotebookPath` em `ToolInput` e na inspeção de argumentos de `filePath()`.
   - Em `Run()`, chamadas de ferramentas de edição (`editTools`) com caminho vazio ou ausente agora retornam `replyDeny` imediato, fechando o bypass silencioso anterior.
3. **`internal/validate/validate.go`:**
   - `changedFiles` expandido para consolidar arquivos de quatro origens Git: `baseRef...HEAD`, `HEAD` (unstaged), `--cached` (staged) e `ls-files --others --exclude-standard` (untracked).
4. **`internal/scaffold/templates/pipeline-ai.api.json`:**
   - Removido `query-performance` de `editor_agents`.

**Arquivos alterados:**
- `internal/scaffold/scaffold.go` — Validação de nome de projeto e confinamento de caminhos
- `internal/scaffold/scaffold_test.go` — Testes de path traversal e rejeição de nomes maliciosos
- `internal/hook/hook.go` — Mapeamento de notebook_path e bloqueio de edições sem caminho
- `cmd/pipeline-hook/main_test.go` — Testes de bloqueio de edições vazias e NotebookEdit
- `internal/validate/validate.go` — Detecção de arquivos unstaged, staged e untracked
- `internal/validate/validate_test.go` — Teste unitário de detecção de arquivos no working tree e staging
- `internal/scaffold/templates/pipeline-ai.api.json` — Remoção de query-performance de editor_agents
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-008
- `docs-ai/deliveries/DEL-008/report.md` — Relatório de entrega da delivery

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.276s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.092s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.387s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.157s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.106s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.154s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.033s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.710s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.488s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| C18 (Scaffold Path Traversal) | `TestScaffoldValidation` e `TestScaffoldInvalidProjectNames` em `internal/scaffold/scaffold_test.go` |
| C2b (Hook Bypass & NotebookEdit) | `TestRun` e `TestAntigravityProtocol` em `cmd/pipeline-hook/main_test.go` |
| C9 (Validador de Diff Unstaged/Untracked) | `TestChangedFiles` em `internal/validate/validate_test.go` |
| C7 (Template API editor_agents) | `TestScaffoldAPI` e `TestTemplateAPIEditorAgents` em `internal/scaffold/scaffold_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Defesa em profundidade no scaffold (validação estrita de regex no nome combinada com verificação de escape por `filepath.Rel` na criação de pastas).

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- `internal/scaffold/scaffold.go` — Sanitização de input externo contra Path Traversal (CWE-22)
- `internal/hook/hook.go` — Bloqueio de ferramentas de edição sem caminho e suporte a notebook_path
- `internal/validate/validate.go` — Inclusão de arquivos unstaged, staged e untracked na auditoria de diff (D-14)
- `internal/scaffold/templates/pipeline-ai.api.json` — Remoção de privilégio indevido em editor_agents (PoLP)

**Achados:**
- Nenhum achado de vulnerabilidade residual (0 CRITICO, 0 ALTO, 0 MEDIO, 0 BAIXO).
- Zero dependências de terceiros no projeto (100% Go stdlib pura).
- Zero credenciais, tokens ou segredos introduzidos.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
