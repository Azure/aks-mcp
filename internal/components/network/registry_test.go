package network

import (
	"testing"

	"github.com/Azure/aks-mcp/internal/azureclient"
	"github.com/Azure/aks-mcp/internal/config"
)

func TestGetVNetInfoHandler(t *testing.T) {
	mockClient := &azureclient.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetVNetInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetNSGInfoHandler(t *testing.T) {
	mockClient := &azureclient.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetNSGInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetRouteTableInfoHandler(t *testing.T) {
	mockClient := &azureclient.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetRouteTableInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetSubnetInfoHandler(t *testing.T) {
	mockClient := &azureclient.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetSubnetInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestGetPrivateEndpointInfoHandler(t *testing.T) {
	mockClient := &azureclient.AzureClient{}
	cfg := &config.ConfigData{}

	handler := GetPrivateEndpointInfoHandler(mockClient, cfg)

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}
}
