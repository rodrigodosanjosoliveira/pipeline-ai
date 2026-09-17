// Package report faz o parsing e a validação do relatório de delivery.
//
// O front matter é lido por um parser próprio, restrito a "chave: valor", para
// manter o módulo sem dependências externas (ver D-17 e D-19).
package report

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// Report é um relatório de delivery já parseado.
type Report struct {
	Front    map[string]string
	Sections map[string]string // título normalizado -> corpo
	Order    []string          // títulos na ordem em que aparecem
	Body     string
}

// Finding é um problema encontrado na validação.
type Finding struct {
	Level   string // "ERRO" ou "AVISO"
	Message string
}

func (f Finding) String() string { return f.Level + ": " + f.Message }

// Marcadores que não podem sobreviver a uma entrega.
//
// A detecção é por palavra inteira, não por substring. A primeira versão usava
// strings.Contains e reprovava qualquer relatório em português, porque "todos"
// contém "todo" e "independente" contém "pendente".
//
// Convenções de código (TODO, FIXME) casam só em caixa alta: é como aparecem em
// comentários, e exigir caixa alta elimina o falso positivo com a palavra
// portuguesa "todo".
var codeMarkerRE = regexp.MustCompile(`\b(TODO|TBD|FIXME|XXX|HACK|WIP)\b`)

// Frases de preenchimento, case-insensitive e por palavra inteira.
var fillerRE = regexp.MustCompile(`(?i)\b(a definir|a preencher|preencher aqui|lorem ipsum|descreva aqui)\b`)

// requiredFront são os campos de front matter obrigatórios em toda delivery.
var requiredFront = []string{"delivery_id", "task_code", "title", "status"}

// alwaysSections são as seções obrigatórias, independentemente das flags.
var alwaysSections = []string{
	"contexto",
	"escopo",
	"implementacao",
	"testes",
	"code review",
	"final checklist",
}

// conditionalSections mapeia a flag de front matter para a seção que ela exige.
var conditionalSections = map[string]string{
	"require_security":    "security",
	"require_migration":   "migration",
	"require_performance": "performance",
}

// Parse lê o conteúdo bruto de um relatório.
func Parse(content string) (*Report, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")

	front, body, err := splitFrontMatter(content)
	if err != nil {
		return nil, err
	}

	r := &Report{
		Front:    front,
		Sections: map[string]string{},
		Body:     body,
	}
	r.parseSections(body)
	return r, nil
}

func splitFrontMatter(content string) (map[string]string, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", fmt.Errorf("relatório não começa com front matter (---)")
	}
	rest := content[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", fmt.Errorf("front matter não foi fechado com ---")
	}
	raw := rest[:end]
	body := rest[end+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")

	front := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		front[key] = value
	}
	if err := sc.Err(); err != nil {
		return nil, "", fmt.Errorf("falha ao ler front matter: %w", err)
	}
	return front, body, nil
}

func (r *Report) parseSections(body string) {
	var current string
	var buf strings.Builder

	flush := func() {
		if current != "" {
			r.Sections[current] = strings.TrimSpace(buf.String())
		}
		buf.Reset()
	}

	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			flush()
			current = normalize(strings.TrimLeft(line, "# "))
			r.Order = append(r.Order, current)
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()
}

// normalize deixa o título comparável: minúsculas, sem acento e sem pontuação.
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u", "ü", "u",
		"ç", "c",
		":", "", ".", "",
	)
	return strings.TrimSpace(replacer.Replace(s))
}

// Flag lê uma flag booleana do front matter. Ausente vale false (ver D-14).
func (r *Report) Flag(name string) bool {
	v, ok := r.Front[strings.ToLower(name)]
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "yes", "sim", "1":
		return true
	default:
		return false
	}
}

// Validate roda todas as checagens internas ao relatório.
// A checagem cruzada com o diff do git fica no comando, que tem acesso ao repo.
func (r *Report) Validate() []Finding {
	var out []Finding

	out = append(out, r.checkFrontMatter()...)
	out = append(out, r.checkSections()...)
	out = append(out, r.checkPlaceholders()...)
	out = append(out, r.checkChecklist()...)

	return out
}

func (r *Report) checkFrontMatter() []Finding {
	var out []Finding
	for _, key := range requiredFront {
		v, ok := r.Front[key]
		if !ok {
			out = append(out, Finding{"ERRO", "front matter: campo obrigatório ausente: " + key})
			continue
		}
		if strings.TrimSpace(v) == "" {
			out = append(out, Finding{"ERRO", "front matter: campo vazio: " + key})
		}
	}
	return out
}

func (r *Report) checkSections() []Finding {
	var out []Finding

	for _, want := range alwaysSections {
		body, ok := r.Sections[want]
		if !ok {
			out = append(out, Finding{"ERRO", "seção obrigatória ausente: " + want})
			continue
		}
		if strings.TrimSpace(body) == "" {
			out = append(out, Finding{"ERRO", "seção vazia: " + want})
		}
	}

	for flag, section := range conditionalSections {
		if !r.Flag(flag) {
			continue
		}
		body, ok := r.Sections[section]
		if !ok {
			out = append(out, Finding{"ERRO",
				fmt.Sprintf("%s=true exige a seção %q, que não existe", flag, section)})
			continue
		}
		trimmed := strings.TrimSpace(body)
		if trimmed == "" {
			out = append(out, Finding{"ERRO",
				fmt.Sprintf("%s=true mas a seção %q está vazia", flag, section)})
			continue
		}
		if isNA(trimmed) {
			out = append(out, Finding{"ERRO",
				fmt.Sprintf("%s=true mas a seção %q está marcada como N/A", flag, section)})
		}
	}

	return out
}

func isNA(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "n/a", "na", "não se aplica", "nao se aplica", "-":
		return true
	}
	return false
}

func (r *Report) checkPlaceholders() []Finding {
	var out []Finding

	for _, re := range []*regexp.Regexp{codeMarkerRE, fillerRE} {
		for _, loc := range re.FindAllStringIndex(r.Body, -1) {
			out = append(out, Finding{"ERRO",
				fmt.Sprintf("marcador %q encontrado no relatório (linha %d)",
					r.Body[loc[0]:loc[1]], lineOf(r.Body, loc[0]))})
		}
	}

	for key, value := range r.Front {
		for _, re := range []*regexp.Regexp{codeMarkerRE, fillerRE} {
			if m := re.FindString(value); m != "" {
				out = append(out, Finding{"ERRO",
					fmt.Sprintf("front matter: campo %q contém o marcador %q", key, m)})
				break
			}
		}
	}

	return out
}

func lineOf(s string, idx int) int {
	if idx < 0 || idx > len(s) {
		return 0
	}
	return strings.Count(s[:idx], "\n") + 1
}

func (r *Report) checkChecklist() []Finding {
	var out []Finding

	body, ok := r.Sections["final checklist"]
	if !ok {
		return out // já reportado em checkSections
	}

	var total, done int
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "- [ ]"):
			total++
			out = append(out, Finding{"ERRO", "checklist com item não concluído: " + strings.TrimSpace(line[5:])})
		case strings.HasPrefix(line, "- [x]"), strings.HasPrefix(line, "- [X]"):
			total++
			done++
		}
	}

	if total == 0 {
		out = append(out, Finding{"ERRO", "Final Checklist não contém nenhum item"})
	}
	_ = done

	return out
}
