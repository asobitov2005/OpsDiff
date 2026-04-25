package app

import (
	"context"
	"fmt"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/argocd"
	"github.com/asobitov2005/OpsDiff/internal/explain"
	"github.com/asobitov2005/OpsDiff/internal/kube"
	"github.com/asobitov2005/OpsDiff/internal/model"
	"github.com/asobitov2005/OpsDiff/internal/prometheus"
	"github.com/asobitov2005/OpsDiff/internal/store"
	timelineutil "github.com/asobitov2005/OpsDiff/internal/timeline"
)

type IncidentOptions struct {
	From           string
	Limit          int
	Top            int
	PrometheusFile string
	ArgoCDFile     string
}

func parseLookback(value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse --from duration: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("--from must be greater than zero")
	}
	return duration, nil
}

func collectTimelineWithImports(ctx context.Context, root *RootOptions, namespace string, duration time.Duration, limit int, prometheusFile, argoCDFile string) (model.Timeline, error) {
	collector := kube.NewCollector(root.Kubeconfig)
	timeline, err := collector.CollectTimeline(ctx, namespace, duration, limit)
	if err != nil {
		return model.Timeline{}, err
	}

	extras, err := loadImportedTimelineEvents(prometheusFile, argoCDFile, timeline.WindowStart, timeline.WindowEnd)
	if err != nil {
		return model.Timeline{}, err
	}

	return timelineutil.Merge(timeline, extras, limit), nil
}

func loadImportedTimelineEvents(prometheusFile, argoCDFile string, windowStart, windowEnd time.Time) ([]model.TimelineEvent, error) {
	events := make([]model.TimelineEvent, 0)

	prometheusEvents, err := prometheus.LoadTimelineEvents(prometheusFile, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}
	events = append(events, prometheusEvents...)

	argocdEvents, err := argocd.LoadTimelineEvents(argoCDFile, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}
	events = append(events, argocdEvents...)

	return events, nil
}

func buildExplainContext(ctx context.Context, root *RootOptions, args []string, options IncidentOptions) (model.ExplainResult, model.Timeline, error) {
	duration, err := parseLookback(options.From)
	if err != nil {
		return model.ExplainResult{}, model.Timeline{}, err
	}

	before, err := store.ReadSnapshot(args[0])
	if err != nil {
		return model.ExplainResult{}, model.Timeline{}, err
	}

	after, err := store.ReadSnapshot(args[1])
	if err != nil {
		return model.ExplainResult{}, model.Timeline{}, err
	}

	namespace := resolveIncidentNamespace(root, after)
	timeline, err := collectTimelineWithImports(ctx, root, namespace, duration, options.Limit, options.PrometheusFile, options.ArgoCDFile)
	if err != nil {
		return model.ExplainResult{}, model.Timeline{}, err
	}

	result := explain.NewEngine().Explain(before, after, timeline, args[0], args[1], options.Top)
	return result, timeline, nil
}

func resolveIncidentNamespace(root *RootOptions, after model.Snapshot) string {
	namespace := root.Namespace
	if namespace == "default" && after.Namespace != "" && after.Namespace != "default" {
		namespace = after.Namespace
	}
	return namespace
}
