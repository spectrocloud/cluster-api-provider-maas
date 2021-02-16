package maasclient

import (
	"encoding/json"
	"fmt"
	"github.com/dghubble/oauth1"
	"io/ioutil"
	"net/http"
	"strings"
)

// Client
type Client struct {
	baseURL    string
	HTTPClient *http.Client
}

// NewClient creates new MaaS client with given API key
func NewClient(maasEndpoint string, apiKey string) *Client {

	key := strings.SplitN(apiKey, ":", 3)
	config := oauth1.NewConfig(key[0], "")
	token := oauth1.NewToken(key[1], key[2])

	return &Client{
		HTTPClient: config.Client(oauth1.NoContext, token),
		baseURL:    fmt.Sprintf("%s/api/2.0", maasEndpoint),
	}
}

// Content-type and body should be already added to req
func (c *Client) sendRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	//debugBody(res)

	// Try to unmarshall into errorResponse
	if res.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("unknown error, status code: %d, body: %s", res.StatusCode, string(bodyBytes))
	}

	if err = json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}

	return nil
}

func debugBody(res *http.Response) {
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	bodyString := string(bodyBytes)

	fmt.Println(bodyString)
}
