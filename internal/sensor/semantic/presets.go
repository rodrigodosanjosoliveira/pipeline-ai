package semantic

import (
	"errors"
	"strings"
)

// ErrInvalidPreset indica que o preset semântico solicitado não existe.
var ErrInvalidPreset = errors.New("invalid semantic preset")

const yagniQuestion = "Este código respeita estritamente o princípio YAGNI e minimalismo, sem overengineering, interfaces de implementação única desnecessárias, tipos duplicados do domínio ou abstrações especulativas não solicitadas?"

// ResolvePreset resolve um preset semântico nomeado (ex: "yagni", "minimalism", "ponytail")
// para o modo e a pergunta calibrada correspondente.
func ResolvePreset(name string) (mode string, question string, err error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "yagni", "minimalism", "ponytail":
		return "noul", yagniQuestion, nil
	default:
		return "", "", ErrInvalidPreset
	}
}
