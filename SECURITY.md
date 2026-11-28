# Política de Segurança (Security Policy)

## Versões Suportadas

Mantemos correções de segurança para a versão estável mais recente da linha `v0.x` e para a branch `main`:

| Versão | Suportada |
| --- | --- |
| `v0.12.x` / `main` | :white_check_mark: Sim |
| `< v0.12.0` | :x: Não |

## Como Reportar uma Vulnerabilidade

**Por favor, não abra issues públicas para relatar vulnerabilidades de segurança.**

Se você descobrir uma falha de segurança no `pipeline-ai` (por exemplo: vazamento de credenciais em logs/CLI/MCP, bypass de hooks `PreToolUse`, falhas de validação em `pipeline-validate` ou vulnerabilidades em workflows CI):

1. Utilize o recurso **Report a vulnerability** na aba **Security → Advisories** deste repositório no GitHub:
   `https://github.com/rodrigodosanjosoliveira/pipeline-ai/security/advisories/new`
2. Inclua na descrição:
   - Passos claros para reproduzir o problema;
   - Versão do `pipeline-ai` e ambiente (OS / versão do Go);
   - Impacto estimado e, se possível, sugestão de mitigação.

### Prazo de Resposta

- **Confirmação de recebimento:** em até 72 horas.
- **Triagem e avaliação:** em até 7 dias corridos.
- **Correção e divulgação coordenada:** assim que a correção for validada e publicada em release.

## Controles de Segurança Nativos do Projeto

O `pipeline-ai` adota práticas defensivas verificadas continuamente:

- **Zero dependências externas no `go.mod`:** 100% Go stdlib para minimizar superfície de supply chain.
- **Sensor `govulncheck` (`pipeline-ai sensor -vuln`):** auditoria de vulnerabilidades invocadas no grafo de chamadas.
- **Redação obrigatória de segredos (`SEC-03`):** chaves (`GEMINI_API_KEY`, `JEV_API_KEY`, `PIPELINE_SEMANTIC_API_KEY`) são enviadas exclusivamente via headers HTTP (nunca em URLs) e redigidas com `[REDACTED]` em mensagens de erro.
- **Superfície de segurança (`security_surface_paths`):** alterações em `go.mod`, `go.sum` ou `.github/workflows/` exigem auditoria formal (`require_security: true`) no `pipeline-validate`.
