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

// DNSResourcesList .
type DNSResourcesList struct {
	Count      int `json:"count"`
	PagesCount int `json:"pages_count"`
	//DNSResources      []DNSResource `json:"dnsResources"`
}

func (c *Client) GetDNSResources(ctx context.Context, options *DNSResourcesListOptions) (*DNSResourcesList, error) {
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

	res := DNSResourcesList{}
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
