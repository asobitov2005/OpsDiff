package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func RenderTimeline(timeline model.Timeline, format string) (string, error) {
	switch strings.ToLower(format) {
	case "", "table", "text":
		return renderTimelineText(timeline), nil
	case "json":
		data, err := json.MarshalIndent(timeline, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal timeline report: %w", err)
		}
		return string(data), nil
	case "markdown", "md":
		return renderTimelineMarkdown(timeline), nil
	default:
		return "", fmt.Errorf("unsupported timeline format %q", format)
	}
}

func renderTimelineText(timeline model.Timeline) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Namespace: %s\n", timeline.Namespace))
	builder.WriteString(fmt.Sprintf("Window: %s -> %s\n", timeline.WindowStart.Format(time.RFC3339), timeline.WindowEnd.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("Summary: total=%d critical=%d warning=%d changes=%d symptoms=%d evidence=%d restarts=%d oomkills=%d crashloops=%d\n\n",
		timeline.Summary.Total,
		timeline.Summary.Critical,
		timeline.Summary.Warning,
		timeline.Summary.Changes,
		timeline.Summary.Symptoms,
		timeline.Summary.Evidence,
		timeline.Summary.Restarts,
		timeline.Summary.OOMKills,
		timeline.Summary.CrashLoops,
	))

	if len(timeline.Events) == 0 {
		builder.WriteString("No timeline events detected.\n")
		return builder.String()
	}

	for _, event := range timeline.Events {
		builder.WriteString(fmt.Sprintf("[%s] %-8s %-8s %s\n",
			event.Time.Format(time.RFC3339),
			strings.ToUpper(event.Severity),
			strings.ToUpper(event.Category),
			timelineHeadline(event),
		))
		builder.WriteString(fmt.Sprintf("           %s\n", event.Message))
	}

	return builder.String()
}

func renderTimelineMarkdown(timeline model.Timeline) string {
	var builder strings.Builder
	builder.WriteString("# OpsDiff Timeline Report\n\n")
	builder.WriteString(fmt.Sprintf("- Namespace: `%s`\n", timeline.Namespace))
	builder.WriteString(fmt.Sprintf("- Window: `%s` -> `%s`\n", timeline.WindowStart.Format(time.RFC3339), timeline.WindowEnd.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Summary: `%d total`, `%d critical`, `%d warning`, `%d changes`, `%d symptoms`, `%d evidence`\n\n",
		timeline.Summary.Total,
		timeline.Summary.Critical,
		timeline.Summary.Warning,
		timeline.Summary.Changes,
		timeline.Summary.Symptoms,
		timeline.Summary.Evidence,
	))

	if len(timeline.Events) == 0 {
		builder.WriteString("No timeline events detected.\n")
		return builder.String()
	}

	for _, event := range timeline.Events {
		builder.WriteString(fmt.Sprintf("- `%s` **%s** `%s` %s\n",
			event.Time.Format(time.RFC3339),
			strings.ToUpper(event.Severity),
			event.Category,
			timelineHeadline(event),
		))
		builder.WriteString(fmt.Sprintf("  %s\n", event.Message))
	}

	return builder.String()
}

func timelineHeadline(event model.TimelineEvent) string {
	parts := make([]string, 0, 5)
	if event.Service != "" {
		parts = append(parts, "service="+event.Service)
	}
	if event.ResourceKind != "" || event.ResourceName != "" {
		parts = append(parts, strings.Trim(event.ResourceKind+"/"+event.ResourceName, "/"))
	}
	if event.Reason != "" {
		parts = append(parts, event.Reason)
	}
	if len(parts) == 0 {
		return event.Source
	}
	return strings.Join(parts, " ")
}
