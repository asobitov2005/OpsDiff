package report

import (
	"bytes"
	"html/template"
	"strings"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func RenderIncidentHTML(result model.ExplainResult, timeline model.Timeline) (string, error) {
	tmpl, err := template.New("incident-report").Funcs(template.FuncMap{
		"fmtTime": func(value time.Time) string {
			if value.IsZero() {
				return ""
			}
			return value.Format(time.RFC3339)
		},
		"upper": strings.ToUpper,
		"join": func(values []string, sep string) string {
			return strings.Join(values, sep)
		},
	}).Parse(incidentHTMLTemplate)
	if err != nil {
		return "", err
	}

	payload := struct {
		Result   model.ExplainResult
		Timeline model.Timeline
	}{
		Result:   result,
		Timeline: timeline,
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, payload); err != nil {
		return "", err
	}
	return out.String(), nil
}

const incidentHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>OpsDiff Incident Report</title>
  <style>
    :root {
      --bg: #f7f7f2;
      --panel: #ffffff;
      --ink: #14213d;
      --muted: #5b6470;
      --line: #d8dee6;
      --accent: #0f766e;
      --warn: #b45309;
      --crit: #b91c1c;
    }
    body {
      margin: 0;
      padding: 32px;
      background: linear-gradient(180deg, #f7f7f2 0%, #eef4f7 100%);
      color: var(--ink);
      font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
    }
    .wrap {
      max-width: 1180px;
      margin: 0 auto;
      display: grid;
      gap: 24px;
    }
    .hero, .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 24px;
      box-shadow: 0 14px 40px rgba(20, 33, 61, 0.06);
    }
    h1, h2, h3 {
      margin: 0 0 12px;
    }
    .eyebrow {
      text-transform: uppercase;
      letter-spacing: 0.12em;
      font-size: 12px;
      color: var(--accent);
      font-weight: 700;
      margin-bottom: 8px;
    }
    .subtle {
      color: var(--muted);
      margin: 0;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
      gap: 12px;
      margin-top: 20px;
    }
    .stat {
      background: #f8fafc;
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 14px;
    }
    .stat b {
      display: block;
      font-size: 24px;
      margin-top: 6px;
    }
    .candidate {
      border-top: 1px solid var(--line);
      padding-top: 18px;
      margin-top: 18px;
    }
    .score {
      display: inline-block;
      padding: 6px 10px;
      border-radius: 999px;
      background: #e7f5f3;
      color: var(--accent);
      font-weight: 700;
      margin-right: 10px;
    }
    .high { color: var(--crit); }
    .medium { color: var(--warn); }
    .low { color: var(--accent); }
    ul {
      margin: 10px 0 0;
      padding-left: 18px;
    }
    li {
      margin-bottom: 8px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 14px;
    }
    th, td {
      border-top: 1px solid var(--line);
      padding: 10px 8px;
      text-align: left;
      vertical-align: top;
    }
    th {
      color: var(--muted);
      font-weight: 600;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
    .grid {
      display: grid;
      gap: 24px;
      grid-template-columns: 1.2fr 0.8fr;
    }
    @media (max-width: 920px) {
      body { padding: 16px; }
      .grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <section class="hero">
      <div class="eyebrow">OpsDiff v0.4 Incident Report</div>
      <h1>Find what changed before it broke</h1>
      <p class="subtle">Snapshots: {{.Result.BeforePath}} -> {{.Result.AfterPath}}</p>
      <p class="subtle">Namespace: {{.Result.Namespace}} | Change window: {{fmtTime .Result.ChangeWindow.Start}} -> {{fmtTime .Result.ChangeWindow.End}}</p>
      <div class="stats">
        <div class="stat">Ranked candidates<b>{{.Result.Summary.RankedCandidates}}</b></div>
        <div class="stat">Critical symptoms<b>{{.Result.Summary.CriticalSymptoms}}</b></div>
        <div class="stat">Warning symptoms<b>{{.Result.Summary.WarningSymptoms}}</b></div>
        <div class="stat">Supporting evidence<b>{{.Result.Summary.SupportingEvidence}}</b></div>
        <div class="stat">Timeline events<b>{{.Timeline.Summary.Total}}</b></div>
      </div>
    </section>

    <section class="grid">
      <div class="panel">
        <h2>Likely Causes</h2>
        {{if .Result.Candidates}}
          {{range .Result.Candidates}}
            <div class="candidate">
              <h3><span class="score">{{.Score}}/100</span><span class="{{.Likelihood}}">{{upper .Likelihood}}</span> {{.Change.ResourceKind}}/{{.Change.ResourceName}}</h3>
              <p class="subtle">{{.Change.Path}}: {{.Change.Before}} -> {{.Change.After}}</p>
              {{if .Evidence}}
                <ul>
                  {{range .Evidence}}<li>{{.}}</li>{{end}}
                </ul>
              {{end}}
              {{if .SuggestedChecks}}
                <p><b>Suggested checks</b></p>
                <ul>
                  {{range .SuggestedChecks}}<li>{{.}}</li>{{end}}
                </ul>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="subtle">No ranked candidates were produced.</p>
        {{end}}
      </div>

      <div class="panel">
        <h2>Timeline Summary</h2>
        <div class="stats">
          <div class="stat">Changes<b>{{.Timeline.Summary.Changes}}</b></div>
          <div class="stat">Symptoms<b>{{.Timeline.Summary.Symptoms}}</b></div>
          <div class="stat">Evidence<b>{{.Timeline.Summary.Evidence}}</b></div>
          <div class="stat">Restarts<b>{{.Timeline.Summary.Restarts}}</b></div>
          <div class="stat">OOMKills<b>{{.Timeline.Summary.OOMKills}}</b></div>
          <div class="stat">CrashLoops<b>{{.Timeline.Summary.CrashLoops}}</b></div>
        </div>
      </div>
    </section>

    <section class="panel">
      <h2>Incident Timeline</h2>
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>Severity</th>
            <th>Source</th>
            <th>Service</th>
            <th>Reason</th>
            <th>Message</th>
          </tr>
        </thead>
        <tbody>
          {{range .Timeline.Events}}
            <tr>
              <td>{{fmtTime .Time}}</td>
              <td>{{upper .Severity}}</td>
              <td>{{.Source}}</td>
              <td>{{.Service}}</td>
              <td>{{.Reason}}</td>
              <td>{{.Message}}</td>
            </tr>
          {{else}}
            <tr><td colspan="6">No timeline events available.</td></tr>
          {{end}}
        </tbody>
      </table>
    </section>
  </div>
</body>
</html>`
