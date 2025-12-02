# AKS-MCP Performance Analysis Report

**Date:** 2025-12-02T22:08:24+08:00

## Overall Performance

| Metric | Value |
|--------|-------|
| Total Tests | 1 |
| Total Execution Time | 40.955653718s |
| Avg Test Duration | 40.955653718s |
| Avg Tool Call Duration | 1.050146617s |
| Avg Tool Calls Per Test | 5.0 |
| Avg LLM Iterations | 5.0 |

## Slowest Tool Calls (Top 10)

| Test ID | Tool Name | Duration | Arguments |
|---------|-----------|----------|----------|
| 02_misconfigured_ingress_class | call_kubectl | 1.90338441s | args=describe ingress web-ingress -n app-25 |
| 02_misconfigured_ingress_class | call_kubectl | 1.238030721s | args=get pods -A |
| 02_misconfigured_ingress_class | call_kubectl | 988.482852ms | args=get ingress web-ingress -n app-25 |
| 02_misconfigured_ingress_class | call_kubectl | 983.112964ms | args=get pods -A -l app.kubernetes.io/name=ingr... |
| 02_misconfigured_ingress_class | call_kubectl | 137.722138ms | args=get pods -A | grep ingress |

## Tool Performance Statistics

| Tool Name | Call Count | Avg Duration | Max Duration | Total Time |
|-----------|------------|--------------|--------------|------------|
| call_kubectl | 5 | 1.050146617s | 1.90338441s | 5.250733085s |

## Performance Recommendations

1. No significant performance issues detected. All metrics are within acceptable ranges.
