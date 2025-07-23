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

package vmhosts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

// VMHost represents a MAAS VM host
type VMHost struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	PowerAddress string `json:"power_address"`
	Zone         string `json:"zone"`
	Pool         string `json:"pool"`
	CPU          struct {
		Total int `json:"total"`
		Used  int `json:"used"`
	} `json:"cpu"`
	Memory struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
	} `json:"memory"`
	Storage struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
	} `json:"storage"`
}

// Machine represents a MAAS machine
type Machine struct {
	SystemID string `json:"system_id"`
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
	Zone     string `json:"zone"`
	Pool     string `json:"pool"`
}

// VMHostsClient is a client for interacting with MAAS VM hosts
type VMHostsClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new VMHostsClient
func NewClient(apiKey, baseURL string) *VMHostsClient {
	return &VMHostsClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetVMHosts returns a list of VM hosts from MAAS
func (v *VMHostsClient) GetVMHosts() ([]VMHost, error) {
	return v.ListVMHosts()
}

// ListVMHosts returns a list of VM hosts from MAAS
func (v *VMHostsClient) ListVMHosts() ([]VMHost, error) {
	// Create HTTP request
	apiURL := fmt.Sprintf("%s/MAAS/api/2.0/vm-hosts/", v.baseURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", v.apiKey))

	// Make request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var hosts []VMHost
	if err := json.Unmarshal(body, &hosts); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return hosts, nil
}

// CreateVMHostWithParams creates a new VM host in MAAS with the given parameters
func (v *VMHostsClient) CreateVMHostWithParams(params url.Values) (*VMHost, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Write form fields from URL values
	for key, values := range params {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return nil, fmt.Errorf("failed to write field %s: %w", key, err)
			}
		}
	}

	writer.Close()

	// Create HTTP request
	apiURL := fmt.Sprintf("%s/MAAS/api/2.0/vm-hosts/", v.baseURL)
	req, err := http.NewRequest("POST", apiURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", v.apiKey))

	// Make request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var vmHost VMHost
	if err := json.Unmarshal(body, &vmHost); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &vmHost, nil
}

// ComposeVM creates a new VM on a VM host
func (v *VMHostsClient) ComposeVM(vmHostID string, params url.Values) (*Machine, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Write form fields from URL values
	for key, values := range params {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return nil, fmt.Errorf("failed to write field %s: %w", key, err)
			}
		}
	}

	writer.Close()

	// Create HTTP request
	apiURL := fmt.Sprintf("%s/MAAS/api/2.0/vm-hosts/%s/compose", v.baseURL, vmHostID)
	req, err := http.NewRequest("POST", apiURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", v.apiKey))

	// Make request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var machine Machine
	if err := json.Unmarshal(body, &machine); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &machine, nil
}

// DeleteMachine deletes a machine from MAAS
func (v *VMHostsClient) DeleteMachine(systemID string) error {
	// Create HTTP request
	apiURL := fmt.Sprintf("%s/MAAS/api/2.0/machines/%s/", v.baseURL, systemID)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", v.apiKey))

	// Make request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
