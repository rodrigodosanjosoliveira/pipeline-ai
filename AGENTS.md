# {{NOME_DO_PROJETO}}

{{UMA_FRASE_SOBRE_O_QUE_O_PROJETO_FAZ}}

**Formato:** {{desktop | api}} · **Banco:** {{PostgreSQL | MongoDB | DynamoDB}}

---

## Como o trabalho acontece aqui

Este projeto adota o padrão **pipeline-ai** integrado nativamente ao ecossistema **Antigravity**. **Você, na sessão principal, atua como o Orquestrador da esteira e não edita código diretamente.** O hook de ciclo de vida `PreToolUse` bloqueia edições em caminhos protegidos quando a chamada não vem de um agent autorizado em `.pipeline-ai.json`.

Com IA, o gargalo do desenvolvimento não é gerar código, mas verificá-lo. Os agents e skills existem para que cada etapa tenha um gate determinístico e evidências comprovadas.

**Princípio da Não-Interrupção:** Uma vez iniciada uma demanda, o orquestrador conduz as transições entre skills de ponta a ponta sem pausas protocolares para confirmação quando o status for `ok`. A intervenção humana só é solicitada em caso de ambiguidade crítica de negócio/design, falha bloqueante (`STATUS: bloqueado`) ou ao concluir com o Pull Request aberto.

**Toda demanda técnica começa na skill `pm-triage`.** O triage define o escopo e roteia para as próximas etapas.

## Fluxo de Execução

```text
pm-triage
  → architecture-guardrails [com REQUISITOS_SEGURANCA quando SECURITY: sim]
  → [data-model]
  → test-writer
  → dev-implementation
  → code-reviewer
  → [migration-reviewer] [query-performance]
  → security-review        (auditoria do diff vs requisitos do PRE)
  → supply-chain-review    (quando go.mod/go.sum mudarem)
  → qa-validation
  → [release-ops]
  → docs-update
  → git-delivery
```

Colchetes são módulos opcionais, ativados no campo `enabled_modules` em `.pipeline-ai.json`.

## Gates Obrigatórios da Toolchain

| Comando | Quando | Motivo |
|---|---|---|
| `gofmt -l .` | a cada edição (automático via hook `PostToolUse`) | Formatação determinística sem gastar turnos |
| `go build ./...` | dev | Validação de compilação do projeto |
| `go vet ./...` | dev e CI | Análise estática da stdlib |
| `go test ./...` | dev, QA e CI | Execução da suíte completa de testes |
| `govulncheck ./...` | supply-chain-review e CI | Detecção de símbolos vulneráveis invocados |
| `go test -fuzz=. -fuzztime=30s` | parsing, validação e desserialização | Fuzz sempre com `-fuzztime` explícito |

## Invariantes

1. **Go 1.24+**: A diretiva `tool` no `go.mod` não é aceita por versões anteriores.
2. **Testes com a stdlib `testing`**, table-driven. Testify só se o projeto optar explicitamente.
3. **Sem CGO** em projeto desktop para manter cross-compilação WSL2 -> Windows.
4. **Dependência externa precisa de justificativa**. A stdlib cobre a maior parte. Mudanças em `go.mod` disparam `supply-chain-review`.
5. **Versão do `wails3` fixada**. Betas mudam com frequência; o build não pode variar entre ambientes.
6. **Nada de segredos em código**. Credenciais sempre via variáveis de ambiente.

## Git & Pull Requests

- **Branch:** `task/{code}` (ou `feat/{nome}`)
- **Commit:** `{type}({scope}): {summary} [{task_code}]`
- **Submissão:** Pushes diretos na branch `main` são bloqueados pelo hook local `pre-push`. Toda entrega sobe via branch e Pull Request (`gh pr create`).

## Delivery

Toda entrega gera um relatório em `docs-ai/deliveries/{DELIVERY_ID}/report.md` e precisa estar registrada em `docs-ai/deliveries/INDEX.md`.

Antes do commit da entrega:

```bash
go tool pipeline-validate docs-ai/deliveries/{DELIVERY_ID}/report.md
```

O validador reprova a delivery se o diff tocar superfície de segurança (`security_surface_paths`) com `require_security` desligado, ou se o `git diff` falhar (a menos que seja passado `--no-git`).

## Estrutura de Customizações do Antigravity

```text
.agents/
├── hooks.json            # Hooks de ciclo de vida (PreToolUse e PostToolUse)
├── rules/                # Regras contextuais (invariants, handoff)
└── skills/               # Skills nativas do pipeline (pm-triage, dev-implementation, etc.)
```

## Onde está o resto

| Arquivo | Conteúdo |
|---|---|
| `docs-ai/DECISOES.md` | Por que o padrão é como é |
| `docs-ai/01-INVARIANTES-GLOBAIS.md` | Invariantes específicos deste projeto |
| `docs-ai/03-MAPA-RAPIDO-MODULOS.md` | Mapa detalhado dos módulos |
| `.pipeline-ai.json` | Caminhos protegidos, agents editores, superfície de segurança, módulos ativos |
