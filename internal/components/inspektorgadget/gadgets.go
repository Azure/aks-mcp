package inspektorgadget

import (
	"fmt"
	"strings"
)

// Gadget defines the structure of a gadget with its parameters and filter options
type Gadget struct {
	// Name for the gadget, LLM focused to help with use-case discovery
	Name string
	// Image is the Gadget image to be used
	Image string
	// Description provides a LLM focused description of the gadget
	Description string
	// Params are the LLM focused parameters that can be used to filter the gadget results
	Params map[string]interface{}
	// ParamsFunc is a function that the gadgetParams based on the filterParams
	ParamsFunc func(filterParams map[string]interface{}, gadgetParams map[string]string)
}

var gadgets = []Gadget{
	{
		Name:        observeDNS,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_dns:latest",
		Description: "Observes DNS queries in the cluster",
		Params: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Filter DNS traffic by name. Only DNS queries containing this string will be shown",
			},
			"nameserver": map[string]interface{}{
				"type":        "string",
				"description": "Filter DNS traffic by nameserver address",
			},
			"minimum_latency": map[string]interface{}{
				"type":        "string",
				"description": "Filter DNS queries with latency greater to this value (in nanoseconds)",
			},
			"response_code": map[string]interface{}{
				"type":        "string",
				"description": "Filter DNS queries by response code",
				"enum":        []string{"Success", "FormatError", "ServerFailure", "NameError", "NotImplemented", "Refused"},
			},
			"unsuccessful_only": map[string]interface{}{
				"type":        "boolean",
				"description": "Filter to only show unsuccessful DNS responses",
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			dnsParams, ok := filterParams[observeDNS].(map[string]interface{})
			if !ok {
				return
			}
			var filter []string
			if name, ok := dnsParams["name"]; ok && name != "" {
				filter = append(filter, fmt.Sprintf("name~%s", name))
			}
			if nameserver, ok := dnsParams["nameserver"]; ok && nameserver != "" {
				filter = append(filter, fmt.Sprintf("nameserver.addr==%s", nameserver))
			}
			if minimumLatency, ok := dnsParams["minimum_latency"]; ok && minimumLatency != "" {
				filter = append(filter, fmt.Sprintf("latency_ns_raw>=%s", minimumLatency))
			}
			if responseCode, ok := dnsParams["response_code"]; ok && responseCode != "" {
				filter = append(filter, fmt.Sprintf("rcode==%s", responseCode))
			}
			if unsuccessfulOnly, ok := dnsParams["unsuccessful_only"]; ok && unsuccessfulOnly.(bool) {
				filter = append(filter, "qr==R,rcode!=Success")
			}
			if len(filter) > 0 {
				gadgetParams[paramFilter] = strings.Join(filter, ",")
			}
		},
	},
	{
		Name:        observeTCP,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_tcp:latest",
		Description: "Observes TCP traffic in the cluster",
		Params: map[string]interface{}{
			"source_port": map[string]interface{}{
				"type":        "string",
				"description": "Filter TCP traffic by source port",
			},
			"destination_port": map[string]interface{}{
				"type":        "string",
				"description": "Filter TCP traffic by destination port",
			},
			"event_type": map[string]interface{}{
				"type":        "string",
				"description": "Filter TCP events by type",
				"enum":        []string{"connect", "accept", "close"},
			},
			"unsuccessful_only": map[string]interface{}{
				"type":        "boolean",
				"description": "Filter to only show unsuccessful TCP connections",
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			tcpParams, ok := filterParams[observeTCP].(map[string]interface{})
			if !ok {
				return
			}
			var filter []string
			if srcPort, ok := tcpParams["source_port"]; ok && srcPort != "" {
				filter = append(filter, fmt.Sprintf("src.port==%s", srcPort))
			}
			if dstPort, ok := tcpParams["destination_port"]; ok && dstPort != "" {
				filter = append(filter, fmt.Sprintf("dst.port==%s", dstPort))
			}
			if typ, ok := tcpParams["event_type"]; ok && typ != "" {
				filter = append(filter, fmt.Sprintf("type==%s", typ))
			}
			if unsuccessfulOnly, ok := tcpParams["unsuccessful_only"]; ok && unsuccessfulOnly.(bool) {
				filter = append(filter, "error_raw!=0")
			}
			if len(filter) > 0 {
				gadgetParams[paramFilter] = strings.Join(filter, ",")
			}
		},
	},
	{
		Name:        observeFileOpen,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_open:latest",
		Description: "Observes file open operations in the cluster",
		Params: map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Filter file operations by path. Only operations on files containing this string will be shown",
			},
			"unsuccessful_only": map[string]interface{}{
				"type":        "boolean",
				"description": "Filter to only show unsuccessful file open operations",
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			openParams, ok := filterParams[observeFileOpen].(map[string]interface{})
			if !ok {
				return
			}
			if path, ok := openParams["path"]; ok && path != "" {
				gadgetParams[paramFilter] = fmt.Sprintf("fname~%s", path)
			}
			if unsuccessfulOnly, ok := openParams["unsuccessful_only"]; ok && unsuccessfulOnly.(bool) {
				gadgetParams[paramFilter] = "error_raw!=0"
			}
		},
	},
	{
		Name:        observeProcessExecution,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_exec:latest",
		Description: "Observes process execution in the cluster",
		Params: map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Filter process execution by command name. Only processes executing the command containing this string will be shown",
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			execParams, ok := filterParams[observeProcessExecution].(map[string]interface{})
			if !ok {
				return
			}
			if command, ok := execParams["command"]; ok && command != "" {
				gadgetParams[paramFilter] = fmt.Sprintf("comm~%s", command)
			}
		},
	},
	{
		Name:        observeSignal,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_signal:latest",
		Description: "Traces signals sent to containers in the cluster",
		Params: map[string]interface{}{
			"signal": map[string]interface{}{
				"type":        "string",
				"description": "Filter by signal type",
				"enum":        []string{"SIGINT", "SIGTERM", "SIGKILL", "SIGHUP", "SIGURG", "SIGUSR1", "SIGUSR2", "SIGQUIT", "SIGSTOP"},
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			sigFilter, ok := filterParams[observeSignal].(map[string]interface{})
			if !ok {
				return
			}
			if signalFilter, ok := sigFilter["signal"]; ok && signalFilter != "" {
				gadgetParams[paramFilter] = fmt.Sprintf("sig==%s", signalFilter)
			}
		},
	},
	{
		Name:        observeSystemCalls,
		Image:       "ghcr.io/inspektor-gadget/gadget/traceloop:latest",
		Description: "Observes system calls in the cluster",
		Params: map[string]interface{}{
			"syscall": map[string]interface{}{
				"type":        "string",
				"description": "Filter by system call names (comma-separated list, e.g. 'open,close,read,write')",
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			// Always set the syscall filter parameter for traceloop
			gadgetParams[paramTraceloopSyscall] = ""
			traceloopParams, ok := filterParams[observeSystemCalls].(map[string]interface{})
			if !ok {
				return
			}
			if syscallFilter, ok := traceloopParams["syscall"].(string); ok && syscallFilter != "" {
				gadgetParams[paramTraceloopSyscall] = syscallFilter
			}
		},
	},
	{
		Name:        topFile,
		Image:       "ghcr.io/inspektor-gadget/gadget/top_file:latest",
		Description: "Shows top files by read/write operations",
		Params: map[string]interface{}{
			"max_entries": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of entries to return",
				"default":     10,
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			// Set default values for sort and limiter parameters
			gadgetParams[paramSort] = "-rbytes_raw,-wbytes_raw"
			gadgetParams[paramLimiter] = "10"
			topFileParams, ok := filterParams[topFile].(map[string]interface{})
			if !ok {
				return
			}
			if maxEntries, ok := topFileParams["max_entries"].(float64); ok && maxEntries > 0 {
				gadgetParams[paramLimiter] = fmt.Sprintf("%d", int(maxEntries))
			}
		},
	},
	{
		Name:        topTCP,
		Image:       "ghcr.io/inspektor-gadget/gadget/top_tcp:latest",
		Description: "Shows top TCP connections by traffic volume",
		Params: map[string]interface{}{
			"max_entries": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of entries to return",
				"default":     10,
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			// Set default values for sort and limiter parameters
			gadgetParams[paramSort] = "-sent_raw,-received_raw"
			gadgetParams[paramLimiter] = "10"
			topTCPParams, ok := filterParams[topTCP].(map[string]interface{})
			if !ok {
				return
			}
			if maxEntries, ok := topTCPParams["max_entries"].(float64); ok && maxEntries > 0 {
				gadgetParams[paramLimiter] = fmt.Sprintf("%d", int(maxEntries))
			}
		},
	},
}

func getGadgetNames() []string {
	names := make([]string, len(gadgets))
	for i, gadget := range gadgets {
		names[i] = gadget.Name
	}
	return names
}

func getGadgetByName(name string) (*Gadget, bool) {
	for _, gadget := range gadgets {
		if gadget.Name == name {
			return &gadget, true
		}
	}
	return nil, false
}

func getGadgetParams() map[string]interface{} {
	params := make(map[string]interface{})
	for _, gadget := range gadgets {
		if gadget.Params == nil {
			continue
		}
		params[gadget.Name] = gadget.Params
	}
	return params
}
