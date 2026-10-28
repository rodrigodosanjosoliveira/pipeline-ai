# Pipeline-AI Plugin para Antigravity

Plugin oficial do **Pipeline-AI** para integração nativa com o ecossistema Antigravity / Gemini CLI.

## O que está incluso

- **Manifesto Antigravity**: Configuração formal e prompts sugeridos em `plugin.json`.
- **Servidor MCP**: Integração com MCP stdio (`pipeline-ai mcp`) exposta via `mcp_config.json`.
- **Ciclo de Vida e Hooks**: `hooks.json` com `PreToolUse` (bloqueio determinístico de edição direta) e `PostToolUse` (`gofmt`).
- **16 Skills Especializadas**: Catálogo completo com instruções especializadas cobrindo triage, arquitetura, testes, implementação, revisão, segurança e delivery.
- **Regras Consolidadas**: Invariantes, governança e contratos de handoff em `rules/AGENTS.md`.

## Como Instalar

### Via CLI do Pipeline-AI

```bash
pipeline-ai plugin install
```

Por padrão, os arquivos do plugin serão instalados no diretório de plugins do usuário (`~/.gemini/config/plugins/pipeline-ai` ou destino especificado com `-target`).

### Verificação de Integridade

Para validar a integridade de um bundle antes da instalação:

```bash
pipeline-ai plugin verify -dir ./plugin
```

## Estrutura do Bundle

```text
plugin/
├── plugin.json       # Manifesto oficial do plugin
├── mcp_config.json   # Configuração do servidor MCP
├── hooks.json        # Hooks PreToolUse e PostToolUse
├── rules/
│   └── AGENTS.md     # Governança e regras consolidadas
├── skills/           # 16 skills especializadas
├── assets/
│   └── logo.svg      # Logo vetorial do projeto
└── README.md         # Esta documentação
```
