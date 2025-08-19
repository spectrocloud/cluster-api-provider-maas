#!/bin/bash

# Test script to verify LXD setup functionality
set -e

echo "=== Testing LXD Setup Functionality ==="

# Test 1: Check if LXD is available
echo "1. Checking LXD availability..."
if command -v lxc &> /dev/null; then
    echo "✅ LXD is available"
else
    echo "❌ LXD is not available - please install LXD first"
    exit 1
fi

# Test 2: Check if LXD is initialized
echo "2. Checking LXD initialization..."
if lxc info &> /dev/null; then
    echo "✅ LXD is already initialized"
else
    echo "ℹ️  LXD is not initialized - will be initialized during setup"
fi

# Test 3: Check network bridge
echo "3. Checking network bridge..."
if ip link show br0 &> /dev/null; then
    echo "✅ Network bridge br0 exists"
else
    echo "❌ Network bridge br0 does not exist - please create it first"
    echo "   Example: sudo ip link add br0 type bridge"
    exit 1
fi

# Test 4: Check MAAS CLI availability
echo "4. Checking MAAS CLI availability..."
if command -v maas &> /dev/null; then
    echo "✅ MAAS CLI is available"
else
    echo "❌ MAAS CLI is not available - please install MAAS CLI first"
    exit 1
fi

# Test 5: Check MAAS connectivity
echo "5. Checking MAAS connectivity..."
if maas admin version &> /dev/null; then
    echo "✅ MAAS connectivity is working"
else
    echo "❌ MAAS connectivity failed - please check MAAS configuration"
    exit 1
fi

echo ""
echo "=== All prerequisites are met! ==="
echo "You can now apply the LXD cluster configuration:"
echo "kubectl apply -f examples/sample-lxd-cluster.yaml"
echo ""
echo "To monitor the LXD setup:"
echo "kubectl get maascluster lxd-cluster -o yaml"
echo "kubectl logs -l app=cluster-api-provider-maas -c manager | grep -i lxd" 