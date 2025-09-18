# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Kubernetes controller for managing cluster claims and cluster pools in OpenShift/ACM (Advanced Cluster Management) environments. The controller consists of two main binaries that work together to automate cluster provisioning through Hive cluster pools.

## Development Commands

### Building
```bash
# Compile both binaries locally
make compile

# Build Docker image
make build

# Build for Konflux CI/CD
make compile-konflux

# Build and push Docker image
make push
```

### Testing
```bash
# Run unit tests for all controllers
make unit-tests

# Run tests for specific controllers
GOFLAGS="" go test -timeout 120s -v -short ./controllers/clusterclaims
GOFLAGS="" go test -timeout 120s -v -short ./controllers/clusterpools
GOFLAGS="" go test -timeout 120s -v -short ./controllers/managedcluster
```

### Deployment
```bash
# Deploy to Kubernetes cluster (requires kubeadmin)
oc apply -k ./deploy
```

## Architecture

The system consists of two main controller binaries:

### 1. ClusterClaims Controller (`cmd/clusterclaims/main.go`)
- **Purpose**: Watches `ClusterClaim` resources and creates corresponding `ManagedCluster` resources
- **Controllers**:
  - `ClusterClaimsReconciler` - handles ClusterClaim lifecycle
  - `ManagedClusterReconciler` - manages ManagedCluster resources
- **Metrics Port**: 9443
- **Leader Election ID**: `clusterclaims-controller.open-cluster-management.io`

### 2. ClusterPools Controller (`cmd/clusterpools/main.go`)
- **Purpose**: Manages cluster pool deletion and cleanup operations
- **Controllers**:
  - `ClusterPoolsReconciler` - handles cluster pool lifecycle and namespace cleanup
- **Metrics Port**: 8383
- **Leader Election ID**: `clusterpools-controller.open-cluster-management.io`

### Key Dependencies
- **Hive APIs**: For cluster pool and cluster provisioning management
- **Open Cluster Management APIs**: For ManagedCluster resources
- **Controller Runtime**: Standard Kubernetes controller framework with leader election

### Controller Structure
- `controllers/clusterclaims/` - ClusterClaim reconciliation logic
- `controllers/clusterpools/` - ClusterPool management and namespace cleanup
- `controllers/managedcluster/` - ManagedCluster lifecycle management

## Key Features

### Namespace Auto-Cleanup
When creating namespaces for cluster pools, add the label `open-cluster-management.io/managed-by: clusterpools` to enable automatic namespace deletion when the last cluster pool is removed.

### Leader Election
Both controllers support leader election with configurable timeouts:
- Lease Duration: 137s (default)
- Renew Deadline: 107s (default)
- Retry Period: 26s (default)

### Logging
- Uses zap logger with configurable levels
- Change `zapcore.InfoLevel` to `zapcore.DebugLevel` in main.go for debug logging
- CGO is enabled for both builds

## Workflow
1. User creates a `ClusterClaim` resource specifying a cluster pool
2. ClusterClaims controller watches for new claims and provisions clusters from the specified pool
3. Upon successful provisioning, a `ManagedCluster` resource is created
4. ClusterPools controller handles cleanup operations when pools are deleted