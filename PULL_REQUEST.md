# Azure Advisor Recommendations Tool for AKS Clusters

## 🎯 **Overview**
This PR implements a comprehensive Azure Advisor recommendation tool specifically designed for AKS (Azure Kubernetes Service) clusters. The tool integrates with the AKS-MCP server to provide AI assistants with the ability to retrieve, analyze, and report on Azure Advisor recommendations for AKS resources.

## 🚀 **Features Implemented**

### **Core Functionality**
- ✅ **Azure Advisor Integration**: Direct integration with Azure CLI to fetch real-time recommendations
- ✅ **AKS-Specific Filtering**: Automatically filters recommendations to only include AKS-related resources
- ✅ **Multiple Operations**: Support for `list`, `details`, and `report` operations
- ✅ **Comprehensive Logging**: Detailed logging with `[ADVISOR]` prefix for debugging and monitoring
- ✅ **Security Integration**: Works with readonly, readwrite, and admin access levels

### **Data Structure & API**
- ✅ **ResourceID Field**: Returns `resourceID` (Azure resource ID) instead of generic impacted value
- ✅ **Structured Response**: Well-defined JSON response format with AKS-specific metadata
- ✅ **Filtering Support**: Filter by severity, resource group, category, and cluster names
- ✅ **Report Generation**: Comprehensive reports with summaries, action items, and cluster breakdowns

### **Quality & Testing**
- ✅ **Complete Test Coverage**: 10 comprehensive test cases covering all functionality
- ✅ **Error Handling**: Robust error handling with informative error messages
- ✅ **Input Validation**: Proper parameter validation and type checking
- ✅ **Documentation**: Comprehensive usage documentation and examples

## 📋 **Changes Made**

### **New Files Added**
```
internal/azure/advisor/
├── aks_recommendations.go    # Main advisor logic with comprehensive logging
├── types.go                  # Data structures and type definitions
└── advisor_test.go          # Complete test suite (10 test cases)

docs/
└── logging.md               # Logging setup and monitoring documentation
```

### **Files Modified**
```
internal/security/validator.go    # Added advisor tool to security validation
.gitignore                        # Added *.ps1 to ignore development scripts
```

### **Key Technical Improvements**
1. **Data Structure Fix**: Updated `CLIRecommendation` struct to match actual Azure CLI output (flattened fields)
2. **ResourceID Integration**: Changed from `ImpactedResource` to `ResourceID` field for better clarity
3. **Comprehensive Logging**: Added detailed logging throughout the tool for debugging and monitoring
4. **Test Coverage**: Fixed all test cases to work with the new data structure and added ResourceID validation

## 🔧 **Usage Examples**

### **List AKS Recommendations**
```json
{
  "operation": "list",
  "subscription_id": "your-subscription-id",
  "resource_group": "your-resource-group",
  "severity": "High"
}
```

### **Get Recommendation Details**
```json
{
  "operation": "details",
  "recommendation_id": "/subscriptions/.../recommendations/rec-id"
}
```

### **Generate Comprehensive Report**
```json
{
  "operation": "report",
  "subscription_id": "your-subscription-id",
  "format": "detailed"
}
```

## 📊 **Response Format**
```json
{
  "id": "/subscriptions/.../recommendations/rec-id",
  "category": "Cost",
  "impact": "High",
  "cluster_name": "my-aks-cluster",
  "resource_group": "my-resource-group",
  "resource_id": "/subscriptions/.../managedClusters/my-aks-cluster",
  "description": "Detailed recommendation with solution",
  "severity": "High",
  "last_updated": "2024-01-15T10:30:00Z",
  "status": "Active",
  "aks_specific": {
    "configuration_area": "compute"
  }
}
```

## 🧪 **Testing**

### **Test Coverage**
- ✅ `TestFilterAKSRecommendationsFromCLI` - AKS-specific filtering
- ✅ `TestIsAKSRelatedCLI` - Resource ID validation
- ✅ `TestExtractAKSClusterNameFromCLI` - Cluster name extraction
- ✅ `TestExtractResourceGroupFromResourceID` - Resource group parsing
- ✅ `TestConvertToAKSRecommendationSummary` - Data transformation & ResourceID validation
- ✅ `TestFilterBySeverity` - Severity-based filtering
- ✅ `TestGenerateAKSAdvisorReport` - Report generation
- ✅ `TestMapCategoryToConfigArea` - Category mapping
- ✅ `TestHandleAdvisorRecommendationInvalidOperation` - Error handling
- ✅ `TestHandleAdvisorRecommendationMissingOperation` - Parameter validation

### **Test Results**
```bash
=== RUN   TestFilterAKSRecommendationsFromCLI
--- PASS: TestFilterAKSRecommendationsFromCLI (0.00s)
=== RUN   TestIsAKSRelatedCLI
--- PASS: TestIsAKSRelatedCLI (0.00s)
# ... all 10 tests pass
PASS
ok      github.com/Azure/aks-mcp/internal/azure/advisor
```

## 🔒 **Security & Access Control**

- **Readonly Mode**: ✅ Advisor tool works in readonly mode (listing recommendations is read-only)
- **Input Validation**: ✅ All parameters are validated and sanitized
- **Access Control**: ✅ Integrated with existing security validation framework
- **Command Injection Protection**: ✅ Uses validated Azure CLI executor

## 📝 **Logging & Monitoring**

The tool includes comprehensive logging with the `[ADVISOR]` prefix:
```
[ADVISOR] Handling operation: list
[ADVISOR] Listing recommendations for subscription: xxx, resource_group: yyy
[ADVISOR] Executing command: az advisor recommendation list --subscription xxx
[ADVISOR] Found 15 total recommendations
[ADVISOR] Found 3 AKS-related recommendations
[ADVISOR] After severity filter: 1 recommendations
[ADVISOR] Returning 1 recommendation summaries
```

## 🔄 **Integration Points**

1. **MCP Server**: Registered as `az_advisor_recommendation` tool
2. **VS Code Extension**: Compatible with VS Code MCP extension
3. **Azure CLI**: Uses existing Azure CLI integration for authentication
4. **Security Layer**: Integrated with access level validation

## ✅ **Verification Steps**

1. **Build Success**: ✅ Project builds without errors
2. **Test Suite**: ✅ All 10 advisor tests pass
3. **Integration Test**: ✅ Tool works with VS Code MCP extension
4. **Security Validation**: ✅ Works in readonly mode as expected
5. **Logging Verification**: ✅ Logs are written to debug file and can be monitored

## 🎯 **Benefits**

1. **Enhanced AI Assistant Capabilities**: AI assistants can now provide AKS optimization recommendations
2. **Proactive Monitoring**: Enables proactive identification of AKS issues and optimizations
3. **Cost Optimization**: Helps identify cost-saving opportunities in AKS clusters
4. **Security Improvements**: Surfaces security-related recommendations for AKS resources
5. **Operational Excellence**: Provides actionable insights for AKS cluster management

## 🔮 **Future Enhancements**

- Integration with Azure Resource Graph for advanced querying
- Support for custom recommendation filtering rules
- Integration with Azure Policy for automated remediation
- Enhanced reporting with trend analysis

---

**Ready for Review** ✅

This PR is ready for review and testing. All tests pass, documentation is complete, and the tool has been verified to work with the VS Code MCP extension in a real environment.
