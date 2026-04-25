package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func RenderExplain(result model.ExplainResult, format string) (string, error) {
	switch strings.ToLower(format) {
	case "", "table", "text":
		return renderExplainText(result), nil
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal explain report: %w", err)
		}
		return string(data), nil
	case "markdown", "md":
		return renderExplainMarkdown(result), nil
	default:
		return "", fmt.Errorf("unsupported explain format %q", format)
	}
}

func renderExplainText(result model.ExplainResult) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Namespace: %s\n", result.Namespace))
	builder.WriteString(fmt.Sprintf("Snapshots: %s -> %s\n", result.BeforePath, result.AfterPath))
	builder.WriteString(fmt.Sprintf("Change window: %s -> %s\n", result.ChangeWindow.Start.Format(time.RFC3339), result.ChangeWindow.End.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("Summary: changes=%d ranked=%d critical_symptoms=%d warning_symptoms=%d evidence=%d\n\n",
		result.Summary.TotalChanges,
		result.Summary.RankedCandidates,
		result.Summary.CriticalSymptoms,
		result.Summary.WarningSymptoms,
		result.Summary.SupportingEvidence,
	))

	if len(result.Candidates) == 0 {
		builder.WriteString("No suspicious changes ranked.\n")
		return builder.String()
	}

	builder.WriteString("Likely causes:\n")
	for _, candidate := range result.Candidates {
		builder.WriteString(fmt.Sprintf("%d. [%d/100 %s] %s/%s %s: %s -> %s\n",
			candidate.Rank,
			candidate.Score,
			strings.ToUpper(candidate.Likelihood),
			candidate.Change.ResourceKind,
			candidate.Change.ResourceName,
			candidate.Change.Path,
			emptyDash(candidate.Change.Before),
			emptyDash(candidate.Change.After),
		))
		for _, evidence := range candidate.Evidence {
			builder.WriteString(fmt.Sprintf("   evidence: %s\n", evidence))
		}
		for _, event := range candidate.MatchedEvents {
			builder.WriteString(fmt.Sprintf("   event: [%s] %s %s (%+d)\n", event.Time.Format(time.RFC3339), event.Reason, event.Message, event.Contribution))
		}
		for _, check := range candidate.SuggestedChecks {
			builder.WriteString(fmt.Sprintf("   check: %s\n", check))
		}
	}

	return builder.String()
}

func renderExplainMarkdown(result model.ExplainResult) string {
	var builder strings.Builder
	builder.WriteString("# OpsDiff Explain Report\n\n")
	builder.WriteString(fmt.Sprintf("- Namespace: `%s`\n", result.Namespace))
	builder.WriteString(fmt.Sprintf("- Snapshots: `%s` -> `%s`\n", result.BeforePath, result.AfterPath))
	builder.WriteString(fmt.Sprintf("- Change window: `%s` -> `%s`\n", result.ChangeWindow.Start.Format(time.RFC3339), result.ChangeWindow.End.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Summary: `%d changes`, `%d ranked`, `%d critical symptoms`, `%d warning symptoms`\n\n",
		result.Summary.TotalChanges,
		result.Summary.RankedCandidates,
		result.Summary.CriticalSymptoms,
		result.Summary.WarningSymptoms,
	))

	if len(result.Candidates) == 0 {
		builder.WriteString("No suspicious changes ranked.\n")
		return builder.String()
	}

	for _, candidate := range result.Candidates {
		builder.WriteString(fmt.Sprintf("## %d. %s/%s `%s`\n\n", candidate.Rank, candidate.Change.ResourceKind, candidate.Change.ResourceName, candidate.Change.Path))
		builder.WriteString(fmt.Sprintf("- Score: `%d/100`\n", candidate.Score))
		builder.WriteString(fmt.Sprintf("- Likelihood: `%s`\n", candidate.Likelihood))
		builder.WriteString(fmt.Sprintf("- Change: `%s` -> `%s`\n", emptyDash(candidate.Change.Before), emptyDash(candidate.Change.After)))
		for _, evidence := range candidate.Evidence {
			builder.WriteString(fmt.Sprintf("- Evidence: %s\n", evidence))
		}
		for _, check := range candidate.SuggestedChecks {
			builder.WriteString(fmt.Sprintf("- Suggested check: %s\n", check))
		}
		builder.WriteByte('\n')
	}

	return builder.String()
}
