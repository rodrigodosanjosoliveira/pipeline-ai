---
name: test-writer
description: >-
  Escreve os testes antes da implementação, incluindo alvos de fuzz quando houver parsing,
  validação ou desserialização. Use após architecture-guardrails, antes de dev-implementation.
---

# Test Writer

## Missão

Escrever os testes que definem o comportamento esperado, antes de existir implementação. Os testes são o gate contra o qual o dev trabalha.

Sem validação externa, a refatoração iterativa de um agent degrada, com erros se acumulando a cada passada. Estes testes fornecem essa validação determinística.

## Entrada

Handoff do `architecture-guardrails`, com `ABORDAGEM` e `NAO_FAZER` (e `REQUISITOS_SEGURANCA` quando `SECURITY: sim`).

## Padrão de teste

**Stdlib `testing`, table-driven.** Testify só se o projeto tiver optado explicitamente — verificar em `AGENTS.md` (ou `GEMINI.md`) antes de importar qualquer coisa.

```go
func TestX(t *testing.T) {
    tests := []struct {
        name    string
        in      string
        want    string
        wantErr bool
    }{
        {name: "caso feliz", in: "a", want: "A"},
        {name: "entrada vazia", in: "", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}
```

## Procedimento

1. Ler o código existente dos módulos no escopo.
2. Escrever os casos: caminho feliz, bordas, erro. Borda não é opcional.
3. Se houver `REQUISITOS_SEGURANCA` no handoff de entrada, obrigatoriamente escrever testes adversariais e negativos cobrindo cada requisito definido (injeção de payload, bypass de autorização, estouro de limite, negação de serviço, valores malformados).
4. Se a demanda envolver parsing, validação de entrada ou desserialização, escrever alvo de fuzz:
   ```go
   func FuzzParse(f *testing.F) {
       f.Add("entrada conhecida")
       f.Fuzz(func(t *testing.T, s string) { ... })
   }
   ```
5. Rodar `go test ./...` e confirmar que os novos testes **falham** pelo motivo certo. Teste que passa antes da implementação não testa nada.
6. Se houver alvo de fuzz, rodar com limite:
   `go test -fuzz=FuzzX -fuzztime=30s ./pacote`
   **Nunca sem `-fuzztime`.** Sem limite, o alvo roda indefinidamente.

## O que NÃO fazer

- Não implementar a funcionalidade para fazer o teste passar.
- Não escrever só o caminho feliz.
- Não ignorar `REQUISITOS_SEGURANCA` ou omitir testes adversariais quando houver requisitos de segurança.
- Não adicionar dependência de teste sem o projeto ter optado.
- Não rodar fuzz sem `-fuzztime`.
- Não tocar em arquivo fora do escopo definido.

## Roteamento

`dev-implementation`.

## Saída obrigatória

O bloco do contrato, com os testes escritos em `ARQUIVOS` e o resultado real do `go test` (a falha esperada) em `CONFIRMADO`.
