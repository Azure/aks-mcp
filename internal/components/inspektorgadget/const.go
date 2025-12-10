package inspektorgadget

// Lifecycle action constants for Inspektor Gadget
const (
	// deployAction is the action to deploy Inspektor Gadget to the cluster
	deployAction = "deploy"
	// undeployAction is the action to remove Inspektor Gadget from the cluster
	undeployAction = "undeploy"
	// upgradeAction is the action to upgrade Inspektor Gadget in the cluster
	upgradeAction = "upgrade"
	// isDeployedAction is the action to check if Inspektor Gadget is deployed
	isDeployedAction = "is_deployed"
)

// Gadget action constants for Inspektor Gadget
const (
	// runAction is the action to run a gadget for a specific duration
	runAction = "run"
	// startAction is the action to start a gadget for continuous observation
	startAction = "start"
	// stopAction is the action to stop a running gadget
	stopAction = "stop"
	// getResultsAction is the action to retrieve results of a gadget run
	getResultsAction = "get_results"
	// listGadgetsAction is the action to list all running gadgets
	listGadgetsAction = "list_gadgets"
)

// Gadget parameter constants for Inspektor Gadget operators
const (
	paramAllNamespaces    = "operator.KubeManager.all-namespaces"
	paramNamespace        = "operator.KubeManager.namespace"
	paramNode             = "node" // Node parameter is handled by the grpc runtime
	paramPod              = "operator.KubeManager.podname"
	paramContainer        = "operator.KubeManager.containername"
	paramSelector         = "operator.KubeManager.selector"
	paramSort             = "operator.sort.sort"
	paramLimiter          = "operator.limiter.max-entries"
	paramFetchInterval    = "operator.oci.ebpf.map-fetch-interval"
	paramFilter           = "operator.filter.filter"
	paramTraceloopSyscall = "operator.oci.wasm.syscall-filters"
	paramPacketFilter     = "operator.oci.ebpf.pf"
)

// Inspektor Gadget Helm chart constants
const (
	inspektorGadgetChartRelease   = "gadget"
	inspektorGadgetChartNamespace = "gadget"
	inspektorGadgetChartURL       = "oci://ghcr.io/inspektor-gadget/inspektor-gadget/charts/gadget"
	inspektorGadgetReleaseURL     = "https://api.github.com/repos/inspektor-gadget/inspektor-gadget/releases/latest"
)

// Name of the Inspektor Gadget gadgets
const (
	observeDNS              = "observe_dns"
	observeTCP              = "observe_tcp"
	observeFileOpen         = "observe_file_open"
	observeProcessExecution = "observe_process_execution"
	observeSignal           = "observe_signal"
	observeSystemCalls      = "observe_system_calls"
	topBlockIO              = "top_blockio"
	topFile                 = "top_file"
	topTCP                  = "top_tcp"
	tcpdump                 = "tcpdump"
	profileBlockIO          = "profile_blockio"
)
