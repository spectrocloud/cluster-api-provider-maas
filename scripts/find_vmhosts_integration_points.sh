#!/bin/bash

# Script to find all VMHosts API integration points
# Run this script when the MAAS VMHosts API becomes available

echo "=== MAAS VMHosts API Integration Points ==="
echo

echo "📍 Primary Integration Files:"
echo "  pkg/maas/client/vmhost.go - Client layer implementation"
echo "  pkg/maas/lxd/service.go - Service layer integration"
echo

echo "🔍 TODO Comments to Replace:"
echo
grep -r "TODO.*VMHosts\|TODO.*client supports\|TODO.*MAAS client" \
    --include="*.go" \
    --exclude-dir=vendor \
    --exclude-dir=.git \
    . | \
    sed 's/^/  /'

echo
echo "🔧 Mock Functions to Replace:"
echo
grep -r "getMock\|mockComposed\|mockVMHost" \
    --include="*.go" \
    --exclude-dir=vendor \
    --exclude-dir=.git \
    pkg/maas/client/ pkg/maas/lxd/ | \
    grep -v "test" | \
    sed 's/^/  /'

echo
echo "📋 Integration Checklist:"
echo "  1. Update maas-client-go dependency"
echo "  2. Replace VMHostExtension.QueryVMHosts() implementation"
echo "  3. Replace VMHostExtension.ComposeVM() implementation" 
echo "  4. Replace Service.GetAvailableLXDHosts() implementation"
echo "  5. Replace Service.ComposeVM() mock composition"
echo "  6. Enhance extractNetworkStatusFromVM() with real API calls"
echo "  7. Add real interface inspection methods"
echo "  8. Update tests for API integration"
echo "  9. Remove mock helper functions"
echo " 10. Update documentation"

echo
echo "🧪 Test Commands:"
echo "  go test ./pkg/maas/client/..."
echo "  go test ./pkg/maas/lxd/..."
echo "  # For integration testing:"
echo "  MAAS_ENDPOINT=http://your-maas/MAAS MAAS_APIKEY=your-key go test -tags integration ./..."