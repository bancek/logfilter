package logfilter_test

import (
	"testing"

	. "github.com/bancek/logfilter/pkg/logfilter"
)

func BenchmarkJSONFilterIsIncluded(b *testing.B) {
	line := []byte(`{"Timestamp":"2020-08-18T17:16:36.9975268+00:00","Level":"Information","MessageTemplate":"Test message","Properties":{"DurationMs":1}}`)

	excludeTemplate := `{{with .Level}}{{eq . "Debug"}}{{end}}{{with .MessageTemplate}}{{eq . "Test message"}}{{end}}`

	jsonFilter, err := NewJSONFilter(excludeTemplate)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		jsonFilter.IsIncluded(line)
	}
}
