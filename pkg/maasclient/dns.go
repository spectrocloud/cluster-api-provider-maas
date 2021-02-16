package maasclient

import (
	"context"
	"fmt"
	"net/http"
)

// DNSResource
type DNSResource struct {
	ID   int    `json:"id"`
	FQDN string `json:"fqdn"`
}

func (c *Client) GetDNSResources(ctx context.Context) ([]*DNSResource, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/dnsresources/", c.baseURL), nil)
	if err != nil {
		return nil, err
	}

	var res []*DNSResource
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}

	return res, nil
}
