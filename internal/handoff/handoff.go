package handoff

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Signals representa os sinais de roteamento do handoff.
type Signals struct {
	Security    bool `json:"security"`
	SupplyChain bool `json:"supply_chain"`
	Migration   bool `json:"migration"`
	Performance bool `json:"performance"`
}

// Handoff representa os campos estruturados de um contrato de handoff entre agentes/skills.
type Handoff struct {
	De                  string   `json:"de"`
	Para                string   `json:"para"`
	DeliveryID          string   `json:"delivery_id"`
	TaskCode            string   `json:"task_code,omitempty"`
	Status              string   `json:"status"`
	Escopo              string   `json:"escopo"`
	Arquivos            []string `json:"arquivos"`
	RequisitosSeguranca string   `json:"requisitos_seguranca,omitempty"`
	Confirmado          string   `json:"confirmado"`
	Assuncoes           string   `json:"assuncoes"`
	Riscos              string   `json:"riscos"`
	Sinais              Signals  `json:"sinais"`
	Proximo             string   `json:"proximo"`
	Motivo              string   `json:"motivo"`
}

// ValidationResult representa o resultado da validação de integridade e regras do handoff.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Handoff  *Handoff `json:"handoff,omitempty"`
}

var recognizedKeys = map[string]bool{
	"DE":                   true,
	"PARA":                 true,
	"DELIVERY_ID":          true,
	"TASK_CODE":            true,
	"STATUS":               true,
	"ESCOPO":               true,
	"ARQUIVOS":             true,
	"REQUISITOS_SEGURANCA": true,
	"CONFIRMADO":           true,
	"ASSUNCOES":            true,
	"RISCOS":               true,
	"SINAIS":               true,
	"PROXIMO":              true,
	"MOTIVO":               true,
}

var evasivePhrases = map[string]bool{
	"ok":                   true,
	"tudo certo":           true,
	"tudo ok":              true,
	"testado":              true,
	"testada":              true,
	"testados":             true,
	"testadas":             true,
	"nenhum":               true,
	"nenhuma":              true,
	"verificado":           true,
	"verificada":           true,
	"verifiquei":           true,
	"verifiquei e esta ok": true,
	"verifiquei e está ok": true,
	"verificado e ok":      true,
	"verificado e está ok": true,
	"verificado e esta ok": true,
	"esta ok":              true,
	"está ok":              true,
	"passou":               true,
	"passaram":             true,
	"sucesso":              true,
	"feito":                true,
	"concluido":            true,
	"concluído":            true,
	"pronto":               true,
	"validei":              true,
	"validado":             true,
	"validada":             true,
	"nada a declarar":      true,
	"n/a":                  true,
	"nao se aplica":        true,
	"não se aplica":        true,
	"testes ok":            true,
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "sim", "true", "yes", "1", "s", "y":
		return true
	default:
		return false
	}
}

func parseInlineSignals(val string, sig *Signals) {
	parts := strings.Split(val, ",")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), ":", 2)
		if len(kv) == 2 {
			k := strings.ToUpper(strings.TrimSpace(kv[0]))
			v := parseBool(kv[1])
			switch k {
			case "SECURITY":
				sig.Security = v
			case "SUPPLY_CHAIN":
				sig.SupplyChain = v
			case "MIGRATION":
				sig.Migration = v
			case "PERFORMANCE":
				sig.Performance = v
			}
		}
	}
}

func cleanMarkdownFences(content string) string {
	s := strings.TrimSpace(content)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if strings.HasSuffix(s, "```") {
			s = strings.TrimSuffix(s, "```")
		}
		s = strings.TrimSpace(s)
		return s
	}

	if strings.Contains(s, "```") {
		start := strings.Index(s, "```")
		afterStart := s[start+3:]
		if firstNL := strings.Index(afterStart, "\n"); firstNL != -1 {
			body := afterStart[firstNL+1:]
			if end := strings.Index(body, "```"); end != -1 {
				inner := strings.TrimSpace(body[:end])
				if strings.Contains(inner, "DE:") || strings.Contains(inner, "STATUS:") {
					return inner
				}
			}
		}
	}

	return s
}

