package observability

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestRequestLoggerEmitsRequiredJSONFields(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	RequestLogger{Logger: logger}.ObserveRequest("id", "api", "GET", "/items", "one", 200, 2, "miss", 25*time.Millisecond)
	for _, field := range []string{`"request_id":"id"`, `"route":"api"`, `"backend":"one"`, `"status":200`, `"cache":"miss"`} {
		if !strings.Contains(output.String(), field) {
			t.Fatalf("log missing %s: %s", field, output.String())
		}
	}
}
