package metrics

import (
	"testing"
)

func FuzzParseMetrics(f *testing.F) {
	// Sementes iniciais com formatos válidos, vazios e corrompidos
	seeds := [][]byte{
		[]byte(`{}`),
		[]byte(`{"version":1}`),
		[]byte(`{"version":1,"events":[{"id":"1","type":"deployment","success":true}]}`),
		[]byte(`{"version":1,"deliveries":[{"delivery_id":"DEL-1","success":true,"lead_time":3600000000000}]}`),
		[]byte(`{"version":1,"stages":[{"stage":"sensor","duration":5000000,"success":true}]}`),
		[]byte(`[]`),
		[]byte(`""`),
		[]byte(`{"invalid": true`),
		[]byte(`null`),
		[]byte(`{"version": 99999999999999999999999999999999999999999999999999}`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// ParseMetrics nunca deve entrar em pânico para nenhuma entrada malformada
		_, _ = ParseMetrics(data)
	})
}
