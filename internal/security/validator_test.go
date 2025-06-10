package security

import (
	"testing"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name        string
		accessLevel string
		command     string
		wantErr     bool
	}{
		{
			name:        "ReadOnly_ReadCommand_ShouldSucceed",
			accessLevel: "readonly",
			command:     "az aks show --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "ReadOnly_WriteCommand_ShouldFail",
			accessLevel: "readonly",
			command:     "az aks create --name myCluster --resource-group myRG",
			wantErr:     true,
		},
		{
			name:        "ReadWrite_ReadCommand_ShouldSucceed",
			accessLevel: "readwrite",
			command:     "az aks show --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "ReadWrite_WriteCommand_ShouldSucceed",
			accessLevel: "readwrite",
			command:     "az aks create --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "Admin_ReadCommand_ShouldSucceed",
			accessLevel: "admin",
			command:     "az aks show --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "Admin_WriteCommand_ShouldSucceed",
			accessLevel: "admin",
			command:     "az aks create --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "Admin_AdminCommand_ShouldSucceed",
			accessLevel: "admin",
			command:     "az aks get-credentials --name myCluster --resource-group myRG",
			wantErr:     false,
		},
		{
			name:        "PartialCommand_ShouldMatch",
			accessLevel: "readonly",
			command:     "az version",
			wantErr:     false,
		},
		{
			name:        "AccountCommands_ShouldWork",
			accessLevel: "readonly",
			command:     "az account list",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secConfig := &SecurityConfig{
				AccessLevel: tt.accessLevel,
			}
			validator := NewValidator(secConfig)
			err := validator.ValidateCommand(tt.command, CommandTypeAz)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsReadOperation(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		allowList []string
		want      bool
	}{
		{
			name:      "ExactMatch",
			command:   "az aks show --name myCluster",
			allowList: []string{"az aks show", "az aks list"},
			want:      true,
		},
		{
			name:      "NoMatch",
			command:   "az aks create --name myCluster",
			allowList: []string{"az aks show", "az aks list"},
			want:      false,
		},
		{
			name:      "PrefixMatch",
			command:   "az account list",
			allowList: []string{"az account", "az aks show"},
			want:      true,
		},
		{
			name:      "TwoWordCommand",
			command:   "az version",
			allowList: []string{"az version", "az help"},
			want:      true,
		},
		{
			name:      "NonAzCommand",
			command:   "kubectl get pods",
			allowList: []string{"az aks show", "az aks list"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(&SecurityConfig{})
			if got := validator.isReadOperation(tt.command, tt.allowList); got != tt.want {
				t.Errorf("isReadOperation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessLevelValidation(t *testing.T) {
	// Setup common allowed operations
	readOps := []string{"az aks show", "az aks list"}

	tests := []struct {
		name        string
		accessLevel string
		command     string
		wantErr     bool
	}{
		{
			name:        "ReadOnly_ReadCommand",
			accessLevel: "readonly",
			command:     "az aks show --name myCluster",
			wantErr:     false,
		},
		{
			name:        "ReadOnly_WriteCommand",
			accessLevel: "readonly",
			command:     "az aks create --name myCluster",
			wantErr:     true,
		},
		{
			name:        "ReadWrite_ReadCommand",
			accessLevel: "readwrite",
			command:     "az aks show --name myCluster",
			wantErr:     false,
		},
		{
			name:        "ReadWrite_WriteCommand",
			accessLevel: "readwrite",
			command:     "az aks create --name myCluster",
			wantErr:     false,
		},
		{
			name:        "Admin_ReadCommand",
			accessLevel: "admin",
			command:     "az aks show --name myCluster",
			wantErr:     false,
		},
		{
			name:        "Admin_WriteCommand",
			accessLevel: "admin",
			command:     "az aks create --name myCluster",
			wantErr:     false,
		},
		{
			name:        "Unknown_ReadCommand",
			accessLevel: "unknown",
			command:     "az aks show --name myCluster",
			wantErr:     false, // Default to readwrite behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secConfig := &SecurityConfig{
				AccessLevel: tt.accessLevel,
			}
			validator := NewValidator(secConfig)
			err := validator.validateAccessLevel(tt.command, readOps)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAccessLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
