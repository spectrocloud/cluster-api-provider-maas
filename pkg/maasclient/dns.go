package maasclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// DNSResource
type DNSResource struct {
	ID          int          `json:"id"`
	FQDN        string       `json:"fqdn"`
	AddressTTL  *int         `json:"address_ttl"`
	IpAddresses []*IpAddress `json:"ip_addresses"`
}

type IpAddress struct {
	IpAddress string `json:"ip"`
	//Interfaces []*Interface `json:"interface_set"`
}

//type Interface struct {
//	SystemID string `json:"system_id"`
//}

type GetDNSResourcesOptions struct {
	FQDN *string
}

func (c *Client) GetDNSResources(ctx context.Context, options *GetDNSResourcesOptions) ([]*DNSResource, error) {

	q := url.Values{}
	if options != nil {
		addParam(q, "fqdn", options.FQDN)
	}

	var res []*DNSResource
	if err := c.send(ctx, http.MethodGet, "/dnsresources/", q, &res); err != nil {
		return nil, err
	}

	return res, nil
}

type CreateDNSResourcesOptions struct {
	FQDN        string
	AddressTTL  string
	IpAddresses []string
}

func (c *Client) CreateDNSResources(ctx context.Context, options CreateDNSResourcesOptions) (*DNSResource, error) {

	q := url.Values{}
	q.Add("fqdn", options.FQDN)
	q.Add("address_ttl", options.AddressTTL)
	q.Add("ip_addresses", strings.Join(options.IpAddresses, " "))

	res := new(DNSResource)
	if err := c.send(ctx, http.MethodPost, "/dnsresources/", q, res); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) DeleteDNSResources(ctx context.Context, id int) error {

	//q := url.Values{}
	//q.Add("id", strconv.Itoa(id))

	//res := new(DNSResource)
	if err := c.send(ctx, http.MethodDelete, fmt.Sprintf("/dnsresources/%v/", id), nil, nil); err != nil {
		return err
	}

	return nil
}

type UpdateDNSResourcesOptions struct {
	ID          int
	IpAddresses []string
}

func (c *Client) UpdateDNSResources(ctx context.Context, options UpdateDNSResourcesOptions) (*DNSResource, error) {

	q := url.Values{}
	q.Add("ip_addresses", strings.Join(options.IpAddresses, " "))

	res := new(DNSResource)
	if err := c.send(ctx, http.MethodPut, fmt.Sprintf("/dnsresources/%v/", options.ID), q, res); err != nil {
		return nil, err
	}

	return res, nil
}
