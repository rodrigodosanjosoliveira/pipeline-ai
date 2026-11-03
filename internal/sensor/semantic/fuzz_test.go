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

// FuzzParseChoiceResponse testa a robustez do parser de múltipla escolha contra entradas malformadas.
func FuzzParseChoiceResponse(f *testing.F) {
	f.Add([]byte(`{"selected": "feature", "confidence": 0.92, "options": {"feature": 0.92, "bugfix": 0.08}, "reason": "OK"}`))
	f.Add([]byte(`{"selected": "", "confidence": 0.0, "options": {}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"selected": null, "options": "invalid"}`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = semantic.ParseChoiceResponse(data)
	})
}

// FuzzParseScoreResponse testa a robustez do parser de score contínuo contra entradas malformadas.
func FuzzParseScoreResponse(f *testing.F) {
	f.Add([]byte(`{"score": 8.5, "max_score": 10.0, "reason": "Good quality"}`))
	f.Add([]byte(`{"score": -5.0, "max_score": 0.0}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"score": "high", "max_score": null}`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = semantic.ParseScoreResponse(data)
	})
}
