/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// ClusterFinalizer allows MaasClusterReconciler to clean up resources associated with MaasCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "maascluster.infrastructure.cluster.x-k8s.io"
)

// MaasClusterSpec defines the desired state of MaasCluster
type MaasClusterSpec struct {
	// DNSDomain configures the MaaS domain to create the cluster on (e.g maas)
	// +kubebuilder:validation:MinLength=1
	DNSDomain string `json:"dnsDomain"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint APIEndpoint `json:"controlPlaneEndpoint"`

	// FailureDomains are not usually defined on the spec.
	// but useful for MaaS since we can limit the domains to these
	// +optional
	FailureDomains []string `json:"failureDomains,omitempty"`

	// LXDConfig contains the configuration for LXD hosts
	// +optional
	LXDConfig *LXDConfig `json:"lxdConfig,omitempty"`
}

// LXDConfig contains the configuration for LXD hosts
type LXDConfig struct {
	// Enabled specifies whether to enable LXD VM support
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// ResourcePool specifies the MAAS resource pool to use for LXD VM hosts
	// +optional
	ResourcePool string `json:"resourcePool,omitempty"`

	// Zone specifies the MAAS availability zone to register LXD VM hosts in
	// +optional
	Zone string `json:"zone,omitempty"`

	// StorageBackend specifies the storage backend to use (zfs, dir, etc.)
	// +kubebuilder:default=zfs
	// +optional
	StorageBackend string `json:"storageBackend,omitempty"`

	// StorageSize specifies the size of the storage pool in GB
	// +kubebuilder:default="50"
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// NICType selects the LXD NIC type (bridge or macvlan)
	// +kubebuilder:validation:Enum=bridge;macvlan
	// +optional
	NICType string `json:"nicType,omitempty"`

	// NICParent selects the parent interface or bridge for the NIC
	// +optional
	NICParent string `json:"nicParent,omitempty"`

	// NetworkBridge specifies the network bridge to use (legacy, prefer NICParent)
	// +optional
	NetworkBridge string `json:"networkBridge,omitempty"`

	// ImageRepository specifies the remote server configuration for images
	// +optional
	ImageRepository *ImageRepositoryConfig `json:"imageRepository,omitempty"`

	// HostOS specifies the host OS configuration for LXD hosts
	// +optional
	HostOS *HostOSConfig `json:"hostOS,omitempty"`

	// SecurityConfig specifies security settings for LXD hosts
	// +optional
	SecurityConfig *SecurityConfig `json:"securityConfig,omitempty"`

	// SkipNetworkUpdate specifies whether to skip updating existing networks
	// +kubebuilder:default=true
	// +optional
	SkipNetworkUpdate *bool `json:"skipNetworkUpdate,omitempty"`
}

// ImageRepositoryConfig specifies the image repository configuration
type ImageRepositoryConfig struct {
	// URL specifies the remote server URL for images
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Protocol specifies the protocol to use (https, http)
	// +kubebuilder:default=https
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Credentials specifies the credentials for the image repository
	// +optional
	Credentials *ImageRepositoryCredentials `json:"credentials,omitempty"`

	// Certificates specifies SSL certificates for the repository
	// +optional
	Certificates *ImageRepositoryCertificates `json:"certificates,omitempty"`
}

// ImageRepositoryCredentials specifies credentials for image repository
type ImageRepositoryCredentials struct {
	// SecretName specifies the Kubernetes secret containing credentials
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`

	// UsernameKey specifies the key in the secret for username
	// +kubebuilder:default=username
	// +optional
	UsernameKey string `json:"usernameKey,omitempty"`

	// PasswordKey specifies the key in the secret for password
	// +kubebuilder:default=password
	// +optional
	PasswordKey string `json:"passwordKey,omitempty"`
}

