package maasclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

// Machine
type Machine struct {
	//ID   int    `json:"id"`
	SystemID     string `json:"system_id"`
	FQDN         string `json:"fqdn"`
	Zone         `json:"zone"`
	PowerState   string   `json:"power_state"`
	Hostname     string   `json:"hostname"`
	IpAddresses  []string `json:"ip_addresses"`
	State        string   `json:"status_name"`
	OSSystem     string   `json:"osystem"`
	DistroSeries string   `json:"distro_series"`
	SwapSize     *int     `json:"swap_size"`
}

type Zone struct {
	AvailabilityZone string `json:"name"`
}

func (c *Client) GetMachine(ctx context.Context, systemID string) (*Machine, error) {

	// creates the zero value
	res := new(Machine)
	if err := c.send(ctx, http.MethodGet, fmt.Sprintf("/machines/%s/", systemID), nil, res); err != nil {
		return nil, err
	}

	return res, nil
}

type AllocateMachineOptions struct {
	SystemID         *string
	Name             *string
	AvailabilityZone *string
	ResourcePool     *string
	MinCPU           *int
	MinMem           *int
}

func (c *Client) AllocateMachine(ctx context.Context, options *AllocateMachineOptions) (*Machine, error) {
	// or you can create new url.Values struct and encode that like so
	q := url.Values{}
	q.Add("op", "allocate")

	if options != nil {
		addParam(q, "zone", options.AvailabilityZone)
		addParam(q, "system_id", options.SystemID)
		addParam(q, "name", options.Name)
		addParam(q, "cpu_count", options.MinCPU)
		addParam(q, "mem", options.MinMem)
		addParam(q, "pool", options.ResourcePool)
	}

	res := new(Machine)
	if err := c.send(ctx, http.MethodPost, "/machines/", q, res); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) ReleaseMachine(ctx context.Context, systemID string) error {
	q := url.Values{}
	q.Add("op", "release")

	// creates the zero value
	res := new(Machine)
	if err := c.send(ctx, http.MethodPost, fmt.Sprintf("/machines/%s/", systemID), q, res); err != nil {
		return err
	}

	return nil
}

type DeployMachineOptions struct {
	SystemID     string
	OSSystem     *string
	DistroSeries *string
	UserData     *string
}

func (c *Client) DeployMachine(ctx context.Context, options DeployMachineOptions) (*Machine, error) {
	q := url.Values{}
	q.Add("op", "deploy")

	addParam(q, "osystem", options.OSSystem)
	addParam(q, "distro_series", options.DistroSeries)
	addParam(q, "user_data", options.UserData)

	// creates the zero value
	res := new(Machine)
	if err := c.send(ctx, http.MethodPost, fmt.Sprintf("/machines/%s/", options.SystemID), q, res); err != nil {
		return nil, err
	}

	return res, nil
}

type UpdateMachineOptions struct {
	SystemID string
	SwapSize *int
}

func (c *Client) UpdateMachine(ctx context.Context, options UpdateMachineOptions) (*Machine, error) {

	q := url.Values{}
	addParam(q, "swap_size", options.SwapSize)

	// creates the zero value
	res := new(Machine)
	if err := c.send(ctx, http.MethodPut, fmt.Sprintf("/machines/%s/", options.SystemID), q, res); err != nil {
		return nil, err
	}

	return res, nil
}

func addParam(values url.Values, key string, value interface{}) {
	if reflect.ValueOf(value).IsNil() {
		return
	}

	switch v := value.(type) {
	case *int:
		values.Add(key, strconv.Itoa(*v))
	case *string:
		values.Add(key, *v)
	default:
		panic(fmt.Sprintf("Unexpected type %v", v))
	}
}
