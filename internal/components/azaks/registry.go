package azaks

import (
	"fmt"
	"slices"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// AksOperationType defines the type of AKS operation
type AksOperationType string

const (
	// Cluster operations
	OpClusterShow           AksOperationType = "show"
	OpClusterList           AksOperationType = "list"
	OpClusterCreate         AksOperationType = "create"
	OpClusterDelete         AksOperationType = "delete"
	OpClusterScale          AksOperationType = "scale"
	OpClusterStart          AksOperationType = "start"
	OpClusterStop           AksOperationType = "stop"
	OpClusterUpdate         AksOperationType = "update"
	OpClusterUpgrade        AksOperationType = "upgrade"
	OpClusterGetVersions    AksOperationType = "get-versions"
	OpClusterCheckNetwork   AksOperationType = "check-network"
	OpClusterGetCredentials AksOperationType = "get-credentials"

	// Nodepool operations
	OpNodepoolList    AksOperationType = "nodepool-list"
	OpNodepoolShow    AksOperationType = "nodepool-show"
	OpNodepoolAdd     AksOperationType = "nodepool-add"
	OpNodepoolDelete  AksOperationType = "nodepool-delete"
	OpNodepoolScale   AksOperationType = "nodepool-scale"
	OpNodepoolUpgrade AksOperationType = "nodepool-upgrade"

	// Account operations
	OpAccountList AksOperationType = "account-list"
	OpAccountSet  AksOperationType = "account-set"
	OpLogin       AksOperationType = "login"
)

// generateToolDescription creates a tool description based on access level
func generateToolDescription(accessLevel string) string {
	baseDesc := "Unified tool for managing Azure Kubernetes Service (AKS) clusters and related operations.\n\nSupported operations:\n"

	var clusterOps, nodepoolOps, accountOps []string

	// Add read-only operations for all access levels
	clusterOps = append(clusterOps, "show", "list", "get-versions", "check-network")
	nodepoolOps = append(nodepoolOps, "nodepool-list", "nodepool-show")
	accountOps = append(accountOps, "account-list")

	// Add read-write operations for readwrite and admin
	if accessLevel == "readwrite" || accessLevel == "admin" {
		clusterOps = append(clusterOps, "create", "delete", "scale", "update", "upgrade", "start", "stop")
		nodepoolOps = append(nodepoolOps, "nodepool-add", "nodepool-delete", "nodepool-scale", "nodepool-upgrade")
		accountOps = append(accountOps, "account-set", "login")
	}

	// Add admin operations for admin only
	if accessLevel == "admin" {
		clusterOps = append(clusterOps, "get-credentials")
	}

	// Build the operations description
	desc := baseDesc
	desc += fmt.Sprintf("- Cluster: %s\n", joinOps(clusterOps))
	desc += fmt.Sprintf("- Nodepool: %s\n", joinOps(nodepoolOps))
	desc += fmt.Sprintf("- Account: %s\n", joinOps(accountOps))

	// Add examples based on access level
	desc += "\nExamples:\n"
	desc += "- Show cluster: operation=\"show\", args=\"--name myCluster --resource-group myRG\"\n"
	desc += "- List nodepools: operation=\"nodepool-list\", args=\"--cluster-name myCluster --resource-group myRG\"\n"

	// Only show write operation examples if access level allows it
	if accessLevel == "readwrite" || accessLevel == "admin" {
		desc += "- Scale cluster: operation=\"scale\", args=\"--name myCluster --resource-group myRG --node-count 5\"\n"
		desc += "- Update cluster (enable autoscaler): operation=\"update\", args=\"--name myCluster --resource-group myRG --enable-cluster-autoscaler --min-count 1 --max-count 5\"\n"
		desc += "- Update cluster (disable autoscaler): operation=\"update\", args=\"--name myCluster --resource-group myRG --disable-cluster-autoscaler\"\n"
	}

	return desc
}

// joinOps joins operation names with commas
func joinOps(ops []string) string {
	result := ""
	for i, op := range ops {
		if i > 0 {
			result += ", "
		}
		result += op
	}
	return result
}

// RegisterAzAksOperations registers the AKS operations tool
func RegisterAzAksOperations(cfg *config.ConfigData) mcp.Tool {
	description := generateToolDescription(cfg.AccessLevel)

	return mcp.NewTool("az_aks_operations",
		mcp.WithDescription(description),
		mcp.WithTitleAnnotation("AKS Operations"),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform"),
		),
		mcp.WithString("resource_type",
			mcp.Description("The resource type (cluster, nodepool, account). Can be inferred from operation."),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Arguments for the operation"),
		),
	)
}

