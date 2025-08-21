package main

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// LXD Host initialization label
	LXDHostInitializedLabel = "lxdhost.cluster.com/initialized"

	// Label values
	LabelValueTrue = "true"

	// Timeouts
	NodeLabelTimeout = 30 * time.Second
)

// NodeLabeler handles labeling nodes with LXD initialization status
type NodeLabeler struct {
	client   kubernetes.Interface
	nodeName string
}

// NewNodeLabeler creates a new NodeLabeler instance
func NewNodeLabeler(nodeName string) (*NodeLabeler, error) {
	if nodeName == "" {
		return nil, fmt.Errorf("node name cannot be empty")
	}

	// Create in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &NodeLabeler{
		client:   clientset,
		nodeName: nodeName,
	}, nil
}

// MarkLXDInitialized adds a label to the node indicating LXD initialization is complete
func (nl *NodeLabeler) MarkLXDInitialized() error {
	ctx, cancel := context.WithTimeout(context.Background(), NodeLabelTimeout)
	defer cancel()

	// Get the current node
	node, err := nl.client.CoreV1().Nodes().Get(ctx, nl.nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", nl.nodeName, err)
	}

	// Initialize labels map if it doesn't exist
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	// Add the LXD initialization label
	node.Labels[LXDHostInitializedLabel] = LabelValueTrue

	// Update the node
	_, err = nl.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update node %s with LXD label: %w", nl.nodeName, err)
	}

	return nil
}

// SafeMarkLXDInitialized safely marks LXD as initialized with error handling
func (nl *NodeLabeler) SafeMarkLXDInitialized() {
	if err := nl.MarkLXDInitialized(); err != nil {
		log.Printf("Warning: Failed to label node %s as LXD initialized: %v", nl.nodeName, err)
	}
}
