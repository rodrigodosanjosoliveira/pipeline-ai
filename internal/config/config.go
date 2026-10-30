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
	"regexp"
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

	// ProjectType é "desktop" (Go + Wails + React), "api" (API Go + UI React
	// separadas) ou "cli" (linha de comando / terminal). Ver D-03.
	ProjectType string `json:"project_type"`

	// ProtectedPaths são os diretórios que a sessão principal não pode editar
	// diretamente. Caminhos relativos à raiz do projeto, com "/" como separador.
	// Um caminho terminado em "/" casa com o diretório e tudo abaixo dele.
	ProtectedPaths []string `json:"protected_paths"`

	// ImmutablePaths são os caminhos estritamente imutáveis que nenhum agente
	// ou delivery ativa pode alterar diretamente.
	ImmutablePaths []string `json:"immutable_paths"`

	// EditorAgents são os agents autorizados a editar código (D-11).
	EditorAgents []string `json:"editor_agents"`

	// SecuritySurfacePaths são os caminhos cuja alteração liga require_security
	// automaticamente, mesmo que o triage tenha esquecido (D-14).
	SecuritySurfacePaths []string `json:"security_surface_paths"`

	// EnabledModules são os módulos opcionais ativados pelo projeto (D-15).
	EnabledModules []string `json:"enabled_modules"`

	// DeliveriesDir é onde ficam os relatórios de entrega.
	DeliveriesDir string `json:"deliveries_dir"`

	// IndexFile é o índice onde toda delivery precisa estar registrada.
	IndexFile string `json:"index_file"`

	// BaseRef é a referência git usada para calcular o diff da delivery.
	BaseRef string `json:"base_ref"`

	// DeliveryIDPattern é a regex para validar identificadores de delivery.
	DeliveryIDPattern string `json:"delivery_id_pattern"`

	// TaskCodePattern é a regex para validar códigos de tarefas vinculadas.
	TaskCodePattern string `json:"task_code_pattern"`

	// CustomSections são seções obrigatórias adicionais no relatório de delivery.
	CustomSections []string `json:"custom_sections"`

	// RequireIndex define se o arquivo INDEX.md é obrigatório. Padrão: true.
	RequireIndex *bool `json:"require_index"`

	// RequireRollback indica se a política de rollback é mandatória em deliveries (C16).
	RequireRollback *bool `json:"require_rollback,omitempty"`

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

// AllowedModules lista os módulos opcionais válidos reconhecidos pelo padrão (D-15).
var AllowedModules = map[string]bool{
	"data-model":         true,
	"migration-reviewer": true,
	"query-performance":  true,
	"release-ops":        true,
	"agent-auditor":      true,
	"context-loader":     true,
}

func (c *Config) validate() error {
	if c.Version != 1 {
		return fmt.Errorf("version %d não suportada por este binário (esperado 1)", c.Version)
	}
	switch c.ProjectType {
	case "desktop", "api", "cli":
	case "":
		return errors.New(`project_type é obrigatório ("desktop", "api" ou "cli")`)
	default:
		return fmt.Errorf(`project_type %q inválido (use "desktop", "api" ou "cli")`, c.ProjectType)
	}
	if len(c.ProtectedPaths) == 0 {
		return errors.New("protected_paths está vazio: o hook não protegeria nada")
	}
	for _, m := range c.EnabledModules {
		mod := strings.ToLower(strings.TrimSpace(m))
		if !AllowedModules[mod] {
			return fmt.Errorf("módulo %q inválido em enabled_modules", m)
		}
	}
	if c.DeliveryIDPattern != "" {
		if _, err := regexp.Compile(c.DeliveryIDPattern); err != nil {
			return fmt.Errorf("delivery_id_pattern %q inválido: %w", c.DeliveryIDPattern, err)
		}
	}
	if c.TaskCodePattern != "" {
		if _, err := regexp.Compile(c.TaskCodePattern); err != nil {
			return fmt.Errorf("task_code_pattern %q inválido: %w", c.TaskCodePattern, err)
		}
	}
	return nil
}

// Padrões default para identificadores de delivery e códigos de tarefa.
const (
	DefaultDeliveryIDPattern = `^DEL-[0-9]{3,}$`
	DefaultTaskCodePattern   = `^[A-Z0-9]+-[0-9]+$`
)

// IsIndexRequired informa se o arquivo INDEX.md é obrigatório para registrar deliveries.
// Se omitido na configuração (nil), o padrão é true.
func (c *Config) IsIndexRequired() bool {
	if c == nil || c.RequireIndex == nil {
		return true
	}
	return *c.RequireIndex
}

// IsRollbackRequired informa se o plano de rollback é mandatório em relatórios de delivery (C16).
// Se omitido na configuração (nil), o padrão é false.
func (c *Config) IsRollbackRequired() bool {
	if c == nil || c.RequireRollback == nil {
		return false
	}
	return *c.RequireRollback
}