// GetOperationAccessLevel returns the required access level for an operation
func GetOperationAccessLevel(operation string) string {
	readOnlyOps := []string{
		string(OpClusterShow), string(OpClusterList), string(OpClusterGetVersions),
		string(OpClusterCheckNetwork), string(OpNodepoolList), string(OpNodepoolShow),
		string(OpAccountList),
	}

	readWriteOps := []string{
		string(OpClusterCreate), string(OpClusterDelete), string(OpClusterScale),
		string(OpClusterUpdate), string(OpClusterUpgrade), string(OpClusterStart),
		string(OpClusterStop), string(OpNodepoolAdd), string(OpNodepoolDelete),
		string(OpNodepoolScale), string(OpNodepoolUpgrade), string(OpAccountSet),
		string(OpLogin),
	}

	adminOps := []string{
		string(OpClusterGetCredentials),
	}

	if slices.Contains(readOnlyOps, operation) {
		return "readonly"
	}

	if slices.Contains(readWriteOps, operation) {
		return "readwrite"
	}

	if slices.Contains(adminOps, operation) {
		return "admin"
	}

	return "unknown"
}

// ValidateOperationAccess checks if the operation is allowed for the given access level
func ValidateOperationAccess(operation string, cfg *config.ConfigData) error {
	requiredLevel := GetOperationAccessLevel(operation)

	switch requiredLevel {
	case "admin":
		if cfg.AccessLevel != "admin" {
			return fmt.Errorf("operation '%s' requires admin access level", operation)
		}
	case "readwrite":
		if cfg.AccessLevel != "readwrite" && cfg.AccessLevel != "admin" {
			return fmt.Errorf("operation '%s' requires readwrite or admin access level", operation)
		}
	case "readonly":
		// All access levels can perform readonly operations
	case "unknown":
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}

// MapOperationToCommand maps an operation to its corresponding az command
func MapOperationToCommand(operation string) (string, error) {
	commandMap := map[string]string{
		// Cluster operations
		string(OpClusterShow):           "az aks show",
		string(OpClusterList):           "az aks list",
		string(OpClusterCreate):         "az aks create",
		string(OpClusterDelete):         "az aks delete",
		string(OpClusterScale):          "az aks scale",
		string(OpClusterStart):          "az aks start",
		string(OpClusterStop):           "az aks stop",
		string(OpClusterUpdate):         "az aks update",
		string(OpClusterUpgrade):        "az aks upgrade",
		string(OpClusterGetVersions):    "az aks get-versions",
		string(OpClusterCheckNetwork):   "az aks check-network outbound",
		string(OpClusterGetCredentials): "az aks get-credentials",

		// Nodepool operations
		string(OpNodepoolList):    "az aks nodepool list",
		string(OpNodepoolShow):    "az aks nodepool show",
		string(OpNodepoolAdd):     "az aks nodepool add",
		string(OpNodepoolDelete):  "az aks nodepool delete",
		string(OpNodepoolScale):   "az aks nodepool scale",
		string(OpNodepoolUpgrade): "az aks nodepool upgrade",

		// Account operations
		string(OpAccountList): "az account list",
		string(OpAccountSet):  "az account set",
		string(OpLogin):       "az login",
	}

	cmd, exists := commandMap[operation]
	if !exists {
		return "", fmt.Errorf("no command mapping for operation: %s", operation)
	}

	return cmd, nil
}

// GetSupportedOperations returns a list of all supported operations
func GetSupportedOperations() []string {
	return []string{
		// Cluster operations
		string(OpClusterShow), string(OpClusterList), string(OpClusterCreate),
		string(OpClusterDelete), string(OpClusterScale), string(OpClusterStart),
		string(OpClusterStop), string(OpClusterUpdate), string(OpClusterUpgrade),
		string(OpClusterGetVersions), string(OpClusterCheckNetwork), string(OpClusterGetCredentials),
		// Nodepool operations
		string(OpNodepoolList), string(OpNodepoolShow), string(OpNodepoolAdd),
		string(OpNodepoolDelete), string(OpNodepoolScale), string(OpNodepoolUpgrade),
		// Account operations
		string(OpAccountList), string(OpAccountSet), string(OpLogin),
	}
}
