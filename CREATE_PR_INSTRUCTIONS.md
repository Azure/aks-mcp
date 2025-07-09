# GitHub Pull Request Instructions

## 📋 **Pull Request Details**

**Title:** `feat: Add Azure Advisor recommendation tool for AKS clusters with logging and testing`

**Base Branch:** `main`
**Compare Branch:** `feature/azure-diagnostics-prompts`

## 🚀 **How to Create the Pull Request**

### **Option 1: GitHub Web Interface**
1. Go to: https://github.com/Azure/aks-mcp
2. Click "Pull requests" tab
3. Click "New pull request"
4. Set base: `main` <- compare: `feature/azure-diagnostics-prompts`
5. Copy the title above
6. Copy the description from `PR_DESCRIPTION.md` (created above)
7. Add labels: `enhancement`, `feature`, `aks`, `advisor`
8. Click "Create pull request"

### **Option 2: GitHub CLI (if available)**
```bash
# Install GitHub CLI first if not available
gh pr create \
  --title "feat: Add Azure Advisor recommendation tool for AKS clusters with logging and testing" \
  --body-file PR_DESCRIPTION.md \
  --base main \
  --head feature/azure-diagnostics-prompts \
  --label enhancement,feature,aks,advisor
```

### **Option 3: Direct GitHub URL**
Navigate to:
```
https://github.com/Azure/aks-mcp/compare/main...feature/azure-diagnostics-prompts
```

## 📄 **Files to Reference**
- **Detailed Description:** `PULL_REQUEST.md` (comprehensive documentation)
- **GitHub PR Description:** `PR_DESCRIPTION.md` (concise version for GitHub)

## 🏷️ **Suggested Labels**
- `enhancement`
- `feature` 
- `aks`
- `advisor`
- `logging`
- `testing`

## 👥 **Suggested Reviewers**
- Team leads familiar with AKS
- Azure CLI integration experts
- MCP server maintainers
- Security/access control reviewers

## ✅ **Pre-submission Checklist**
- [x] All tests pass (10/10 advisor tests)
- [x] Project builds successfully
- [x] Code follows project conventions
- [x] Comprehensive logging implemented
- [x] Security validation integrated
- [x] Documentation complete
- [x] VS Code MCP extension compatibility verified
- [x] ResourceID field properly implemented
- [x] Error handling comprehensive

## 📊 **Commit Summary**
```
9370274 fix: Update Azure Advisor recommendation tool with logging and resourceID field
0e47372 fix: resolve MCP tool parameter validation error for Azure Advisor  
06c90db docs: add Azure Advisor tool usage documentation
11c71fc feat: implement Azure Advisor recommendations for AKS clusters
439e603 chore: remove pull request template and exe files from tracking
```

**Total commits:** 5
**Files changed:** 6 files (+247 insertions, -104 deletions)

The pull request is ready to be created! 🚀
