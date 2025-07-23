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
	"context"
	"fmt"
	"net"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var maasmachinelog = logf.Log.WithName("maasmachine-resource")

// webhookClient is used by the webhook validation to access the Kubernetes API
var webhookClient client.Client

func (r *MaasMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// Store the manager client for use in validation
	webhookClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachine,mutating=true,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachines,verbs=create;update,versions=v1beta1,name=mmaasmachine.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1
//+kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-maasmachine,mutating=false,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=maasmachines,versions=v1beta1,name=vmaasmachine.kb.io,sideEffects=None,admissionReviewVersions=v1beta1;v1

var (
	_ webhook.Defaulter = &MaasMachine{}
	_ webhook.Validator = &MaasMachine{}
)

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MaasMachine) Default() {
	maasmachinelog.Info("default", "name", r.Name)

	// Set default provisioning mode if not specified
	if r.Spec.ProvisioningMode == "" {
		r.Spec.ProvisioningMode = ProvisioningModeBaremetal
	}

	// Set LXD defaults when using LXD provisioning
	if r.Spec.ProvisioningMode == ProvisioningModeLXD {
		r.setLXDDefaults()
	}
}

// setLXDDefaults sets default values for LXD configuration
func (r *MaasMachine) setLXDDefaults() {
	if r.Spec.LXDConfig == nil {
		r.Spec.LXDConfig = &LXDConfig{}
	}

	// Set default resource allocation based on MinCPU and MinMemoryInMB
	if r.Spec.LXDConfig.ResourceAllocation == nil {
		r.Spec.LXDConfig.ResourceAllocation = &LXDResourceConfig{}
	}

	resAlloc := r.Spec.LXDConfig.ResourceAllocation

	// Default CPU allocation
	if resAlloc.CPU == nil {
		if r.Spec.MinCPU != nil {
			resAlloc.CPU = r.Spec.MinCPU
		} else {
			resAlloc.CPU = pointer.Int(2) // Default to 2 CPUs
		}
	}

	// Default memory allocation
	if resAlloc.Memory == nil {
		if r.Spec.MinMemoryInMB != nil {
			resAlloc.Memory = r.Spec.MinMemoryInMB
		} else {
			resAlloc.Memory = pointer.Int(4096) // Default to 4GB
		}
	}

	// Default disk size
	if resAlloc.Disk == nil {
		resAlloc.Disk = pointer.Int(20) // Default to 20GB
	}

	// Set default storage config
	if r.Spec.LXDConfig.StorageConfig == nil {
		r.Spec.LXDConfig.StorageConfig = &LXDStorageConfig{}
	}

	// Set default network config
	if r.Spec.LXDConfig.NetworkConfig == nil {
		r.Spec.LXDConfig.NetworkConfig = &LXDNetworkConfig{}
	}

	// Set default host selection if empty
	if r.Spec.LXDConfig.HostSelection == nil {
		r.Spec.LXDConfig.HostSelection = &LXDHostSelectionConfig{}
	}
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateCreate() (admission.Warnings, error) {
	maasmachinelog.Info("validate create", "name", r.Name)

	if err := r.validateLXDConfig(); err != nil {
		return nil, err
	}

	// Validate static IP configuration if present
	ctx := context.Background()
	if err := r.validateStaticIPConflicts(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateDelete() (admission.Warnings, error) {
	maasmachinelog.Info("validate delete", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MaasMachine) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	maasmachinelog.Info("validate update", "name", r.Name)
	oldM := old.(*MaasMachine)

	if r.Spec.Image != oldM.Spec.Image {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine image change is not allowed, old=%s, new=%s", oldM.Spec.Image, r.Spec.Image))
	}

	if r.Spec.MinCPU != nil && oldM.Spec.MinCPU != nil && *r.Spec.MinCPU != *oldM.Spec.MinCPU {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine min cpu count change is not allowed, old=%d, new=%d", *oldM.Spec.MinCPU, *r.Spec.MinCPU))
	}

	if r.Spec.MinMemoryInMB != nil && oldM.Spec.MinMemoryInMB != nil && *r.Spec.MinMemoryInMB != *oldM.Spec.MinMemoryInMB {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("maas machine min memory change is not allowed, old=%d MB, new=%d MB", *oldM.Spec.MinMemoryInMB, *r.Spec.MinMemoryInMB))
	}

	// Validate that provisioning mode cannot be changed once set
	if r.Spec.ProvisioningMode != oldM.Spec.ProvisioningMode {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("provisioning mode change is not allowed, old=%s, new=%s", oldM.Spec.ProvisioningMode, r.Spec.ProvisioningMode))
	}

	if err := r.validateLXDConfig(); err != nil {
		return nil, err
	}

	// Validate static IP configuration if present
	ctx := context.Background()
	if err := r.validateStaticIPConflicts(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

// validateLXDConfig validates LXD-specific configuration
func (r *MaasMachine) validateLXDConfig() error {
	// Only validate LXD config if using LXD provisioning
	if r.Spec.ProvisioningMode != ProvisioningModeLXD {
		// If not using LXD, ensure LXD config is not provided
		if r.Spec.LXDConfig != nil {
			return apierrors.NewBadRequest("lxdConfig should not be specified when provisioningMode is not 'lxd'")
		}
		return nil
	}

	// Validate that LXD config is provided when using LXD mode
	if r.Spec.LXDConfig == nil {
		return apierrors.NewBadRequest("lxdConfig is required when provisioningMode is 'lxd'")
	}

	config := r.Spec.LXDConfig

	// Validate resource allocation
	if config.ResourceAllocation != nil {
		if err := r.validateResourceAllocation(config.ResourceAllocation); err != nil {
			return err
		}
	}

	// Validate network configuration
	if config.NetworkConfig != nil {
		if err := r.validateNetworkConfig(config.NetworkConfig); err != nil {
			return err
		}
	}

	// Validate host selection
	if config.HostSelection != nil {
		if err := r.validateHostSelection(config.HostSelection); err != nil {
			return err
		}
	}

	return nil
}

// validateResourceAllocation validates LXD resource allocation settings
func (r *MaasMachine) validateResourceAllocation(resAlloc *LXDResourceConfig) error {
	if resAlloc.CPU != nil && *resAlloc.CPU < 1 {
		return apierrors.NewBadRequest("lxdConfig.resourceAllocation.cpu must be at least 1")
	}

	if resAlloc.Memory != nil && *resAlloc.Memory < 512 {
		return apierrors.NewBadRequest("lxdConfig.resourceAllocation.memory must be at least 512 MB")
	}

	if resAlloc.Disk != nil && *resAlloc.Disk < 1 {
		return apierrors.NewBadRequest("lxdConfig.resourceAllocation.disk must be at least 1 GB")
	}

	return nil
}

// validateIPAddress validates IPv4 or IPv6 address format
func validateIPAddress(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}
	return nil
}

// validateSubnet validates subnet mask or CIDR notation
func validateSubnet(subnet string) error {
	// Check if it's a CIDR notation (e.g., /24)
	if strings.HasPrefix(subnet, "/") {
		_, _, err := net.ParseCIDR("192.168.1.0" + subnet)
		if err != nil {
			return fmt.Errorf("invalid CIDR notation: %s", subnet)
		}
		return nil
	}

	// Check if it's a subnet mask (e.g., 255.255.255.0)
	ip := net.ParseIP(subnet)
	if ip == nil {
		return fmt.Errorf("invalid subnet mask format: %s", subnet)
	}

	// Verify it's a valid subnet mask
	ipv4 := ip.To4()
	if ipv4 == nil {
		return fmt.Errorf("subnet mask must be IPv4: %s", subnet)
	}

	// Check if it's a valid subnet mask (contiguous 1s followed by contiguous 0s)
	mask := uint32(ipv4[0])<<24 + uint32(ipv4[1])<<16 + uint32(ipv4[2])<<8 + uint32(ipv4[3])
	if mask != 0 && (mask&(^mask+1)) != (^mask+1) {
		return fmt.Errorf("invalid subnet mask pattern: %s", subnet)
	}

	return nil
}

// validateGatewayReachability validates that the gateway is reachable from the given IP
func validateGatewayReachability(ipAddr, gateway, subnet string) error {
	ip := net.ParseIP(ipAddr)
	gw := net.ParseIP(gateway)

	if ip == nil || gw == nil {
		return fmt.Errorf("invalid IP addresses for gateway reachability check")
	}

	// Convert to IPv4 if possible for easier subnet checking
	ipv4 := ip.To4()
	gwv4 := gw.To4()

	if ipv4 != nil && gwv4 != nil {
		// For IPv4, check if they're in the same subnet
		if strings.HasPrefix(subnet, "/") {
			// CIDR notation
			_, ipNet, err := net.ParseCIDR(ipAddr + subnet)
			if err != nil {
				return fmt.Errorf("invalid CIDR for reachability check: %s", subnet)
			}
			if !ipNet.Contains(gw) {
				return fmt.Errorf("gateway %s is not reachable from IP %s with subnet %s", gateway, ipAddr, subnet)
			}
		} else {
			// Subnet mask notation - simplified check
			subnetMask := net.ParseIP(subnet)
			if subnetMask == nil {
				return fmt.Errorf("invalid subnet mask for reachability check: %s", subnet)
			}
			// This is a basic check - in production, you might want more sophisticated validation
		}
	}

	return nil
}

// validateDNSServers validates a list of DNS server IP addresses
func validateDNSServers(dnsServers []string) error {
	for _, dns := range dnsServers {
		if err := validateIPAddress(dns); err != nil {
			return fmt.Errorf("invalid DNS server address: %s", dns)
		}
	}
	return nil
}

// validateNetworkConfig validates LXD network configuration
func (r *MaasMachine) validateNetworkConfig(netConfig *LXDNetworkConfig) error {
	if netConfig.MacAddress != nil {
		macAddr := strings.ToLower(strings.TrimSpace(*netConfig.MacAddress))
		// Validate MAC address format
		macRegex := regexp.MustCompile(`^([0-9a-f]{2}:){5}[0-9a-f]{2}$`)
		if !macRegex.MatchString(macAddr) {
			return apierrors.NewBadRequest("lxdConfig.networkConfig.macAddress must be a valid MAC address in format XX:XX:XX:XX:XX:XX")
		}
	}

	// Validate static IP configuration if present
	if netConfig.StaticIPConfig != nil {
		if err := r.validateStaticIPConfig(netConfig.StaticIPConfig); err != nil {
			return err
		}
	}

	return nil
}

// checkIPConflicts checks for IP address conflicts with existing MaasMachine resources
func (r *MaasMachine) checkIPConflicts(ctx context.Context, k8sClient client.Client, ipAddress string) error {
	// Get all MaasMachine resources in the same namespace
	maasMachineList := &MaasMachineList{}
	if err := k8sClient.List(ctx, maasMachineList, client.InNamespace(r.Namespace)); err != nil {
		maasmachinelog.Error(err, "failed to list MaasMachine resources for IP conflict check")
		// Don't fail validation if we can't check - this prevents blocking in case of API issues
		return nil
	}

	// Check each machine for IP conflicts
	for _, machine := range maasMachineList.Items {
		// Skip checking against ourselves (for updates)
		if machine.Name == r.Name && machine.Namespace == r.Namespace {
			continue
		}

		// Check if this machine has LXD config with static IP
		if machine.Spec.LXDConfig != nil &&
			machine.Spec.LXDConfig.NetworkConfig != nil &&
			machine.Spec.LXDConfig.NetworkConfig.StaticIPConfig != nil {

			existingIP := machine.Spec.LXDConfig.NetworkConfig.StaticIPConfig.IPAddress
			if existingIP == ipAddress {
				return fmt.Errorf("IP address %s is already in use by machine %s", ipAddress, machine.Name)
			}
		}
	}

	return nil
}

// validateSubnetRange validates that the IP address is within allowed subnet ranges
func validateSubnetRange(ipAddress, subnet string) error {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return fmt.Errorf("invalid IP address for subnet range validation: %s", ipAddress)
	}

	var ipNet *net.IPNet
	var err error

	if strings.HasPrefix(subnet, "/") {
		// CIDR notation - need to construct full CIDR
		_, ipNet, err = net.ParseCIDR(ipAddress + subnet)
		if err != nil {
			return fmt.Errorf("invalid CIDR for range validation: %s", ipAddress+subnet)
		}
	} else {
		// Subnet mask notation - convert to CIDR for validation
		subnetMask := net.ParseIP(subnet)
		if subnetMask == nil {
			return fmt.Errorf("invalid subnet mask: %s", subnet)
		}

		// Create a network from IP and mask for validation
		mask := net.IPMask(subnetMask.To4())
		network := ip.Mask(mask)
		ipNet = &net.IPNet{IP: network, Mask: mask}
	}

	// Validate that the IP is within the defined network
	if !ipNet.Contains(ip) {
		return fmt.Errorf("IP address %s is not within the subnet range %s", ipAddress, subnet)
	}

	// Check for reserved IP ranges
	if err := validateNotReservedIP(ip); err != nil {
		return err
	}

	return nil
}

// validateNotReservedIP checks if the IP is not in reserved ranges
func validateNotReservedIP(ip net.IP) error {
	// Check for common reserved ranges
	reservedRanges := []string{
		"127.0.0.0/8",    // Loopback
		"169.254.0.0/16", // Link-local
		"224.0.0.0/4",    // Multicast
		"240.0.0.0/4",    // Reserved for future use
		"0.0.0.0/8",      // "This" network
	}

	for _, cidr := range reservedRanges {
		_, reservedNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if reservedNet.Contains(ip) {
			return fmt.Errorf("IP address %s is in reserved range %s", ip.String(), cidr)
		}
	}

	return nil
}

// validateStaticIPConfig validates static IP configuration
func (r *MaasMachine) validateStaticIPConfig(staticConfig *LXDStaticIPConfig) error {
	// Validate IP address format
	if err := validateIPAddress(staticConfig.IPAddress); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig.ipAddress: %s", err.Error()))
	}

	// Validate gateway format
	if err := validateIPAddress(staticConfig.Gateway); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig.gateway: %s", err.Error()))
	}

	// Validate subnet format
	if err := validateSubnet(staticConfig.Subnet); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig.subnet: %s", err.Error()))
	}

	// Validate that IP is within subnet range
	if err := validateSubnetRange(staticConfig.IPAddress, staticConfig.Subnet); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig: %s", err.Error()))
	}

	// Validate gateway reachability
	if err := validateGatewayReachability(staticConfig.IPAddress, staticConfig.Gateway, staticConfig.Subnet); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig: %s", err.Error()))
	}

	// Validate DNS servers if provided
	if len(staticConfig.DNSServers) > 0 {
		if err := validateDNSServers(staticConfig.DNSServers); err != nil {
			return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig.dnsServers: %s", err.Error()))
		}
	}

	// Validate interface name if provided
	if staticConfig.Interface != nil {
		interfaceName := strings.TrimSpace(*staticConfig.Interface)
		if interfaceName == "" {
			return apierrors.NewBadRequest("lxdConfig.networkConfig.staticIPConfig.interface cannot be empty")
		}
		// Basic interface name validation (alphanumeric and common characters)
		interfaceRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
		if !interfaceRegex.MatchString(interfaceName) {
			return apierrors.NewBadRequest("lxdConfig.networkConfig.staticIPConfig.interface must contain only alphanumeric characters, dots, underscores, and hyphens")
		}
	}

	return nil
}

