// Package config carrega a configuração do pipeline a partir do arquivo
// .pipeline-ai.json na raiz do projeto.
//
// A configuração fica no projeto, e não no binário, porque o padrão suporta
// dois formatos de projeto (desktop e API+UI separadas) com estruturas de
// diretório diferentes. Um binário único atende aos dois (ver D-03 e D-11).
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileName é o nome do arquivo de configuração, procurado na raiz do projeto.
const FileName = ".pipeline-ai.json"

// ErrNotFound indica que nenhum arquivo de configuração foi encontrado a partir
// do diretório informado até a raiz do filesystem. Não é um erro fatal: projetos
// que não adotam o pipeline simplesmente não têm o arquivo.
var ErrNotFound = errors.New("config: " + FileName + " não encontrado")

// Config é o conteúdo de .pipeline-ai.json.
type Config struct {
	// Version permite evoluir o formato sem quebrar projetos antigos.
	Version int `json:"version"`

	// ProjectType é "desktop" (Go + Wails + React) ou "api" (API Go + UI React
	// separadas). Ver D-03.
	ProjectType string `json:"project_type"`

	// ProtectedPaths são os diretórios que a sessão principal não pode editar
	// diretamente. Caminhos relativos à raiz do projeto, com "/" como separador.
	// Um caminho terminado em "/" casa com o diretório e tudo abaixo dele.
	ProtectedPaths []string `json:"protected_paths"`

	// EditorAgents são os agents autorizados a editar código (D-11).
	EditorAgents []string `json:"editor_agents"`

	// SecuritySurfacePaths são os caminhos cuja alteração liga require_security
	// automaticamente, mesmo que o triage tenha esquecido (D-14).
	SecuritySurfacePaths []string `json:"security_surface_paths"`

	// DeliveriesDir é onde ficam os relatórios de entrega.
	DeliveriesDir string `json:"deliveries_dir"`

	// IndexFile é o índice onde toda delivery precisa estar registrada.
	IndexFile string `json:"index_file"`

	// BaseRef é a referência git usada para calcular o diff da delivery.
	BaseRef string `json:"base_ref"`

	// root é a raiz do projeto, preenchida no carregamento.
	root string
}

// Root devolve o diretório raiz do projeto (onde o arquivo foi encontrado).
func (c *Config) Root() string { return c.root }

// Find sobe a partir de start procurando FileName, até a raiz do filesystem.
// Devolve o caminho do arquivo encontrado.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("config: caminho inicial inválido: %w", err)
	}
	for {
		candidate := filepath.Join(dir, FileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Load procura e carrega a configuração a partir de start.
func Load(start string) (*Config, error) {
	path, err := Find(start)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: falha ao ler %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config: %s tem JSON inválido: %w", path, err)
	}
	c.root = filepath.Dir(path)
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("config: %s inválido: %w", path, err)
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
	if c.DeliveriesDir == "" {
		c.DeliveriesDir = "docs-ai/deliveries"
	}
	if c.IndexFile == "" {
		c.IndexFile = filepath.ToSlash(filepath.Join(c.DeliveriesDir, "INDEX.md"))
	}
	if c.BaseRef == "" {
		c.BaseRef = "origin/main"
	}
}

func (c *Config) validate() error {
	if c.Version != 1 {
		return fmt.Errorf("version %d não suportada por este binário (esperado 1)", c.Version)
	}
	switch c.ProjectType {
	case "desktop", "api":
	case "":
		return errors.New(`project_type é obrigatório ("desktop" ou "api")`)
	default:
		return fmt.Errorf(`project_type %q inválido (use "desktop" ou "api")`, c.ProjectType)
	}
	if len(c.ProtectedPaths) == 0 {
		return errors.New("protected_paths está vazio: o hook não protegeria nada")
	}
	return nil
}

// IsEditorAgent informa se agentType pode editar código.
// Um agentType vazio significa sessão principal, que nunca pode.
func (c *Config) IsEditorAgent(agentType string) bool {
	if agentType == "" {
		return false
	}
	for _, a := range c.EditorAgents {
		if strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(agentType)) {
			return true
		}
	}
	return false
}

// IsProtected informa se o caminho absoluto absPath cai dentro de algum
// protected_path. Caminhos fora da raiz do projeto nunca são protegidos.
func (c *Config) IsProtected(absPath string) bool {
	rel, ok := c.relToRoot(absPath)
	if !ok {
		return false
	}
	return matchesAny(rel, c.ProtectedPaths)
}

// IsSecuritySurface informa se o caminho relativo relPath toca superfície de
// segurança. Recebe caminho relativo porque a origem é `git diff --name-only`.
func (c *Config) IsSecuritySurface(relPath string) bool {
	return matchesAny(filepath.ToSlash(relPath), c.SecuritySurfacePaths)
}

// relToRoot converte um caminho absoluto em relativo à raiz do projeto.
// Devolve ok=false se o caminho estiver fora da raiz.
func (c *Config) relToRoot(absPath string) (string, bool) {
	clean, err := filepath.Abs(absPath)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(c.root, clean)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// matchesAny aplica as regras de casamento de padrão a rel.
//
// Regras:
//   - padrão terminado em "/" casa com o diretório e tudo abaixo
//   - padrão sem "/" final casa com o caminho exato ou como prefixo de diretório
func matchesAny(rel string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(filepath.ToSlash(p))
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(rel, p) {
				return true
			}
			continue
		}
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}
