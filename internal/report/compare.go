package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func RenderCompare(result model.CompareResult, format string) (string, error) {
	switch strings.ToLower(format) {
	case "", "table", "text":
		return renderCompareText(result), nil
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal compare report: %w", err)
		}
		return string(data), nil
	case "markdown", "md":
		return renderCompareMarkdown(result), nil
	default:
		return "", fmt.Errorf("unsupported compare format %q", format)
	}
}

func renderCompareText(result model.CompareResult) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Namespace: %s\n", result.Namespace))
	builder.WriteString(fmt.Sprintf("Compared: %s -> %s\n\n", result.BeforePath, result.AfterPath))

	if len(result.Changes) == 0 {
		builder.WriteString("No changes detected.\n")
		return builder.String()
	}

	for _, change := range result.Changes {
		builder.WriteString(fmt.Sprintf("%-5s %s/%s\n", strings.ToUpper(string(change.Risk)), change.ResourceKind, change.ResourceName))
		builder.WriteString(fmt.Sprintf("      %s: %s -> %s\n", change.Path, emptyDash(change.Before), emptyDash(change.After)))
	}

	builder.WriteString(fmt.Sprintf("\nSummary: total=%d high=%d medium=%d low=%d\n", result.Summary.Total, result.Summary.High, result.Summary.Medium, result.Summary.Low))
	return builder.String()
}

func renderCompareMarkdown(result model.CompareResult) string {
	var builder strings.Builder
	builder.WriteString("# OpsDiff Compare Report\n\n")
	builder.WriteString(fmt.Sprintf("- Namespace: `%s`\n", result.Namespace))
	builder.WriteString(fmt.Sprintf("- Compared: `%s` -> `%s`\n", result.BeforePath, result.AfterPath))
	builder.WriteString(fmt.Sprintf("- Summary: `%d total`, `%d high`, `%d medium`, `%d low`\n\n", result.Summary.Total, result.Summary.High, result.Summary.Medium, result.Summary.Low))

	if len(result.Changes) == 0 {
		builder.WriteString("No changes detected.\n")
		return builder.String()
	}

	for _, change := range result.Changes {
		builder.WriteString(fmt.Sprintf("- **%s** `%s/%s` `%s`: `%s` -> `%s`\n", strings.ToUpper(string(change.Risk)), change.ResourceKind, change.ResourceName, change.Path, emptyDash(change.Before), emptyDash(change.After)))
	}

	return builder.String()
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
