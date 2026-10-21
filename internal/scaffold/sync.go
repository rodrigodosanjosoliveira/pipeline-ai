package scaffold

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Sync atualiza as skills, regras e hooks em .agents/ no projeto existente
// a partir dos templates embutidos mais recentes, preservando configurações e arquivos de negócio.
func Sync(targetDir string, stdout, stderr io.Writer) error {
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("caminho de destino inválido: %w", err)
	}

	agentsDir := filepath.Join(absTarget, ".agents")
	info, err := os.Stat(agentsDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("diretório .agents não encontrado em %s: este comando deve ser executado na raiz de um projeto pipeline-ai", absTarget)
	}

	fmt.Fprintf(stdout, "Sincronizando .agents a partir dos templates embutidos...\n")

	// 1. Sincronizar hooks.json
	hooksPath := filepath.Join(agentsDir, "hooks.json")
	if err := copyEmbeddedFile("templates/agents/hooks.json", hooksPath); err != nil {
		return fmt.Errorf("falha ao sincronizar hooks.json: %w", err)
	}
	fmt.Fprintf(stdout, "  ✓ .agents/hooks.json atualizado\n")

	// 2. Sincronizar regras oficiais em .agents/rules
	rulesDir := filepath.Join(agentsDir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return err
	}
	err = fs.WalkDir(templateFS, "templates/agents/rules", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel("templates/agents/rules", path)
		if err != nil {
			return err
		}
		target := filepath.Join(rulesDir, rel)
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "  ✓ .agents/rules/%s atualizado\n", rel)
		return nil
	})
	if err != nil {
		return fmt.Errorf("falha ao sincronizar rules: %w", err)
	}

	// 3. Sincronizar catálogo de skills em .agents/skills
	skillsDir := filepath.Join(agentsDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}
	err = fs.WalkDir(templateFS, "templates/agents/skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("templates/agents/skills", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(skillsDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		return fmt.Errorf("falha ao sincronizar skills: %w", err)
	}
	fmt.Fprintf(stdout, "  ✓ 16 skills em .agents/skills sincronizadas com sucesso\n")

	fmt.Fprintf(stdout, "\nSincronização concluída com sucesso!\n")
	return nil
}
