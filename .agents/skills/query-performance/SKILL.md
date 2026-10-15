---
name: query-performance
description: >-
  Analisa performance de queries, índices e planos de execução, prevenindo varreduras completas e problemas de N+1.
  Use após code-reviewer ou migration-reviewer quando houver alterações em consultas ou persistência.
---

# Query Performance

## Missão

Garantir que todas as operações de banco de dados (SQL, BSON ou NoSQL) sejam eficientes, executem com uso adequado de índices, não causem varreduras completas de dados (*full table scans* / *collection scans*) e evitem problemas clássicos de latência como consultas N+1.

## Entrada

Handoff do `code-reviewer` ou `migration-reviewer`, com as queries e modelos alterados na camada de persistência (`internal/storage/` ou `migrations/`).

## Procedimento

1. **Auditoria de Consultas SQL (PostgreSQL):**
   - **Índices Cobridores:** Confirmar se colunas utilizadas em cláusulas `WHERE`, `JOIN` e `ORDER BY` possuem índices correspondentes.
   - **Índices Compostos:** Validar a regra de seletividade (colunas de igualdade primeiro, seguidas por faixas e ordenações).
   - **Prevenção de Invalidação de Índice:** Barrar funções aplicadas diretamente a colunas no `WHERE` (ex: `WHERE LOWER(email) = ...`) sem índice funcional correspondente.
   - **Projeção Estrita:** Evitar `SELECT *`. Sempre projetar as colunas estritamente necessárias para a struct Go receptora.

2. **Auditoria de Consultas MongoDB:**
   - Validar a regra **ESR** (*Equality, Sort, Range*) na definição de índices compostos.
   - Garantir que operações de leitura não resultem em `COLLSCAN` em coleções com crescimento contínuo de registros.

3. **Auditoria de Acesso no DynamoDB:**
   - Garantir que consultas utilizem a API `Query` com `KeyConditionExpression` baseada em `PK`, e **nunca** a operação `Scan` em fluxos transacionais.

4. **Detecção e Eliminação de N+1:**
   - Verificar loops em código Go (`for _, item := range items`) que executem consultas individuais de banco.
   - Exigir consultas em lote (`WHERE id IN (...)` ou `batch-get`) ou joins para hidratação de dados relacionados.

## O que NÃO fazer

- Não criar índices excessivos ou redundantes que degradem a performance de escrita (`INSERT`/`UPDATE`).
- Não validar apenas o comportamento funcional: focar em custo computacional e latência sob carga.
- Não reescrever código sem alinhamento com a arquitetura: apontar achados e devolver para `dev-implementation`.

## Roteamento

- `security-review` se houver alteração de superfície sensível ou credenciais.
- `supply-chain-review` se novas dependências de drivers foram adicionadas.
- Se nenhuma revisão intermediária for necessária, `qa-validation`.

## Saída obrigatória

O bloco do contrato de handoff, com:
- `ANALISE_PERFORMANCE`: aprovação das queries, uso de índices e ausência de N+1.
- `INDICES_RECOMENDADOS`: novos índices criados ou verificados para suportar as consultas.