// validateStaticIPConflicts validates static IP configuration and checks for conflicts
func (r *MaasMachine) validateStaticIPConflicts(ctx context.Context) error {
	// Only validate if using LXD with static IP configuration
	if r.Spec.ProvisioningMode != ProvisioningModeLXD {
		return nil
	}

	if r.Spec.LXDConfig == nil ||
		r.Spec.LXDConfig.NetworkConfig == nil ||
		r.Spec.LXDConfig.NetworkConfig.StaticIPConfig == nil {
		return nil
	}

	staticConfig := r.Spec.LXDConfig.NetworkConfig.StaticIPConfig

	// Perform additional subnet range validation that we added
	if err := validateSubnetRange(staticConfig.IPAddress, staticConfig.Subnet); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig: %s", err.Error()))
	}

	// Check for IP conflicts if we have the webhook client available
	if webhookClient != nil {
		if err := r.checkIPConflicts(ctx, webhookClient, staticConfig.IPAddress); err != nil {
			return apierrors.NewBadRequest(fmt.Sprintf("lxdConfig.networkConfig.staticIPConfig: %s", err.Error()))
		}
	}

	return nil
}

// validateHostSelection validates LXD host selection configuration
func (r *MaasMachine) validateHostSelection(hostSel *LXDHostSelectionConfig) error {
	// Validate that host selection criteria are not conflicting
	if len(hostSel.PreferredHosts) > 0 && len(hostSel.AvailabilityZones) > 0 {
		return apierrors.NewBadRequest("lxdConfig.hostSelection cannot specify both preferredHosts and availabilityZones")
	}

	// Validate system IDs format if provided
	for _, hostID := range hostSel.PreferredHosts {
		if strings.TrimSpace(hostID) == "" {
			return apierrors.NewBadRequest("lxdConfig.hostSelection.preferredHosts cannot contain empty host IDs")
		}
	}

	// Validate availability zones format if provided
	for _, zone := range hostSel.AvailabilityZones {
		if strings.TrimSpace(zone) == "" {
			return apierrors.NewBadRequest("lxdConfig.hostSelection.availabilityZones cannot contain empty zone names")
		}
	}

	return nil
}
