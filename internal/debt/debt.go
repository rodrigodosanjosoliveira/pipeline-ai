package debt

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrNotImplemented indica que a função ainda está em fase Red do TDD.
var ErrNotImplemented = errors.New("debt: não implementado")

// ShortcutItem representa um atalho deliberado rastreado por limite e gatilho de evolução.
type ShortcutItem struct {
	File           string `json:"file"`
	Line           int    `json:"line"`
	Limit          string `json:"limit"`
	UpgradeTrigger string `json:"upgrade_trigger"`
	Raw            string `json:"raw"`
}

// Summary agrega o inventário de atalhos deliberados de um diretório.
type Summary struct {
	Total int            `json:"total"`
	Items []ShortcutItem `json:"items"`
}

// ParseComment extrai o limite conhecido e o gatilho de evolução de comentários no formato:
// "// shortcut: <limite> | upgrade: <gatilho>" ou "// shortcut: <limite>, <gatilho>"
func ParseComment(line string) (limit string, trigger string, ok bool) {
	idx := strings.Index(line, "//")
	if idx == -1 {
		return "", "", false
	}

	comment := strings.TrimSpace(line[idx+2:])
	if len(comment) < len("shortcut:") {
		return "", "", false
	}

	if !strings.EqualFold(comment[:len("shortcut:")], "shortcut:") {
		return "", "", false
	}

	rest := strings.TrimSpace(comment[len("shortcut:"):])
	if rest == "" {
		return "", "", false
	}

	if pipeIdx := strings.Index(rest, "|"); pipeIdx != -1 {
		limit = strings.TrimSpace(rest[:pipeIdx])
		right := strings.TrimSpace(rest[pipeIdx+1:])
		if len(right) >= len("upgrade:") && strings.EqualFold(right[:len("upgrade:")], "upgrade:") {
			trigger = strings.TrimSpace(right[len("upgrade:"):])
		} else {
			trigger = right
		}
	} else if commaIdx := strings.Index(rest, ","); commaIdx != -1 {
		limit = strings.TrimSpace(rest[:commaIdx])
		right := strings.TrimSpace(rest[commaIdx+1:])
		if len(right) >= len("upgrade:") && strings.EqualFold(right[:len("upgrade:")], "upgrade:") {
			trigger = strings.TrimSpace(right[len("upgrade:"):])
		} else {
			trigger = right
		}
	} else {
		return "", "", false
	}

	if limit == "" || trigger == "" {
		return "", "", false
	}

	return limit, trigger, true
}

// ScanDir varre recursivamente os arquivos .go (exceto *_test.go e diretórios ocultos/.git/vendor/node_modules)
// a partir de rootDir e retorna o inventário determinístico de atalhos deliberados ordenado por File e Line.
func ScanDir(rootDir string) (*Summary, error) {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}

	items := make([]ShortcutItem, 0)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()
		if d.IsDir() {
			if path != rootDir {
				if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if limit, trigger, ok := ParseComment(line); ok {
				items = append(items, ShortcutItem{
					File:           relPath,
					Line:           lineNum,
					Limit:          limit,
					UpgradeTrigger: trigger,
					Raw:            strings.TrimSpace(line),
				})
			}
		}

		return scanner.Err()
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].File != items[j].File {
			return items[i].File < items[j].File
		}
		return items[i].Line < items[j].Line
	})

	return &Summary{
		Total: len(items),
		Items: items,
	}, nil
}
