package k8s

import (
	"fmt"
	"strings"

	"github.com/Azure/mcp-kubernetes/pkg/kubectl"
	"github.com/Azure/mcp-kubernetes/pkg/security"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	AccessLevelReadOnly  = "readonly"
	AccessLevelReadWrite = "readwrite"
)

func createCallKubectlTool(accessLevel string) mcp.Tool {
	var description string

	readCommands := strings.Join(security.KubectlReadOperations, ", ")
	writeCommands := strings.Join(security.KubectlReadWriteOperations, ", ")

	switch accessLevel {
	case AccessLevelReadOnly:
		description = fmt.Sprintf(`Execute kubectl commands with read-only access.

Pass full kubectl command including 'kubectl' prefix. All standard kubectl flags are supported.

Allowed commands:
%s

Examples:
- command='kubectl get pods -n default'
- command='kubectl describe deployment myapp -n production'
- command='kubectl logs nginx-pod -f'
- command='kubectl top pods'
- command='kubectl events --all-namespaces'
- command='kubectl explain pods.spec.containers'
- command='kubectl auth can-i create pods'`, readCommands)
	case AccessLevelReadWrite:
		description = fmt.Sprintf(`Execute kubectl commands with read and write access.

Pass full kubectl command including 'kubectl' prefix. All standard kubectl flags are supported.

Allowed commands:
Read: %s
Write: %s

Examples:
- command='kubectl get pods -n default'
- command='kubectl create -f deployment.yaml'
- command='kubectl apply -f deployment.yaml'
- command='kubectl delete pod nginx-pod'
- command='kubectl scale deployment myapp --replicas=3'
- command='kubectl rollout status deployment/myapp'
- command='kubectl label pods foo unhealthy=true'
- command='kubectl exec nginx-pod -- date'
- command='kubectl config use-context my-cluster-context'`, readCommands, writeCommands)
	default:
		description = fmt.Sprintf(`Execute kubectl commands with unknown access level (defaulting to read-only).

Pass full kubectl command including 'kubectl' prefix. All standard kubectl flags are supported.

Allowed commands:
%s

Examples:
- command='kubectl get pods -n default'
- command='kubectl describe deployment myapp -n production'
- command='kubectl logs nginx-pod -f'`, readCommands)
	}

	return mcp.NewTool("call_kubectl",
		mcp.WithDescription(description),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Full kubectl command to execute (e.g., 'kubectl get pods -n default', 'kubectl describe deployment myapp', 'kubectl logs nginx-pod -f')"),
		),
		mcp.WithString("aks_resource_id",
			mcp.Required(),
			mcp.Description("Full Azure Resource ID of the AKS cluster (e.g., /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{clusterName})"),
		),
	)
}

func RegisterKubectlTools(accessLevel string, useUnifiedTool bool, enableMultiCluster bool) []mcp.Tool {
	if enableMultiCluster {
		return []mcp.Tool{
			createCallKubectlTool(accessLevel),
		}
	}

	return kubectl.RegisterKubectlTools(accessLevel, useUnifiedTool)
}
