package grafana

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterGrafanaObservability registers the grafana_observability tool.
func RegisterGrafanaObservability() mcp.Tool {
	description := `Unified observability tool for querying the LGTM stack (Loki, Mimir, Tempo, Grafana).
Use this tool alongside kubectl to diagnose deployment failures end-to-end: kubectl gives you cluster state, this tool gives you logs, metrics, traces and alerts.

Supported Operations:

1. loki_query — Query pod/container logs using LogQL
   Required: query (LogQL expression)
   Optional: start_time, end_time (RFC3339, Unix nanoseconds, or relative duration like "1h"), limit (default 100)
   Use for: finding error messages, stack traces, crash reasons, application output
   Examples:
   - All error logs in a namespace: query='{namespace="my-ns"} |= "error"'
   - Logs for a specific deployment: query='{namespace="my-ns", app="my-app"} | json | level="error"'
   - Recent CrashLoopBackOff logs: query='{namespace="my-ns"} |~ "panic|fatal|OOM"', start_time="30m"

2. mimir_query — Query metrics using PromQL
   Required: query (PromQL expression)
   Optional: start_time, end_time (RFC3339 or Unix seconds, default: last 1h), step (default: 60s)
   Use for: pod restart counts, OOMKilled events, CPU/memory trends, error rates
   Examples:
   - Pod restarts: query='increase(kube_pod_container_status_restarts_total{namespace="my-ns"}[1h])'
   - OOMKilled events: query='kube_pod_container_status_last_terminated_reason{reason="OOMKilled", namespace="my-ns"}'
   - Memory usage: query='container_memory_working_set_bytes{namespace="my-ns", container="my-app"}'
   - HTTP error rate: query='sum(rate(http_requests_total{namespace="my-ns", status=~"5.."}[5m]))'

3. tempo_search — Search for traces using TraceQL
   Optional: query (TraceQL expression), start_time, end_time (Unix seconds), limit
   Use for: finding error traces, slow requests, tracing a specific service or namespace
   Examples:
   - Error traces in a service: query='{.service.name="my-service"} && status = error'
   - Traces by namespace: query='{.k8s.namespace.name="my-ns"}'
   - Slow traces: query='{.service.name="my-service"} && duration > 2s'

4. tempo_get_trace — Retrieve a full trace by ID
   Required: trace_id
   Use for: getting the complete span tree for a known trace ID (e.g. from a log line or tempo_search result)

5. alerts — List currently firing alerts from Grafana Alertmanager
   Optional: filter (Alertmanager label matcher, e.g. 'namespace="my-ns"'), silenced ("true"/"false"), inhibited ("true"/"false")
   Use for: checking whether an alert is already firing before investigating further, correlating alert context with failures
   Examples:
   - All active alerts: (no filter)
   - Alerts for a squad: filter='squad="ember"'
   - Alerts in a namespace: filter='namespace="my-ns"'

Diagnostic workflow for a failing deployment:
1. kubectl describe pod → identify the failure reason (CrashLoopBackOff, OOMKilled, etc.)
2. loki_query → pull logs around the time of the failure
3. mimir_query → check restart count trend and resource usage
4. alerts → check if any existing alerts cover this issue
5. tempo_search → find error traces if the service is instrumented
6. tempo_get_trace → drill into a specific trace for root cause
`

	return mcp.NewTool("grafana_observability",
		mcp.WithDescription(description),
		mcp.WithTitleAnnotation("Grafana Observability"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("Operation to perform: 'loki_query' (logs), 'mimir_query' (metrics), 'tempo_search' (find traces), 'tempo_get_trace' (full trace by ID), 'alerts' (firing Alertmanager alerts)"),
		),
		mcp.WithString("query",
			mcp.Description("Query expression. LogQL for loki_query, PromQL for mimir_query, TraceQL for tempo_search"),
		),
		mcp.WithString("trace_id",
			mcp.Description("Trace ID for tempo_get_trace operation"),
		),
		mcp.WithString("start_time",
			mcp.Description("Start of the query window. Accepts RFC3339 (2006-01-02T15:04:05Z), Unix timestamp, or relative duration (e.g. '1h', '30m'). Defaults to 1h ago"),
		),
		mcp.WithString("end_time",
			mcp.Description("End of the query window. Same formats as start_time. Defaults to now"),
		),
		mcp.WithString("step",
			mcp.Description("Step interval for mimir_query (e.g. '60s', '5m'). Defaults to 60s"),
		),
		mcp.WithString("limit",
			mcp.Description("Maximum number of results to return for loki_query and tempo_search. Defaults to 100"),
		),
		mcp.WithString("filter",
			mcp.Description("Alertmanager label matcher for the 'alerts' operation (e.g. 'namespace=\"my-ns\"', 'squad=\"ember\"')"),
		),
		mcp.WithString("silenced",
			mcp.Description("Include silenced alerts in 'alerts' operation. 'true' or 'false' (default: false)"),
		),
		mcp.WithString("inhibited",
			mcp.Description("Include inhibited alerts in 'alerts' operation. 'true' or 'false' (default: false)"),
		),
	)
}
