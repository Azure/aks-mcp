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
	// ParamsFunc is a function that prepares gadgetParams based on the filterParams
	ParamsFunc func(filterParams map[string]interface{}, gadgetParams map[string]string)
}

func (g *Gadget) getImage(version string) string {
	return fmt.Sprintf("%s:%s", g.Image, gadgetVersionFor(version))
}

var gadgets = []Gadget{
	{
		Name:        observeDNS,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_dns",
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
			dnsParams, ok := getGadgetParam(filterParams, observeDNS)
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
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_tcp",
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
			tcpParams, ok := getGadgetParam(filterParams, observeTCP)
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
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_open",
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
			fileOpenParams, ok := getGadgetParam(filterParams, observeFileOpen)
			if !ok {
				return
			}
			var filter []string
			if path, ok := fileOpenParams["path"]; ok && path != "" {
				filter = append(filter, fmt.Sprintf("fname~%s", path))
			}
			if unsuccessfulOnly, ok := fileOpenParams["unsuccessful_only"]; ok && unsuccessfulOnly.(bool) {
				filter = append(filter, "error_raw!=0")
			}
			if len(filter) > 0 {
				gadgetParams[paramFilter] = strings.Join(filter, ",")
			}
		},
	},
	{
		Name:        observeProcessExecution,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_exec",
		Description: "Observes process execution in the cluster",
		Params: map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Filter process execution by command name. Only processes executing the command containing this string will be shown",
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			processParams, ok := getGadgetParam(filterParams, observeProcessExecution)
			if !ok {
				return
			}
			if command, ok := processParams["command"]; ok && command != "" {
				gadgetParams[paramFilter] = fmt.Sprintf("proc.comm~%s", command)
			}
		},
	},
	{
		Name:        observeSignal,
		Image:       "ghcr.io/inspektor-gadget/gadget/trace_signal",
		Description: "Traces signals sent to containers in the cluster",
		Params: map[string]interface{}{
			"signal": map[string]interface{}{
				"type":        "string",
				"description": "Filter by signal type",
				"enum":        []string{"SIGINT", "SIGTERM", "SIGKILL", "SIGHUP", "SIGURG", "SIGUSR1", "SIGUSR2", "SIGQUIT", "SIGSTOP"},
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			signalParams, ok := getGadgetParam(filterParams, observeSignal)
			if !ok {
				return
			}
			if signalFilter, ok := signalParams["signal"]; ok && signalFilter != "" {
				gadgetParams[paramFilter] = fmt.Sprintf("sig==%s", signalFilter)
			}
		},
	},
	{
		Name:        observeSystemCalls,
		Image:       "ghcr.io/inspektor-gadget/gadget/traceloop",
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
			syscallParams, ok := getGadgetParam(filterParams, observeSystemCalls)
			if !ok {
				return
			}
			if syscallFilter, ok := syscallParams["syscall"].(string); ok && syscallFilter != "" {
				gadgetParams[paramTraceloopSyscall] = syscallFilter
			}
		},
	},
	{
		Name:        topFile,
		Image:       "ghcr.io/inspektor-gadget/gadget/top_file",
		Description: "Shows top files by read/write operations",
		Params: map[string]interface{}{
			"max_entries": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of entries to return",
				"default":     5,
			},
			"sort_by": map[string]interface{}{
				"type":        "string",
				"description": "Which metric to sort the results by.",
				"default":     "wbytes_raw",
				"enum":        []string{"rbytes_raw", "wbytes_raw"},
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			// Set default values for sort and limiter parameters
			gadgetParams[paramLimiter] = "5"

			topFileParams, ok := getGadgetParam(filterParams, topFile)
			if !ok {
				return
			}
			if maxEntries, ok := topFileParams["max_entries"].(float64); ok && maxEntries > 0 {
				gadgetParams[paramLimiter] = fmt.Sprintf("%d", int(maxEntries))
			}
			if sortBy, ok := topFileParams["sort_by"].(string); ok && sortBy != "" {
				gadgetParams[paramSort] = fmt.Sprintf("-%s", sortBy)
			}
		},
	},
	{
		Name:        topTCP,
		Image:       "ghcr.io/inspektor-gadget/gadget/top_tcp",
		Description: "Shows top TCP connections by traffic volume",
		Params: map[string]interface{}{
			"max_entries": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of entries to return",
				"default":     5,
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			// Set default values for sort and limiter parameters
			gadgetParams[paramSort] = "-sent_raw,-received_raw"
			gadgetParams[paramLimiter] = "5"

			topTCPParams, ok := getGadgetParam(filterParams, topTCP)
			if !ok {
				return
			}
			if maxEntries, ok := topTCPParams["max_entries"].(float64); ok && maxEntries > 0 {
				gadgetParams[paramLimiter] = fmt.Sprintf("%d", int(maxEntries))
			}
		},
	},
	{
		Name:        tcpdump,
		Image:       "ghcr.io/inspektor-gadget/gadget/tcpdump",
		Description: "Captures network traffic in the cluster",
		Params: map[string]interface{}{
			"pcap-filter": map[string]interface{}{
				"type": "string",
				"description": `Parses tcpdump/BPF-style filter expressions for network packet filtering.
				Syntax:
				expression = [not] [direction] [protocol] type [id] | expression and expression | expression or expression | ( expression )
                Qualifiers:
				  direction = src | dst | src or dst | src and dst
				  protocol = ether | fddi | tr | wlan | ip | ip6 | arp | rarp | decnet | tcp | udp | icmp | igmp | atalk | aarp | vlan | ppptalk | pptp | sctp | mpls
				  type = host | net | port | portrange | less | greater | gateway | proto (protocol)
				  id = address | number
				Examples:
				  host 192.168.1.1
				  src port 80
				  tcp dst port 443
				  not src host 10.0.0.1
				  (src port 22) and (dst host 192.168.1.1)
				  ip6 and not tcp
				  proto icmp
				  ether host aa:bb:cc:dd:ee:ff
				  src or dst port 80 or 443
				`,
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			tcpdumpParams, ok := getGadgetParam(filterParams, tcpdump)
			if !ok {
				return
			}
			if filter, ok := tcpdumpParams["pcap-filter"].(string); ok && filter != "" {
				gadgetParams[paramPacketFilter] = filter
			}
		},
	},
	{
		Name:        profileBlockIO,
		Image:       "ghcr.io/inspektor-gadget/gadget/profile_blockio",
		Description: "Profiles a single node and provides a histogram of block IO (disk) latency for it",
		Params: map[string]interface{}{
			"node": map[string]interface{}{
				"type":        "string",
				"description": "Target node of the block IO latency profiling. Result is a series of histograms for the given node",
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			gadgetParams[paramNode] = ""
			nodeParams, ok := getGadgetParam(filterParams, profileBlockIO)
			if !ok {
				return
			}
			if nodeFilter, ok := nodeParams["node"].(string); ok && nodeFilter != "" {
				gadgetParams[paramNode] = nodeFilter
			}
		},
	},
	{
		Name:        topBlockIO,
		Image:       "ghcr.io/inspektor-gadget/gadget/top_blockio",
		Description: "Shows top block IO (disk) activity by bytes for read/write operations (requires Kernel version >=6.6)",
		Params: map[string]interface{}{
			"node": map[string]interface{}{
				"type":        "string",
				"description": "Target node of this operation",
			},
			"max_entries": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of entries to return",
				"default":     5,
			},
		},
		ParamsFunc: func(filterParams map[string]interface{}, gadgetParams map[string]string) {
			gadgetParams[paramNode] = ""
			gadgetParams[paramSort] = "-bytes"
			gadgetParams[paramLimiter] = "5"

			topBlockIOParams, ok := getGadgetParam(filterParams, topBlockIO)
			if !ok {
				return
			}
			if nodeFilter, ok := topBlockIOParams["node"].(string); ok && nodeFilter != "" {
				gadgetParams[paramNode] = nodeFilter
			}
			if maxEntries, ok := topBlockIOParams["max_entries"].(float64); ok && maxEntries > 0 {
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

// getGadgetParams returns a map of all gadget parameters with their names prefixed by the gadget name
func getGadgetParams() map[string]interface{} {
	params := make(map[string]interface{})
	for _, gadget := range gadgets {
		if gadget.Params == nil {
			continue
		}
		for key, value := range gadget.Params {
			params[gadget.Name+"."+key] = value
		}
	}
	return params
}

func getGadgetParamsKeys() []string {
	keys := make([]string, 0, len(getGadgetParams()))
	for key := range getGadgetParams() {
		keys = append(keys, key)
	}
	return keys
}

func getGadgetParam(filterParams map[string]interface{}, name string) (map[string]interface{}, bool) {
	if filterParams == nil {
		return nil, false
	}
	params := make(map[string]interface{})
	for key, value := range filterParams {
		if strings.HasPrefix(key, name+".") {
			paramKey := strings.TrimPrefix(key, name+".")
			params[paramKey] = value
		}
	}
	return params, len(params) > 0
}