// ImageRepositoryCertificates specifies SSL certificates for image repository
type ImageRepositoryCertificates struct {
	// SecretName specifies the Kubernetes secret containing certificates
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`

	// CACertKey specifies the key in the secret for CA certificate
	// +kubebuilder:default=ca.crt
	// +optional
	CACertKey string `json:"caCertKey,omitempty"`

	// ClientCertKey specifies the key in the secret for client certificate
	// +kubebuilder:default=client.crt
	// +optional
	ClientCertKey string `json:"clientCertKey,omitempty"`

	// ClientKeyKey specifies the key in the secret for client key
	// +kubebuilder:default=client.key
	// +optional
	ClientKeyKey string `json:"clientKeyKey,omitempty"`
}

// HostOSConfig specifies the host OS configuration
type HostOSConfig struct {
	// AutoUpdate specifies whether to enable automatic OS updates
	// +kubebuilder:default=false
	// +optional
	AutoUpdate *bool `json:"autoUpdate,omitempty"`

	// UpdateSchedule specifies the schedule for OS updates (cron format)
	// +optional
	UpdateSchedule string `json:"updateSchedule,omitempty"`

	// MaintenanceWindow specifies the maintenance window for updates
	// +optional
	MaintenanceWindow *MaintenanceWindow `json:"maintenanceWindow,omitempty"`

	// RollingUpdate specifies rolling update configuration
	// +optional
	RollingUpdate *RollingUpdateConfig `json:"rollingUpdate,omitempty"`
}

// MaintenanceWindow specifies a maintenance window
type MaintenanceWindow struct {
	// StartTime specifies the start time (HH:MM format)
	// +kubebuilder:validation:Required
	StartTime string `json:"startTime"`

	// Duration specifies the duration in minutes
	// +kubebuilder:validation:Minimum=30
	// +kubebuilder:validation:Maximum=480
	// +kubebuilder:default=120
	// +optional
	Duration int32 `json:"duration,omitempty"`

	// Days specifies the days of the week (1=Monday, 7=Sunday)
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=7
	// +optional
	Days []int32 `json:"days,omitempty"`
}

// RollingUpdateConfig specifies rolling update configuration
type RollingUpdateConfig struct {
	// MaxUnavailable specifies the maximum number of unavailable nodes
	// +kubebuilder:default=1
	// +optional
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`

	// MaxSurge specifies the maximum number of extra nodes
	// +kubebuilder:default=1
	// +optional
	MaxSurge *int32 `json:"maxSurge,omitempty"`

	// MinReadySeconds specifies the minimum ready seconds
	// +kubebuilder:default=300
	// +optional
	MinReadySeconds *int32 `json:"minReadySeconds,omitempty"`
}

// SecurityConfig specifies security settings for LXD hosts
type SecurityConfig struct {
	// NetworkIsolation specifies network isolation settings
	// +optional
	NetworkIsolation *NetworkIsolationConfig `json:"networkIsolation,omitempty"`

	// ResourceIsolation specifies resource isolation settings
	// +optional
	ResourceIsolation *ResourceIsolationConfig `json:"resourceIsolation,omitempty"`

	// MultiTenancy specifies multi-tenancy settings
	// +optional
	MultiTenancy *MultiTenancyConfig `json:"multiTenancy,omitempty"`
}

// NetworkIsolationConfig specifies network isolation settings
type NetworkIsolationConfig struct {
	// EnableVLAN specifies whether to enable VLAN isolation
	// +kubebuilder:default=false
	// +optional
	EnableVLAN *bool `json:"enableVLAN,omitempty"`

	// DefaultVLAN specifies the default VLAN ID
	// +optional
	DefaultVLAN *int32 `json:"defaultVLAN,omitempty"`

	// NetworkPolicies specifies network policies for VMs
	// +optional
	NetworkPolicies []NetworkPolicy `json:"networkPolicies,omitempty"`
}

// NetworkPolicy specifies a network policy
type NetworkPolicy struct {
	// Name specifies the policy name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type specifies the policy type (allow, deny, isolate)
	// +kubebuilder:validation:Enum=allow;deny;isolate
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Rules specifies the network rules
	// +optional
	Rules []NetworkRule `json:"rules,omitempty"`
}

// NetworkRule specifies a network rule
type NetworkRule struct {
	// Protocol specifies the protocol (tcp, udp, icmp)
	// +kubebuilder:validation:Enum=tcp;udp;icmp
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Port specifies the port or port range
	// +optional
	Port string `json:"port,omitempty"`

	// Source specifies the source address or range
	// +optional
	Source string `json:"source,omitempty"`

	// Destination specifies the destination address or range
	// +optional
	Destination string `json:"destination,omitempty"`
}

// ResourceIsolationConfig specifies resource isolation settings
type ResourceIsolationConfig struct {
	// CPUQuota specifies CPU quota settings
	// +optional
	CPUQuota *CPUQuotaConfig `json:"cpuQuota,omitempty"`

	// MemoryQuota specifies memory quota settings
	// +optional
	MemoryQuota *MemoryQuotaConfig `json:"memoryQuota,omitempty"`

	// StorageQuota specifies storage quota settings
	// +optional
	StorageQuota *StorageQuotaConfig `json:"storageQuota,omitempty"`
}

