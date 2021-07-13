package maasclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spectrocloud/cluster-api-provider-maas/pkg/maasclient/oauth1"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client
type Client struct {
	baseURL    string
	HTTPClient *http.Client
	apiKey     string
	auth       oauth1.OAuth1
}

// NewClient creates new MaaS client with given API key
func NewClient(maasEndpoint string, apiKey string) *Client {

	return &Client{
		apiKey:     apiKey,
		HTTPClient: http.DefaultClient,
		//HTTPClient: config.Client(oauth1.NoContext, token),
		baseURL: fmt.Sprintf("%s/api/2.0", maasEndpoint),
	}
}

// send sends the request
// Content-type and body should be already added to req
func (c *Client) send(ctx context.Context, method string, apiPath string, params url.Values, v interface{}) error {

	var err error
	var req *http.Request

	if method == http.MethodGet {
		req, err = http.NewRequestWithContext(
			ctx,
			method,
			fmt.Sprintf("%s%s", c.baseURL, apiPath),
			nil,
		)
		if err != nil {
			return err
		}

		req.URL.RawQuery = params.Encode()
	} else {
		req, err = http.NewRequestWithContext(
			ctx,
			method,
			fmt.Sprintf("%s%s", c.baseURL, apiPath),
			strings.NewReader(params.Encode()),
		)
		if err != nil {
			return err
		}
	}

	return c.sendRequest(req, params, v)
}

// send sends the request
// Content-type and body should be already added to req
func (c *Client) sendRequestWithBody(ctx context.Context, method string, apiPath string, contenttype string, params url.Values, body io.Reader, v interface{}) error {

	var err error
	var req *http.Request

		req, err = http.NewRequestWithContext(
			ctx,
			method,
			fmt.Sprintf("%s%s", c.baseURL, apiPath),
			body,
		)
		if err != nil {
			return err
		}

	return c.sendRequestUpload(req, params, contenttype, v)
}

func (c *Client) sendRequest(req *http.Request, params url.Values, v interface{}) error {
	//func (c *Client) sendRequest(req *http.Request, urlValues *url.Values, v interface{}) error {
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")



	authHeader := authHeader(req, params, c.apiKey)
	req.Header.Set("Authorization", authHeader)



	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	//debugBody(res)

	// Try to unmarshall into errorResponse
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("unknown error, status code: %d, body: %s", res.StatusCode, string(bodyBytes))
	} else if res.StatusCode == http.StatusNoContent {
		return nil
	}

	if err = json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}


	return nil
}

func (c *Client) sendRequestUpload(req *http.Request, params url.Values, contenttype string, v interface{}) error {
	//func (c *Client) sendRequest(req *http.Request, urlValues *url.Values, v interface{}) error {
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Content-Type", contenttype)


	// for post requests longer than 300 seconds
	ticker := time.NewTicker(2 * time.Minute)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("refresing auth token", time.Unix(t.Unix(), 0).Format(time.RFC3339))
				authHeader := authHeader(req, params, c.apiKey)
				req.Header.Set("Authorization", authHeader)
			}
		}
	}()


	defer func() {
		ticker.Stop()
		done <- true
	}()


	authHeader := authHeader(req, params, c.apiKey)
	req.Header.Set("Authorization", authHeader)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	//debugBody(res)

	// Try to unmarshall into errorResponse
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("unknown error, status code: %d, body: %s", res.StatusCode, string(bodyBytes))
	} else if res.StatusCode == http.StatusNoContent {
		return nil
	}

	if err = json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}

	return nil
}


func authHeader(req *http.Request, queryParams url.Values, apiKey string) string {
	key := strings.SplitN(apiKey, ":", 3)
	//config := oauth1.NewConfig(key[0], "")
	//token := oauth1

	auth := oauth1.OAuth1{
		ConsumerKey:    key[0],
		ConsumerSecret: "",
		AccessToken:    key[1],
		AccessSecret:   key[2],
	}

	//queryParams, _ := url.ParseQuery(req.URL.RawQuery)
	params := make(map[string]string)
	if req.Method != http.MethodPut {
		// for some bizarre-reason PUT doesn't need this
		for k, v := range queryParams {
			params[k] = v[0]
		}
	}

	authHeader := auth.BuildOAuth1Header(req.Method, req.URL.String(), params)
	return authHeader
}

func debugBody(res *http.Response) {
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	bodyString := string(bodyBytes)

	fmt.Println(bodyString)
}
