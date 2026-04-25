package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asobitov2005/OpsDiff/internal/kube"
)

func RenderDoctor(checks []kube.Check, format string) (string, error) {
	switch strings.ToLower(format) {
	case "", "table", "text":
		return renderDoctorText(checks), nil
	case "json":
		data, err := json.MarshalIndent(checks, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal doctor report: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported doctor format %q", format)
	}
}

func renderDoctorText(checks []kube.Check) string {
	var builder strings.Builder
	for _, check := range checks {
		status := "✓"
		if !check.Passed {
			status = "✗"
		}
		builder.WriteString(fmt.Sprintf("%s %s", status, check.Name))
		if check.Detail != "" {
			builder.WriteString(": " + check.Detail)
		}
		builder.WriteByte('\n')
		if !check.Passed && check.FixHint != "" {
			builder.WriteString("  Fix: " + check.FixHint + "\n")
		}
	}
	return builder.String()
}
