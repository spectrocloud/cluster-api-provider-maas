package maasclient

import (
	"context"
	"net/http"
	"net/url"
)

// DNSResource
type DNSResource struct {
	ID          int          `json:"id"`
	FQDN        string       `json:"fqdn"`
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
	FQDN *string `json:"fqdn"`
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
