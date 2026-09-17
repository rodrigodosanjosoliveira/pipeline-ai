# pipeline-ai

Ferramentas do padrão de novos projetos. Dois binários Go, sem dependências
externas além da stdlib.

| Binário | Função |
|---|---|
| `pipeline-hook` | Hook `PreToolUse` que impede edição direta de código fora dos agents autorizados (D-11) |
| `pipeline-validate` | Valida o relatório de delivery antes do commit (D-14, D-19) |

## Estado

**Este código ainda não foi compilado nem executado.** Foi escrito fora de um
ambiente com Go disponível. Rode `go vet ./...` e `go test ./...` antes de usar.

## Antes de tudo

O `go.mod` está com um caminho de módulo provisório:

```
module github.com/rodrigodosanjosoliveira/pipeline-ai
```

Troque `SEU-USUARIO` pelo usuário real do GitHub, aqui e nos imports dos três
arquivos que referenciam o módulo.

## Instalação em um projeto

Requer Go 1.24 ou superior: a diretiva `tool` no `go.mod` não é aceita por
toolchains anteriores.

```bash
go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook@v0.1.0
go get -tool github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate@v0.1.0
```

A versão fica registrada no `go.mod` do projeto, então dá para auditar qual
projeto usa qual versão do padrão, e projetos diferentes podem rodar versões
diferentes sem conflito.

## Configuração

Copie `examples/.pipeline-ai.json` para a raiz do projeto e ajuste
`protected_paths` conforme o formato (`desktop` ou `api`).

Os caminhos protegidos ficam aqui, e não no binário, porque o padrão suporta dois
formatos de projeto com estruturas de diretório diferentes.

## Uso

```bash
# hook: recebe o payload do PreToolUse no stdin
go tool pipeline-hook

# validador
go tool pipeline-validate docs-ai/deliveries/DEL-001/report.md
go tool pipeline-validate -no-git docs-ai/deliveries/DEL-001/report.md
```

### Exit codes

**pipeline-hook**

| Código | Significado |
|---|---|
| 0 | Permitido |
| 2 | Bloqueado; a razão vai no stderr e o Claude Code a mostra ao modelo |

**pipeline-validate**

| Código | Significado |
|---|---|
| 0 | Aprovado (com ou sem avisos) |
| 1 | Reprovado |
| 2 | Erro de uso ou ambiente |

## Decisões de comportamento

Estes pontos foram escolhas deliberadas e merecem revisão:

- **Config ausente libera.** Um projeto sem `.pipeline-ai.json` não adota o
  pipeline, então o hook não interfere.
- **Config quebrada bloqueia.** Seguir em frente daria a falsa impressão de que a
  proteção está ativa.
- **Payload ilegível libera.** As deny rules de permissão continuam sendo a
  camada que de fato barra, então liberar aqui não abre brecha. Evita travar a
  sessão por um problema de formato.
- **Git indisponível vira aviso, não erro.** A salvaguarda de segurança não roda,
  mas a delivery não é reprovada por causa do ambiente. Isso enfraquece o D-14 e
  está em aberto.

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