// CPUQuotaConfig specifies CPU quota settings
type CPUQuotaConfig struct {
	// DefaultLimit specifies the default CPU limit per VM
	// +kubebuilder:default=4
	// +optional
	DefaultLimit *int32 `json:"defaultLimit,omitempty"`

	// MaxLimit specifies the maximum CPU limit per VM
	// +kubebuilder:default=16
	// +optional
	MaxLimit *int32 `json:"maxLimit,omitempty"`

	// BurstLimit specifies the CPU burst limit
	// +optional
	BurstLimit *int32 `json:"burstLimit,omitempty"`
}

// MemoryQuotaConfig specifies memory quota settings
type MemoryQuotaConfig struct {
	// DefaultLimit specifies the default memory limit per VM (in MB)
	// +kubebuilder:default=8192
	// +optional
	DefaultLimit *int32 `json:"defaultLimit,omitempty"`

	// MaxLimit specifies the maximum memory limit per VM (in MB)
	// +kubebuilder:default=32768
	// +optional
	MaxLimit *int32 `json:"maxLimit,omitempty"`

	// SwapLimit specifies the swap limit (in MB)
	// +optional
	SwapLimit *int32 `json:"swapLimit,omitempty"`
}

// StorageQuotaConfig specifies storage quota settings
type StorageQuotaConfig struct {
	// DefaultLimit specifies the default storage limit per VM (in GB)
	// +kubebuilder:default=50
	// +optional
	DefaultLimit *int32 `json:"defaultLimit,omitempty"`

	// MaxLimit specifies the maximum storage limit per VM (in GB)
	// +kubebuilder:default=500
	// +optional
	MaxLimit *int32 `json:"maxLimit,omitempty"`
}

// MultiTenancyConfig specifies multi-tenancy settings
type MultiTenancyConfig struct {
	// EnableIsolation specifies whether to enable tenant isolation
	// +kubebuilder:default=false
	// +optional
	EnableIsolation *bool `json:"enableIsolation,omitempty"`

	// TenantLabels specifies labels for tenant identification
	// +optional
	TenantLabels []string `json:"tenantLabels,omitempty"`

	// ResourceSharing specifies resource sharing policies
	// +optional
	ResourceSharing *ResourceSharingConfig `json:"resourceSharing,omitempty"`
}

// ResourceSharingConfig specifies resource sharing policies
type ResourceSharingConfig struct {
	// CPUSharing specifies CPU sharing policy
	// +kubebuilder:validation:Enum=equal;proportional;weighted
	// +kubebuilder:default=equal
	// +optional
	CPUSharing string `json:"cpuSharing,omitempty"`

	// MemorySharing specifies memory sharing policy
	// +kubebuilder:validation:Enum=equal;proportional;weighted
	// +kubebuilder:default=equal
	// +optional
	MemorySharing string `json:"memorySharing,omitempty"`

	// StorageSharing specifies storage sharing policy
	// +kubebuilder:validation:Enum=equal;proportional;weighted
	// +kubebuilder:default=equal
	// +optional
	StorageSharing string `json:"storageSharing,omitempty"`
}

// InfrastructureClusterRef references an infrastructure cluster
type InfrastructureClusterRef struct {
	// Name is the name of the infrastructure cluster
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace is the namespace of the infrastructure cluster
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// MaasClusterStatus defines the observed state of MaasCluster
type MaasClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Ready denotes that the maas cluster (infrastructure) is ready.
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// Network represents the network
	Network Network `json:"network,omitempty"`

	// FailureDomains don't mean much in CAPMAAS since it's all local, but we can see how the rest of cluster API
	// will use this if we populate it.
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`

	// Conditions defines current service state of the MaasCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// Network encapsulates the Cluster Network
type Network struct {
	// DNSName is the Kubernetes api server name
	DNSName string `json:"dnsName,omitempty"`
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
type APIEndpoint struct {

	// Host is the hostname on which the API server is serving.
	Host string `json:"host"`

	// Port is the port on which the API server is serving.
	Port int `json:"port"`
}

// IsZero returns true if both host and port are zero values.
func (in APIEndpoint) IsZero() bool {
	return in.Host == "" && in.Port == 0
}

// +kubebuilder:resource:path=maasclusters,scope=Namespaced,categories=cluster-api
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// MaasCluster is the Schema for the maasclusters API
type MaasCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaasClusterSpec   `json:"spec,omitempty"`
	Status MaasClusterStatus `json:"status,omitempty"`
}

func (in *MaasCluster) GetConditions() clusterv1.Conditions {
	return in.Status.Conditions
}

func (in *MaasCluster) SetConditions(conditions clusterv1.Conditions) {
	in.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// MaasClusterList contains a list of MaasCluster
type MaasClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaasCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MaasCluster{}, &MaasClusterList{})
}
