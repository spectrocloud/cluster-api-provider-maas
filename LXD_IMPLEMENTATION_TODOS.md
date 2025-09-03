# LXD Implementation TODOs

This document lists the TODOs in the LXD implementation that need to be addressed for production readiness.

## Fixed TODOs

1. **MAAS Client Identity** (Priority: High) ✅
   - **Location**: `pkg/maas/scope/cluster.go`
   - **Description**: The MAAS client identity was hardcoded with dummy values.
   - **Fix**: Updated to read from a Kubernetes secret, with fallback to environment variables or default values.

## Remaining TODOs

1. **Context Management** (Priority: Medium)
   - **Location**: Multiple files
   - **Description**: Replace `context.TODO()` with proper context handling.
   - **Files**:
     - `pkg/maas/scope/cluster.go`
     - `pkg/maas/scope/machine.go`
     - `pkg/maas/machine/machine.go`
   - **Recommendation**: Use context passed from the controller's Reconcile method.

2. **Machine Hostname Management** (Priority: Medium)
   - **Location**: `pkg/maas/machine/machine.go`
   - **Description**: Need to revisit if we need to set the hostname or not.
   - **Comment**: `// TODO need to revisit if we need to set the hostname OR not`
   - **Recommendation**: Evaluate whether hostname management is necessary or if it should be handled by MAAS.

3. **Active Status Check** (Priority: Low)
   - **Location**: `pkg/maas/scope/cluster.go`
   - **Description**: Comment questions if active status is needed.
   - **Comment**: `// TODO need active?`
   - **Recommendation**: Evaluate if the active status check is necessary for the implementation.

## Testing Guidelines

When testing the LXD integration, keep in mind the following:

1. **MAAS Credentials**: Create a Kubernetes secret named `maas-credentials` in the same namespace as your MaasCluster with the following keys:
   - `url`: The MAAS API URL (e.g., `http://maas.example.com:5240/MAAS`)
   - `token`: The MAAS API token

2. **Environment Variables**: If you can't use a Kubernetes secret, set the following environment variables:
   - `MAAS_API_URL`: The MAAS API URL
   - `MAAS_API_TOKEN`: The MAAS API token

3. **LXD Configuration**: Ensure your MaasCluster has the appropriate LXD configuration:
   ```yaml
   apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
   kind: MaasCluster
   metadata:
     name: my-cluster
   spec:
     lxdControlPlaneCluster: true
     lxdConfig:
       enabled: true
       storageBackend: "zfs"
       storageSize: "50"
       networkBridge: "br0"
       resourcePool: "default"
   ```

## Next Steps

1. Address the remaining TODOs in order of priority.
2. Add comprehensive unit tests for the LXD integration.
3. Create end-to-end tests to verify the LXD integration works as expected.
4. Document the LXD integration in the project's documentation. 