func isEvasiveEvidence(text string) bool {
	clean := strings.ToLower(strings.TrimSpace(text))
	clean = strings.Trim(clean, ".!?,;:- \t\r\n")

	if evasivePhrases[clean] {
		return true
	}

	if len(clean) < 10 {
		return true
	}

	hasCommandOrTool := strings.Contains(clean, "go ") ||
		strings.Contains(clean, "git ") ||
		strings.Contains(clean, "cat ") ||
		strings.Contains(clean, "grep ") ||
		strings.Contains(clean, "wc ") ||
		strings.Contains(clean, "curl ") ||
		strings.Contains(clean, "diff ") ||
		strings.Contains(clean, "make ") ||
		strings.Contains(clean, "gofmt") ||
		strings.Contains(clean, "govulncheck")

	hasPathOrFile := strings.Contains(clean, "/") ||
		strings.Contains(clean, "\\") ||
		strings.Contains(clean, ".go") ||
		strings.Contains(clean, ".md") ||
		strings.Contains(clean, ".json")

	hasExecutionArtifact := strings.Contains(text, "PASS") ||
		strings.Contains(text, "FAIL") ||
		strings.Contains(clean, "exit") ||
		strings.Contains(clean, "status") ||
		strings.Contains(clean, "teste") ||
		strings.Contains(clean, "linha") ||
		strings.Contains(clean, "tempo") ||
		strings.Contains(clean, "%")

	if !hasCommandOrTool && !hasPathOrFile {
		hasDigits := false
		for _, r := range clean {
			if unicode.IsDigit(r) {
				hasDigits = true
				break
			}
		}
		if !hasDigits || !hasExecutionArtifact {
			return true
		}
	}

	return false
}

// Parse extrai os campos estruturados de um handoff textual (puro ou envolto em cerca de código markdown).
func Parse(content string) (*Handoff, error) {
	clean := cleanMarkdownFences(content)
	if strings.TrimSpace(clean) == "" {
		return nil, errors.New("conteúdo de handoff vazio")
	}

	h := &Handoff{
		Arquivos: make([]string, 0),
	}

	lines := strings.Split(clean, "\n")
	currentField := ""
	foundAnyKey := false

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "```") {
			continue
		}

		if currentField == "SINAIS" || strings.HasPrefix(trimmedLine, "SECURITY:") || strings.HasPrefix(trimmedLine, "SUPPLY_CHAIN:") || strings.HasPrefix(trimmedLine, "MIGRATION:") || strings.HasPrefix(trimmedLine, "PERFORMANCE:") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				k := strings.ToUpper(strings.TrimSpace(parts[0]))
				v := strings.TrimSpace(parts[1])
				switch k {
				case "SECURITY":
					h.Sinais.Security = parseBool(v)
					foundAnyKey = true
					continue
				case "SUPPLY_CHAIN":
					h.Sinais.SupplyChain = parseBool(v)
					foundAnyKey = true
					continue
				case "MIGRATION":
					h.Sinais.Migration = parseBool(v)
					foundAnyKey = true
					continue
				case "PERFORMANCE":
					h.Sinais.Performance = parseBool(v)
					foundAnyKey = true
					continue
				}
			}
		}

		parts := strings.SplitN(trimmedLine, ":", 2)
		if len(parts) == 2 {
			k := strings.ToUpper(strings.TrimSpace(parts[0]))
			if recognizedKeys[k] {
				currentField = k
				foundAnyKey = true
				val := strings.TrimSpace(parts[1])

				switch k {
				case "DE":
					h.De = val
				case "PARA":
					h.Para = val
				case "DELIVERY_ID":
					h.DeliveryID = val
				case "TASK_CODE":
					h.TaskCode = val
				case "STATUS":
					h.Status = strings.ToLower(val)
				case "ESCOPO":
					h.Escopo = val
				case "ARQUIVOS":
					if val != "" {
						for _, item := range strings.Split(val, ",") {
							item = strings.TrimSpace(item)
							if item != "" {
								h.Arquivos = append(h.Arquivos, item)
							}
						}
					}
				case "REQUISITOS_SEGURANCA":
					h.RequisitosSeguranca = val
				case "CONFIRMADO":
					h.Confirmado = val
				case "ASSUNCOES":
					h.Assuncoes = val
				case "RISCOS":
					h.Riscos = val
				case "SINAIS":
					if val != "" {
						parseInlineSignals(val, &h.Sinais)
					}
				case "PROXIMO":
					h.Proximo = val
				case "MOTIVO":
					h.Motivo = val
				}
				continue
			}
		}

		if trimmedLine == "" {
			continue
		}

		switch currentField {
		case "ARQUIVOS":
			item := strings.TrimPrefix(trimmedLine, "-")
			item = strings.TrimPrefix(item, "*")
			item = strings.TrimSpace(item)
			if item != "" {
				h.Arquivos = append(h.Arquivos, item)
			}
		case "CONFIRMADO":
			if h.Confirmado == "" {
				h.Confirmado = trimmedLine
			} else {
				h.Confirmado += "\n" + trimmedLine
			}
		case "REQUISITOS_SEGURANCA":
			if h.RequisitosSeguranca == "" {
				h.RequisitosSeguranca = trimmedLine
			} else {
				h.RequisitosSeguranca += "\n" + trimmedLine
			}
		case "ASSUNCOES":
			if h.Assuncoes == "" {
				h.Assuncoes = trimmedLine
			} else {
				h.Assuncoes += "\n" + trimmedLine
			}
		case "RISCOS":
			if h.Riscos == "" {
				h.Riscos = trimmedLine
			} else {
				h.Riscos += "\n" + trimmedLine
			}
		case "ESCOPO":
			if h.Escopo == "" {
				h.Escopo = trimmedLine
			} else {
				h.Escopo += " " + trimmedLine
			}
		case "MOTIVO":
			if h.Motivo == "" {
				h.Motivo = trimmedLine
			} else {
				h.Motivo += " " + trimmedLine
			}
		}
	}

	if !foundAnyKey {
		return nil, errors.New("nenhuma chave de handoff reconhecida no conteúdo")
	}

	return h, nil
}

