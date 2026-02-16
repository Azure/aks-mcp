# Inspektor Gadget Observability Tool Usage in AKS-MCP Server

This document gives an overview of `inspektor_gadget_observability` tool in the AKS-MCP server.

## Tool Overview

The [Inspektor Gadget](https://go.microsoft.com/fwlink/?linkid=2260072) tool allows users to run various diagnostics and inspections on Kubernetes clusters.
It uses gadgets to collect real-time data with Kubernetes enrichment. `inspektor_gadget_observability` MCP tool essentially allows managing the gadgets, enabling users to
run diagnostics, collect data, and analyze workloads in a Kubernetes environment. It currently supports the following actions:

- **start**: Start a gadget to collect data continuously
- **stop**: Stop a running gadget
- **run**: Run a gadget with a specific duration and return the collected data
- **get_results**: Retrieve results from a previously started gadget
- **list**: List all running gadgets, can be used to stop or get results for running gadgets.
- **deploy**: Deploy the Inspektor Gadget in the cluster
- **undeploy**: Undeploy the Inspektor Gadget from the cluster
- **is_deployed**: Check the status of the Inspektor Gadget deployment

In terms of the type of data collected, it includes:

- **observe_dns**: Captures DNS queries across the cluster for troubleshooting service discovery issues, slow DNS lookups, and problematic upstream DNS servers.
- **observe_tcp**: Monitors TCP traffic to analyze network connections, identify bottlenecks, and diagnose connectivity problems.
- **observe_file_open**: Tracks file open operations to understand file access patterns and detect potential security issues or application misconfigurations.
- **observe_process_execution**: Records process lifecycle events to monitor unexpected process behavior and troubleshoot application issues.
- **observe_signal**: Traces signals sent to containers for debugging graceful shutdowns, process terminations, and other signal-related problems.
- **observe_system_calls**: Provides comprehensive system-level interaction data for debugging and performance analysis.
- **top_file**: Displays files with the highest read/write operations to pinpoint frequently accessed files and performance bottlenecks.
- **top_tcp**: Shows TCP connections sorted by traffic volume to identify high-traffic connections and network issues.
- **tcpdump**: Captures network traffic to troubleshoot network connectivity issues and identify potential security issues.

All the gadgets can be run for a specific pod or namespace. Also, if the user runs the MCP server with `--allow-namespaces=default,kube-system`, the data will be
limited to the specified namespaces otherwise it will be collected for all namespaces.  The gadgets also have use-case-specific parameters that can be passed to customize the data collection.
For example, with `observe_dns`, you can trace queries for specific domains or nameservers. Since output may be truncated, use specific identifiers in your prompts (pod names, namespaces, labels)
to get the most relevant data for your workload.

## Sample Prompts

Following are some sample prompts that can be used with to quickly try `inspektor_gadget_observability` tool in the AKS-MCP server:

```
Can you check if any DNS queries are failing in AKS cluster?
```

```
Can you give me DNS queries taking more than 5ms in the AKS cluster?
```

```
Can you continueously monitor DNS queries taking more than 500ms in the AKS cluster?
```

```
Can you get me results for already running gadgets monitoring DNS queries in the AKS cluster?
```

```
Can you check if any pods are having issues connecting each other in the AKS cluster?
```

```
Can you give me the top 3 pods with highest traffic in the AKS cluster?
```

```
Can you observe system calls for the pod my-pod in the default namespace for few seconds? I want to understand why it migth be slow.
```

```
Can you capture network traffic for the pod my-pod in the default namespace for 15 seconds? I want to debug connectivity issues.
```

## Prerequisites

- A kubeconfig file that has access to the AKS cluster.
- The tool requires Inspektor Gadget to be installed in the cluster. The MCP server will automatically
  install Inspektor Gadget (action `deploy` ) in the cluster otherwise you can follow the steps to install it manually: [Inspektor Gadget Installation](https://learn.microsoft.com/en-us/troubleshoot/azure/azure-kubernetes/logs/capture-system-insights-from-aks#how-to-install-inspektor-gadget-in-an-aks-cluster) or
  use the official Helm chart: [Inspektor Gadget Helm Chart](https://inspektor-gadget.io/docs/latest/reference/install-kubernetes#installation-with-the-helm-chart):

```bash
IG_VERSION=$(curl -s https://api.github.com/repos/inspektor-gadget/inspektor-gadget/releases/latest | jq -r '.tag_name' | sed 's/^v//')
helm install gadget --namespace=gadget --create-namespace oci://ghcr.io/inspektor-gadget/inspektor-gadget/charts/gadget --version=$IG_VERSION
```

This is useful if you want to deploy the AKS-MCP server in the cluster.