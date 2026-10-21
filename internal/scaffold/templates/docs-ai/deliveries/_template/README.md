# Template de relatório de delivery

Copie `report.md` para `docs-ai/deliveries/{DELIVERY_ID}/report.md` e preencha.

## Seções condicionais

O template traz as três flags desligadas. Quando uma é ligada, a seção
correspondente passa a ser obrigatória e **não pode** ser preenchida com `N/A`:

| Flag | Seção a acrescentar |
|---|---|
| `require_security: true` | `# Security` |
| `require_migration: true` | `# Migration` |
| `require_performance: true` | `# Performance` |

`require_security` não depende do triage lembrar: o validador liga a exigência
sozinho quando o diff toca `security_surface_paths` de `.pipeline-ai.json`.

## O que o validador reprova

- Campo obrigatório de front matter vazio ou ausente
- Seção obrigatória ausente ou vazia
- Seção condicional marcada como `N/A` com a flag ligada
- Item de checklist não concluído
- Delivery não registrada no INDEX.md
- Marcadores `TODO`, `TBD`, `FIXME`, `XXX`, `HACK`, `WIP` em caixa alta
- Frases de preenchimento: "a definir", "a preencher", "lorem ipsum"
- Diff em superfície de segurança com `require_security` desligado

Os marcadores de código casam **só em caixa alta e por palavra inteira**. Texto
português legítimo como "todos os testes" ou "módulo independente" passa.

## Uso

```bash
go tool pipeline-validate docs-ai/deliveries/{DELIVERY_ID}/report.md
```
