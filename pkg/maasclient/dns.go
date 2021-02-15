package maasclient

import (
	"context"
	"fmt"
	"net/http"
)

// DNSResourcesListOptions .
type DNSResourcesListOptions struct {
	//Limit int `json:"limit"`
	//Page  int `json:"page"`
}

// DNSResource .
type DNSResource struct {
	ID   int    `json:"id"`
	FQDN string `json:"fqdn"`
}

func (c *Client) GetDNSResources(ctx context.Context, options *DNSResourcesListOptions) ([]*DNSResource, error) {
	//limit := 100
	//page := 1
	//if options != nil {
	//	limit = options.Limit
	//	page = options.Page
	//}
	//
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/dnsresources/", c.baseURL), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	var res []*DNSResource
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}

	return res, nil
}
