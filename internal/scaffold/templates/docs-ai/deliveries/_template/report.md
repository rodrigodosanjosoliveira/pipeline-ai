---
delivery_id: {{DEL-NNN}}
task_code: {{TASK-NNN}}
title: {{resumo em uma linha, imperativo}}
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

{{Por que esta delivery existe. O problema, não a solução.}}

# Escopo

{{O que entra. Copiado do handoff do pm-triage.}}

**Fora de escopo:**
{{O que deliberadamente não entra. Copiado do mesmo handoff.}}

# Implementação

{{A abordagem tomada e por quê. Se divergiu do que a arquitetura definiu,
explique aqui.}}

**Arquivos alterados:**
- `{{caminho}}` — {{o que mudou}}

# Testes

**Gates:**

```
$ gofmt -l .
{{saída real}}

$ go build ./...
{{saída real}}

$ go vet ./...
{{saída real}}

$ go test ./...
{{saída real}}
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| {{item}} | `{{TestX}}` em `{{arquivo}}` |

# Code Review

**Bloqueios:** {{nenhum, ou a lista com [arquivo:linha]}}

**Sugestões acatadas:** {{o que foi ajustado}}

**Sugestões recusadas:** {{o que não foi, e por quê}}

# Rollback Plan

{{Obrigatório se require_rollback=true ou exigido pela política do projeto. Descreva passos de reversão, comandos operacionais e impacto.}}

# Final Checklist

- [ ] `gofmt -l .` sem saída
- [ ] `go build ./...` limpo
- [ ] `go vet ./...` limpo
- [ ] `go test ./...` verde
- [ ] Escopo coberto por teste
- [ ] Nada do fora-de-escopo foi implementado
- [ ] Documentação dos módulos tocados atualizada
- [ ] Delivery registrada no INDEX.md
