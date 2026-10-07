# {{NOME_DO_PROJETO}}

{{UMA_FRASE_SOBRE_O_QUE_O_PROJETO_FAZ}}

**Formato:** {{desktop | api}} · **Banco:** {{PostgreSQL | MongoDB | DynamoDB}}

---

## Como o trabalho acontece aqui

Este projeto usa pipeline de agents. **Você, na sessão principal, não edita
código.** Um hook bloqueia edições em caminhos protegidos quando a chamada não
vem de um agent autorizado.

Isso não é burocracia. Com IA, o gargalo do desenvolvimento deixa de ser gerar
código e passa a ser verificá-lo, e refatoração iterativa sem validação externa
degrada rápido: os erros se acumulam a cada passada. Os agents existem para que
cada passo tenha um gate determinístico.

**Toda demanda começa em `/triage`.** O triage roteia para o agent certo.

## Fluxo

```
triage
  → architecture-guardrails
  → [data-model]
  → test-writer
  → dev-implementation
  → code-reviewer
  → [migration-reviewer] [query-performance]
  → security-review        (quando houver superfície de segurança)
  → supply-chain-review    (quando go.mod/go.sum mudarem)
  → qa-validation
  → [release-ops]
  → docs-update
  → git-delivery
```

Colchetes são módulos opcionais, ativados em `.pipeline-ai.json`.

## Gates

| Comando | Quando |
|---|---|
| `gofmt -l .` | a cada edição (automático via hook) |
| `go build ./...` | dev |
| `go vet ./...` | dev e CI |
| `go test ./...` | dev, QA e CI |
| `govulncheck ./...` | supply-chain-review e CI |
| `go test -fuzz=. -fuzztime=30s` | parsing, validação e desserialização |

Fuzz **sempre** com `-fuzztime`. Sem limite, o alvo roda indefinidamente e trava
o agent.

## Invariantes

1. **Go 1.24+.** A diretiva `tool` no `go.mod` não é aceita por versões
   anteriores.
2. **Testes com a stdlib `testing`**, table-driven. Testify só se este projeto
   optar explicitamente.
3. **Sem CGO** em projeto desktop. É o que mantém a cross-compilação do WSL2 para
   Windows funcionando sem Docker.
4. **Dependência externa precisa de justificativa.** A stdlib do Go cobre a maior
   parte, e cada dependência é superfície de supply chain. Mudança em `go.mod`
   dispara `supply-chain-review`.
5. **Versão do `wails3` fixada.** O v3 está em beta; um build não pode variar
   entre máquinas.
6. **Nada de segredo em código.** Credenciais vêm de ambiente.

## Git

- Branch: `task/{code}`
- Commit: `{type}({scope}): {summary} [{task_code}]`
- PR no GitHub. `git push` e `gh pr create` pedem confirmação.

## Delivery

Toda entrega gera um relatório em `docs-ai/deliveries/{DELIVERY_ID}/report.md` e
precisa estar registrada em `docs-ai/deliveries/INDEX.md`.

Antes do commit:

```bash
go tool pipeline-validate docs-ai/deliveries/{DELIVERY_ID}/report.md
```

O validador reprova a delivery se o diff tocar superfície de segurança com
`require_security` desligado. Ligar a flag é do triage, mas o validador não
depende disso.

## Estrutura

<!-- Mantenha atualizado. É o mapa que os agents usam para se localizar. -->

```
{{ARVORE_DE_DIRETORIOS}}
```

## Onde está o resto

| Arquivo | Conteúdo |
|---|---|
| `docs-ai/DECISOES.md` | Por que o padrão é como é |
| `docs-ai/01-INVARIANTES-GLOBAIS.md` | Invariantes específicos deste projeto |
| `docs-ai/03-MAPA-RAPIDO-MODULOS.md` | Mapa detalhado dos módulos |
| `.pipeline-ai.json` | Caminhos protegidos, agents editores, superfície de segurança |
