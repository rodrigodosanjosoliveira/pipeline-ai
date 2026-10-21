package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var delRegex = regexp.MustCompile(`\|\s*DEL-(\d+)\s*\|`)

// ScaffoldDelivery cria a estrutura de uma nova delivery e atualiza o INDEX.md.
func ScaffoldDelivery(rootDir, title, delivType, taskCode string) (string, string, error) {
	delivDir := filepath.Join(rootDir, "docs-ai", "deliveries")
	indexFile := filepath.Join(delivDir, "INDEX.md")

	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return "", "", fmt.Errorf("arquivo INDEX.md não encontrado em %s", indexFile)
	}

	indexBytes, err := os.ReadFile(indexFile)
	if err != nil {
		return "", "", fmt.Errorf("falha ao ler INDEX.md: %w", err)
	}

	indexContent := string(indexBytes)
	matches := delRegex.FindAllStringSubmatch(indexContent, -1)
	maxNum := 0
	for _, m := range matches {
		if len(m) > 1 {
			if num, err := strconv.Atoi(m[1]); err == nil && num > maxNum {
				maxNum = num
			}
		}
	}

	nextNum := maxNum + 1
	deliveryID := fmt.Sprintf("DEL-%03d", nextNum)
	if taskCode == "" {
		taskCode = fmt.Sprintf("TASK-%03d", nextNum)
	}
	if delivType == "" {
		delivType = "feat"
	}

	today := time.Now().Format("2006-01-02")
	targetFolder := filepath.Join(delivDir, deliveryID)
	if err := os.MkdirAll(targetFolder, 0755); err != nil {
		return "", "", fmt.Errorf("falha ao criar pasta da delivery: %w", err)
	}

	reportContent := fmt.Sprintf(`---
delivery_id: %s
task_code: %s
title: %s
type: %s
status: draft
require_security: false
require_migration: false
require_performance: false
date: %s
---

# Contexto

Descreva aqui o contexto e a motivação desta entrega.

# Escopo

- Defina aqui os itens dentro do escopo.

**Fora de escopo:**
- Defina aqui os itens explicitamente fora de escopo.

# Implementação

Detalhes técnicos da implementação.

# Testes

**Gates:**

`+"```"+`
$ gofmt -l .
$ go build ./...
$ go vet ./...
$ go test ./...
`+"```"+`

# Code Review

**Bloqueios:** Nenhum.

**Sugestões acatadas:** Nenhuma.

**Sugestões recusadas:** Nenhuma.

# Final Checklist

- [ ] `+"`gofmt -l .`"+` sem saída
- [ ] `+"`go build ./...`"+` limpo
- [ ] `+"`go vet ./...`"+` limpo
- [ ] `+"`go test ./...`"+` verde
- [ ] Escopo coberto por teste
- [ ] Nada do fora-de-escopo foi implementado
- [ ] Documentação dos módulos tocados atualizada
- [ ] Delivery registrada no INDEX.md
`, deliveryID, taskCode, title, delivType, today)

	reportPath := filepath.Join(targetFolder, "report.md")
	if err := os.WriteFile(reportPath, []byte(reportContent), 0644); err != nil {
		return "", "", fmt.Errorf("falha ao criar report.md: %w", err)
	}

	// Insere nova linha no INDEX.md logo após a linha separadora da tabela |---|
	newLine := fmt.Sprintf("| %s | %s | %s | %s | %s | draft |", deliveryID, taskCode, title, delivType, today)
	lines := strings.Split(indexContent, "\n")
	inserted := false
	var newLines []string

	for _, l := range lines {
		newLines = append(newLines, l)
		if !inserted && strings.HasPrefix(strings.TrimSpace(l), "|---") {
			newLines = append(newLines, newLine)
			inserted = true
		}
	}

	if !inserted {
		newLines = append(newLines, newLine)
	}

	if err := os.WriteFile(indexFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return "", "", fmt.Errorf("falha ao atualizar INDEX.md: %w", err)
	}

	relReportPath, _ := filepath.Rel(rootDir, reportPath)
	if relReportPath == "" {
		relReportPath = reportPath
	}
	return deliveryID, relReportPath, nil
}
