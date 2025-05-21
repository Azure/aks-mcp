// Package models provides data models for AKS MCP server.
package models

// ClusterInfo represents basic information about an AKS cluster.
type ClusterInfo struct {
	Name                string   `json:"name"`
	ResourceGroup       string   `json:"resourceGroup"`
	Location            string   `json:"location"`
	KubernetesVersion   string   `json:"kubernetesVersion"`
	NodeResourceGroup   string   `json:"nodeResourceGroup"`
	NetworkPlugin       string   `json:"networkPlugin"`
	NetworkPolicy       string   `json:"networkPolicy"`
	DNSPrefix           string   `json:"dnsPrefix"`
	FQDN                string   `json:"fqdn"`
	AgentPoolProfiles   []string `json:"agentPoolProfiles"`
	SubscriptionID      string   `json:"subscriptionId"`
	ResourceID          string   `json:"resourceId"`
	NetworkProfile      string   `json:"networkProfile"`
	APIServerAccessProfile string `json:"apiServerAccessProfile"`
}

// VNetInfo represents information about a virtual network.
type VNetInfo struct {
	Name              string                 `json:"name"`
	ResourceGroup     string                 `json:"resourceGroup"`
	Location          string                 `json:"location"`
	ID                string                 `json:"id"`
	AddressSpace      []string               `json:"addressSpace"`
	Subnets           []SubnetInfo           `json:"subnets"`
	Tags              map[string]string      `json:"tags"`
	ResourceGUID      string                 `json:"resourceGuid"`
	ProvisioningState string                 `json:"provisioningState"`
}

// SubnetInfo represents information about a subnet.
type SubnetInfo struct {
	Name                 string `json:"name"`
	ID                   string `json:"id"`
	AddressPrefix        string `json:"addressPrefix"`
	NetworkSecurityGroup string `json:"networkSecurityGroup,omitempty"`
	RouteTable           string `json:"routeTable,omitempty"`
	ProvisioningState    string `json:"provisioningState"`
}

// RouteTableInfo represents information about a route table.
type RouteTableInfo struct {
	Name              string      `json:"name"`
	ResourceGroup     string      `json:"resourceGroup"`
	Location          string      `json:"location"`
	ID                string      `json:"id"`
	Routes            []RouteInfo `json:"routes"`
	Tags              map[string]string `json:"tags"`
	ProvisioningState string      `json:"provisioningState"`
}

// RouteInfo represents information about a route.
type RouteInfo struct {
	Name                   string `json:"name"`
	ID                     string `json:"id"`
	AddressPrefix          string `json:"addressPrefix"`
	NextHopType            string `json:"nextHopType"`
	NextHopIPAddress       string `json:"nextHopIpAddress,omitempty"`
	ProvisioningState      string `json:"provisioningState"`
}

// NSGInfo represents information about a network security group.
type NSGInfo struct {
	Name                 string      `json:"name"`
	ResourceGroup        string      `json:"resourceGroup"`
	Location             string      `json:"location"`
	ID                   string      `json:"id"`
	SecurityRules        []NSGRule   `json:"securityRules"`
	DefaultSecurityRules []NSGRule   `json:"defaultSecurityRules"`
	Tags                 map[string]string `json:"tags"`
	ProvisioningState    string      `json:"provisioningState"`
}

// NSGRule represents information about a network security group rule.
type NSGRule struct {
	Name                     string `json:"name"`
	ID                       string `json:"id"`
	Protocol                 string `json:"protocol"`
	SourceAddressPrefix      string `json:"sourceAddressPrefix"`
	SourcePortRange          string `json:"sourcePortRange"`
	DestinationAddressPrefix string `json:"destinationAddressPrefix"`
	DestinationPortRange     string `json:"destinationPortRange"`
	Access                   string `json:"access"`
	Priority                 int32  `json:"priority"`
	Direction                string `json:"direction"`
	ProvisioningState        string `json:"provisioningState"`
}
