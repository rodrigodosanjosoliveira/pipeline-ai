package semantic_test

import (
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic"
)

// FuzzParseNoulResponse testa a robustez do parser contra entradas malformadas ou randômicas.
func FuzzParseNoulResponse(f *testing.F) {
	// Corpus inicial representativo
	f.Add([]byte(`{"decision": true, "confidence": 0.95, "reason": "OK"}`))
	f.Add([]byte(`{"decision": false, "confidence": 0.12, "reason": "Failed condition"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"decision": null, "confidence": "high"}`))
	f.Add([]byte(`{"nested": {"key": 123}}`))
	f.Add([]byte(`invalid json string`))
	f.Add([]byte(``))
	f.Add([]byte(`{"decision": true, "confidence": 1e309}`))
	f.Add([]byte(`{"decision": true, "confidence": -0.5, "reason": "\u0000\ufffd"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// O parser não deve entrar em pânico nem travar com qualquer sequência de bytes
		_, _ = semantic.ParseNoulResponse(data)
	})
}
