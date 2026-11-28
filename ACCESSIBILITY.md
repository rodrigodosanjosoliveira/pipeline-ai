# Declaração de Acessibilidade (Accessibility)

O projeto **pipeline-ai** tem o compromisso de manter suas interfaces — incluindo a CLI, o Dashboard HTML (`pipeline-ai dashboard`) e o [Guia Interativo Step-by-Step](https://rodrigodosanjosoliveira.github.io/pipeline-ai/) — acessíveis e utilizáveis pelo maior número possível de pessoas.

## Recursos de Acessibilidade Suportados

- **Navegação completa por teclado no Guia Interativo:**
  - `←` / `→`: navegar entre etapas;
  - `Space`: pausar/retomar animações dos diagramas SVG;
  - `R`: reiniciar a animação da etapa atual;
  - `⌘K` / `Ctrl+K`: busca global por lições e conceitos;
  - `Esc`: fechar modais e menus.
- **Temas Light e Dark de alto contraste:** alternância nativa no Guia Interativo e no Dashboard HTML.
- **Saída estruturada e limpa na CLI:** suporte a `-json`, `-agent`, detecção de `TTY` e mensagens de erro claras com `cause`, `hint` e `fix_command`.
- **Atributos semânticos:** uso de `role="img"`, `aria-label` e estrutura semântica HTML5 (`<main>`, `<aside>`, `<header>`).

## Limitações Conhecidas

- Os diagramas SVG animados do Guia Interativo ainda não respeitam automaticamente a preferência do sistema `prefers-reduced-motion` (mas podem ser pausados imediatamente com a tecla `Space` ou pelo botão **Pausar**).

## Como Reportar Barreiras de Acessibilidade

Se você encontrar qualquer barreira de acessibilidade na CLI, no Dashboard ou no Guia Interativo, abra uma issue em:
`https://github.com/rodrigodosanjosoliveira/pipeline-ai/issues/new`

Descreva a tela/comando afetado, a tecnologia assistiva ou navegador utilizado e o comportamento esperado.
