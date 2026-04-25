package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/diff"
	"github.com/asobitov2005/OpsDiff/internal/explain"
	"github.com/asobitov2005/OpsDiff/internal/kube"
	"github.com/asobitov2005/OpsDiff/internal/store"
	"github.com/spf13/cobra"
)

func newWatchCommand(options *RootOptions) *cobra.Command {
	var (
		from           string
		interval       string
		iterations     int
		limit          int
		top            int
		prometheusFile string
		argoCDFile     string
		sqlitePath     string
		snapshotDir    string
	)

	command := &cobra.Command{
		Use:   "watch",
		Short: "Continuously collect snapshots and incident signals into SQLite",
		RunE: func(cmd *cobra.Command, _ []string) error {
			lookback, err := parseLookback(from)
			if err != nil {
				return err
			}

			intervalDuration, err := time.ParseDuration(interval)
			if err != nil {
				return fmt.Errorf("parse --interval duration: %w", err)
			}
			if intervalDuration <= 0 {
				return fmt.Errorf("--interval must be greater than zero")
			}

			sqliteStore, err := store.OpenSQLite(sqlitePath)
			if err != nil {
				return err
			}
			defer sqliteStore.Close()

			namespace := options.Namespace
			collector := kube.NewCollector(options.Kubeconfig)

			previous, err := sqliteStore.LatestSnapshot(cmd.Context(), namespace)
			if err != nil {
				return err
			}

			run := func(cycle int) error {
				snapshot, err := collector.CollectSnapshot(cmd.Context(), namespace)
				if err != nil {
					return err
				}

				resolvedNamespace := resolveIncidentNamespace(options, snapshot)
				snapshot.Namespace = resolvedNamespace

				snapshotPath := filepath.Join(snapshotDir, snapshotFilename(resolvedNamespace, snapshot.CreatedAt))
				if err := store.WriteSnapshot(snapshotPath, snapshot); err != nil {
					return err
				}

				if err := sqliteStore.SaveSnapshot(cmd.Context(), snapshot, snapshotPath); err != nil {
					return err
				}

				timeline, err := collectTimelineWithImports(cmd.Context(), options, resolvedNamespace, lookback, limit, prometheusFile, argoCDFile)
				if err != nil {
					return err
				}
				if err := sqliteStore.SaveTimeline(cmd.Context(), timeline); err != nil {
					return err
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[cycle %d] stored snapshot %s (%d resources), timeline events=%d\n",
					cycle,
					snapshotPath,
					len(snapshot.Resources),
					len(timeline.Events),
				)

				if previous != nil {
					compare := diff.NewEngine().Compare(previous.Snapshot, snapshot, previous.Path, snapshotPath)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "          diff summary: total=%d high=%d medium=%d low=%d\n",
						compare.Summary.Total,
						compare.Summary.High,
						compare.Summary.Medium,
						compare.Summary.Low,
					)

					if compare.Summary.Total > 0 {
						result := explain.NewEngine().Explain(previous.Snapshot, snapshot, timeline, previous.Path, snapshotPath, top)
						if len(result.Candidates) > 0 {
							topCandidate := result.Candidates[0]
							_, _ = fmt.Fprintf(cmd.OutOrStdout(), "          top candidate: [%d/100 %s] %s/%s %s\n",
								topCandidate.Score,
								strings.ToUpper(topCandidate.Likelihood),
								topCandidate.Change.ResourceKind,
								topCandidate.Change.ResourceName,
								topCandidate.Change.Path,
							)
						}
					}
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "          baseline snapshot stored")
				}

				previous = &store.SnapshotRecord{
					Path:     snapshotPath,
					Snapshot: snapshot,
				}
				return nil
			}

			cycle := 0
			for {
				cycle++
				if err := run(cycle); err != nil {
					return err
				}

				if iterations > 0 && cycle >= iterations {
					return nil
				}

				select {
				case <-cmd.Context().Done():
					return cmd.Context().Err()
				case <-time.After(intervalDuration):
				}
			}
		},
	}

	command.Flags().StringVar(&from, "from", "2h", "Relative lookback window such as 30m, 2h, or 24h")
	command.Flags().StringVar(&interval, "interval", "1m", "Polling interval such as 30s, 1m, or 5m")
	command.Flags().IntVar(&iterations, "iterations", 0, "Number of collection cycles to run; 0 means run until interrupted")
	command.Flags().IntVar(&limit, "limit", 200, "Maximum number of timeline events to collect per cycle")
	command.Flags().IntVar(&top, "top", 3, "Maximum number of likely-cause candidates to evaluate per cycle")
	command.Flags().StringVar(&prometheusFile, "prometheus-file", "", "Path to a Prometheus alert export JSON file")
	command.Flags().StringVar(&argoCDFile, "argocd-file", "", "Path to an ArgoCD sync event export JSON file")
	command.Flags().StringVar(&sqlitePath, "db-path", defaultSQLitePath(), "Path to the SQLite event store")
	command.Flags().StringVar(&snapshotDir, "snapshot-dir", defaultSnapshotDir(), "Directory where watch mode writes snapshot JSON files")

	return command
}

func snapshotFilename(namespace string, timestamp time.Time) string {
	return fmt.Sprintf("%s-%s.json", sanitizeFileComponent(namespace), timestamp.UTC().Format("20060102T150405Z"))
}

func sanitizeFileComponent(value string) string {
	if strings.TrimSpace(value) == "" {
		return "default"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-")
	return replacer.Replace(value)
}