// Validate valida as regras e invariantes de integridade sobre um handoff parseado.
func Validate(h *Handoff) *ValidationResult {
	res := &ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
		Handoff:  h,
	}

	if h == nil {
		res.Valid = false
		res.Errors = append(res.Errors, "handoff nulo")
		return res
	}

	if strings.TrimSpace(h.De) == "" {
		res.Errors = append(res.Errors, "campo DE é obrigatório")
	}
	if strings.TrimSpace(h.Para) == "" {
		res.Errors = append(res.Errors, "campo PARA é obrigatório")
	}
	if strings.TrimSpace(h.DeliveryID) == "" {
		res.Errors = append(res.Errors, "campo DELIVERY_ID é obrigatório")
	}
	if strings.TrimSpace(h.Proximo) == "" {
		res.Errors = append(res.Errors, "campo PROXIMO é obrigatório")
	}
	if strings.TrimSpace(h.Motivo) == "" {
		res.Errors = append(res.Errors, "campo MOTIVO é obrigatório")
	}

	statusLower := strings.ToLower(strings.TrimSpace(h.Status))
	if statusLower == "" {
		res.Errors = append(res.Errors, "campo STATUS é obrigatório")
	} else if statusLower != "ok" && statusLower != "bloqueado" {
		res.Errors = append(res.Errors, fmt.Sprintf("campo STATUS inválido: %q (valores permitidos: 'ok' ou 'bloqueado')", h.Status))
	}

	trimmedConf := strings.TrimSpace(h.Confirmado)
	if trimmedConf == "" {
		res.Errors = append(res.Errors, "campo CONFIRMADO é obrigatório e deve conter evidência real dos comandos e gates executados")
	} else if isEvasiveEvidence(trimmedConf) {
		res.Errors = append(res.Errors, "evidência insuficiente em CONFIRMADO: respostas genéricas ou evasivas não são permitidas; forneça saídas reais de comandos, caminhos de arquivos ou testes executados")
	}

	if h.Sinais.Security && strings.TrimSpace(h.RequisitosSeguranca) == "" {
		res.Errors = append(res.Errors, "campo REQUISITOS_SEGURANCA é obrigatório quando o sinal SECURITY está ativado (SECURITY: sim)")
	}

	if len(res.Errors) > 0 {
		res.Valid = false
	}

	return res
}

// ValidateContent faz o parsing e validação de um conteúdo bruto de handoff.
func ValidateContent(content string) (*ValidationResult, error) {
	h, err := Parse(content)
	if err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("erro ao processar handoff: %v", err)},
		}, err
	}
	res := Validate(h)
	return res, nil
}
