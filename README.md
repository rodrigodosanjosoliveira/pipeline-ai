# pipeline-ai

Ferramentas do padrão de novos projetos. Dois binários Go, sem dependências
externas além da stdlib.

| Binário | Função |
|---|---|
| `pipeline-hook` | Hook `PreToolUse` que impede edição direta de código fora dos agents autorizados (D-11) |
| `pipeline-validate` | Valida o relatório de delivery antes do commit (D-14, D-19) |

## Estado

Compilado e testado com **100% de sucesso em Go 1.24+** (stdlib pura, sem dependências externas).
Integrado nativamente ao ecossistema **Antigravity** via `.agents/hooks.json` e skills modulares.

## Instalação em um projeto

Requer Go 1.24 ou superior: a diretiva `tool` no `go.mod` não é aceita por toolchains anteriores.

```bash
go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook@v0.1.0
go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate@v0.1.0
```

A versão fica registrada no `go.mod` do projeto, permitindo auditar qual projeto usa qual versão do padrão sem conflito de dependências.

## Configuração

Copie o template correspondente de `examples/` para a raiz do seu projeto (`.pipeline-ai.json`):
- `examples/desktop/.pipeline-ai.json`: para aplicações desktop (Wails + Go).
- `examples/api/.pipeline-ai.json`: para serviços backend / API.

Campos suportados:
- `protected_paths`: caminhos que a sessão principal não pode editar diretamente.
- `editor_agents`: skills/agents autorizados a alterar código protegido.
- `security_surface_paths`: caminhos sensíveis que exigem auditoria de segurança (D-14).
- `enabled_modules`: módulos opcionais ativados pelo projeto (D-15), como `migration-reviewer`, `query-performance`, `release-ops`.

## Uso

### 1. Hook de Proteção em Tempo de Execução (`pipeline-hook`)

Configurado em `.agents/hooks.json` para ser acionado pelo Antigravity antes de ferramentas de edição (`write_to_file`, `replace_file_content`, `multi_replace_file_content`):

```json
{
  "pipeline-guard": {
    "PreToolUse": [
      {
        "matcher": "write_to_file|replace_file_content|multi_replace_file_content",
        "hooks": [
          {
            "type": "command",
            "command": "cd .. && go tool pipeline-hook"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "write_to_file|replace_file_content|multi_replace_file_content",
        "hooks": [
          {
            "type": "command",
            "command": "cd .. && gofmt -w . 2>/dev/null; exit 0"
          }
        ]
      }
    ]
  }
}
```

O hook recebe o payload no `stdin` e responde em JSON no `stdout`:
- `{"decision": "allow"}`: edição autorizada.
- `{"decision": "deny", "reason": "..."}`: edição bloqueada com justificativa legível ao modelo.

*(Também suporta o protocolo legado via stderr e exit code 2).*

### 2. Validador de Delivery (`pipeline-validate`)

Valida o relatório de entrega antes do commit e abertura de PR:

```bash
# Validação completa com checagem de diff do git
go tool pipeline-validate docs-ai/deliveries/DEL-001/report.md

# Pula as checagens dependentes de git
go tool pipeline-validate -no-git docs-ai/deliveries/DEL-001/report.md
```

#### Exit codes do validador

| Código | Significado |
|---|---|
| `0` | Aprovado (com ou sem avisos) |
| `1` | Reprovado (bloqueia o commit/delivery) |
| `2` | Erro de uso ou parâmetros |

### 3. Proteção de Branch Git (`pre-push`)

Para repositórios privados no plano gratuito do GitHub, um hook local em `.githooks/pre-push` impede pushes diretos para a branch `main`:

```bash
git config core.hooksPath .githooks
```

Toda entrega sobe obrigatoriamente via branch de feature (`task/*` ou `feat/*`) e Pull Request (`gh pr create`).

## Decisões de comportamento

- **Config ausente libera.** Um projeto sem `.pipeline-ai.json` não adota o pipeline, então o hook não interfere.
- **Config quebrada bloqueia.** Se o arquivo existir com JSON inválido ou schema incorreto, bloqueia imediatamente.
- **Git indisponível reprova fatalmente com ERRO.** Sem cálculo de diff contra a base, a salvaguarda de segurança (D-14) não pode garantir a integridade da entrega. Para ignorar voluntariamente, passe `-no-git`.

## Correções em relação ao kit original

| Problema | Correção |
|---|---|
| Hook lia `file_path` na raiz do JSON e nunca bloqueava | Lê `tool_input.file_path`; há teste de regressão |
| Hook dependia de `python3`/`jq` | Binário Go estático |
| Identificação do agent por `CLAUDE_AGENT` (inexistente) | Campo `agent_type` do payload |
| Sem lista de agents autorizados | `editor_agents` no config |
| Caminhos protegidos fixos no script | `protected_paths` no config |
| `TODO` só rejeitado se fosse o valor inteiro | Busca por substring, case-insensitive |
| Duas versões divergentes do validador | Uma implementação |
| `require_security` ligado por padrão | Desligado, com salvaguarda pelo diff |