// EnableModule adiciona um módulo opcional a EnabledModules de forma idempotente e valida a configuração.
func (c *Config) EnableModule(mod string) error {
	if c == nil {
		return errors.New("config: config nula")
	}
	normalized := strings.ToLower(strings.TrimSpace(mod))
	if normalized == "" || !AllowedModules[normalized] {
		return fmt.Errorf("config: módulo inválido ou desconhecido: %q", mod)
	}
	if c.IsModuleEnabled(normalized) {
		return nil
	}
	c.EnabledModules = append(c.EnabledModules, normalized)
	if err := c.validate(); err != nil {
		return err
	}
	return nil
}

// DisableModule remove um módulo de EnabledModules de forma idempotente.
func (c *Config) DisableModule(mod string) error {
	if c == nil {
		return errors.New("config: config nula")
	}
	normalized := strings.ToLower(strings.TrimSpace(mod))
	if normalized == "" {
		return nil
	}
	filtered := make([]string, 0, len(c.EnabledModules))
	for _, m := range c.EnabledModules {
		if !strings.EqualFold(strings.TrimSpace(m), normalized) {
			filtered = append(filtered, m)
		}
	}
	c.EnabledModules = filtered
	return nil
}

// Save serializa a configuração em JSON formatado e grava no arquivo especificado ou no arquivo raiz padrão.
func (c *Config) Save(targetPath ...string) error {
	if c == nil {
		return errors.New("config: config nula")
	}
	var dest string
	if len(targetPath) > 0 && targetPath[0] != "" {
		dest = targetPath[0]
	} else {
		if c.root == "" {
			return errors.New("config: root não definido para salvar configuração")
		}
		dest = filepath.Join(c.root, FileName)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: falha ao serializar json: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return fmt.Errorf("config: falha ao escrever %s: %w", dest, err)
	}
	return nil
}

// ValidateDeliveryID valida se o deliveryID atende ao padrão configurado (ou ao padrão default).
func (c *Config) ValidateDeliveryID(id string) error {
	pattern := DefaultDeliveryIDPattern
	if c != nil && c.DeliveryIDPattern != "" {
		pattern = c.DeliveryIDPattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("delivery_id_pattern %q inválido: %w", pattern, err)
	}
	if !re.MatchString(id) {
		return fmt.Errorf("delivery_id %q inválido (não atende ao padrão %q)", id, pattern)
	}
	return nil
}

// ValidateTaskCode valida se o taskCode atende ao padrão configurado (ou ao padrão default).
func (c *Config) ValidateTaskCode(code string) error {
	pattern := DefaultTaskCodePattern
	if c != nil && c.TaskCodePattern != "" {
		pattern = c.TaskCodePattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("task_code_pattern %q inválido: %w", pattern, err)
	}
	if !re.MatchString(code) {
		return fmt.Errorf("task_code %q inválido (não atende ao padrão %q)", code, pattern)
	}
	return nil
}

// IsModuleEnabled informa se o módulo opcional name está habilitado no projeto (D-15).
func (c *Config) IsModuleEnabled(name string) bool {
	if name == "" {
		return false
	}
	for _, m := range c.EnabledModules {
		if strings.EqualFold(strings.TrimSpace(m), strings.TrimSpace(name)) {
			return true
		}
	}
	return false
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

// IsImmutable informa se o caminho absoluto absPath cai dentro de algum
// immutable_path. Caminhos fora da raiz do projeto nunca são imutáveis.
func (c *Config) IsImmutable(absPath string) bool {
	rel, ok := c.relToRoot(absPath)
	if !ok {
		return false
	}
	return matchesAny(rel, c.ImmutablePaths)
}

// IsSecuritySurface informa se o caminho relativo relPath toca superfície de
// segurança. Recebe caminho relativo porque a origem é `git diff --name-only`.
func (c *Config) IsSecuritySurface(relPath string) bool {
	return matchesAny(filepath.ToSlash(relPath), c.SecuritySurfacePaths)
}

// relToRoot converte um caminho absoluto em relativo à raiz do projeto.
// Devolve ok=false se o caminho estiver fora da raiz.
func (c *Config) relToRoot(absPath string) (string, bool) {
	canonicalRoot := resolveSymlinkAncestors(c.root)

	clean, err := filepath.Abs(absPath)
	if err != nil {
		return "", false
	}
	clean = resolveSymlinkAncestors(clean)

	rel, err := filepath.Rel(canonicalRoot, clean)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// resolveSymlinkAncestors resolve symlinks no caminho absPath ou no ancestral
// existente mais próximo quando subdiretórios/arquivos finais ainda não existem.
func resolveSymlinkAncestors(absPath string) string {
	current := filepath.Clean(absPath)
	var tail []string
	for {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(absPath)
		}
		tail = append(tail, filepath.Base(current))
		current = parent
	}
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
