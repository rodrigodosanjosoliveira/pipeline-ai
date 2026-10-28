package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Manifest representa o manifesto do plugin (plugin.json) de acordo com a especificação do Antigravity.
type Manifest struct {
	Name             string   `json:"name"`
	DisplayName      string   `json:"displayName,omitempty"`
	Version          string   `json:"version,omitempty"`
	Description      string   `json:"description,omitempty"`
	Logo             string   `json:"logo,omitempty"`
	SuggestedPrompts []string `json:"suggestedPrompts,omitempty"`
	Author           string   `json:"author,omitempty"`
	Repository       string   `json:"repository,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	Disabled         bool     `json:"disabled,omitempty"`
}

// BundleReport consolida o diagnóstico de verificação do bundle.
type BundleReport struct {
	Dir      string    `json:"dir"`
	Manifest *Manifest `json:"manifest"`
	Skills   []string  `json:"skills"`
}

// ValidateManifest valida o conteúdo e campos obrigatórios do manifesto plugin.json.
func ValidateManifest(data []byte) (*Manifest, error) {
	if len(data) == 0 {
		return nil, errors.New("payload vazio")
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("json inválido: %w", err)
	}

	if strings.TrimSpace(m.Name) == "" {
		return nil, errors.New("campo 'name' é obrigatório")
	}
	if strings.TrimSpace(m.Description) == "" {
		return nil, errors.New("campo 'description' é obrigatório")
	}
	if strings.TrimSpace(m.DisplayName) == "" {
		return nil, errors.New("campo 'displayName' é obrigatório")
	}
	if strings.TrimSpace(m.Version) == "" {
		return nil, errors.New("campo 'version' é obrigatório")
	}
	if strings.TrimSpace(m.Repository) == "" {
		return nil, errors.New("campo 'repository' é obrigatório")
	}
	if len(m.SuggestedPrompts) > 3 {
		return nil, errors.New("o campo 'suggestedPrompts' não pode exceder 3 itens")
	}

	return &m, nil
}

// VerifyBundle verifica a estrutura do diretório do bundle do plugin.
func VerifyBundle(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("diretório do bundle inválido ou inexistente: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s não é um diretório", dir)
	}

	manifestPath := filepath.Join(dir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("arquivo obrigatório plugin.json não encontrado: %w", err)
	}
	if _, err := ValidateManifest(data); err != nil {
		return fmt.Errorf("plugin.json inválido: %w", err)
	}

	rulesPath := filepath.Join(dir, "rules", "AGENTS.md")
	if _, err := os.Stat(rulesPath); err != nil {
		return fmt.Errorf("arquivo obrigatório rules/AGENTS.md não encontrado: %w", err)
	}

	mcpPath := filepath.Join(dir, "mcp_config.json")
	if _, err := os.Stat(mcpPath); err != nil {
		return fmt.Errorf("arquivo obrigatório mcp_config.json não encontrado: %w", err)
	}

	hooksPath := filepath.Join(dir, "hooks.json")
	if _, err := os.Stat(hooksPath); err != nil {
		return fmt.Errorf("arquivo obrigatório hooks.json não encontrado: %w", err)
	}

	skillsDir := filepath.Join(dir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("diretório de catálogo skills/ não encontrado: %w", err)
	}
	if len(entries) == 0 {
		return errors.New("o catálogo de skills está vazio")
	}

	return nil
}

// InstallToDir instala os arquivos do bundle no diretório de destino de forma idempotente.
func InstallToDir(srcDir, targetDir string) error {
	if err := VerifyBundle(srcDir); err != nil {
		return fmt.Errorf("não é possível instalar bundle inválido: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório de destino: %w", err)
	}

	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		dest := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		// Copia arquivo de forma atômica/segura
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}
