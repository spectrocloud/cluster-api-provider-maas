package client

import (
	"encoding/json"
	"errors"
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
		baseURL: fmt.Sprintf("%s/api/2.0", maasEndpoint),
	}
}

//// Rectangle .
//type Rectangle struct {
//	Top    int `json:"top"`
//	Left   int `json:"left"`
//	Width  int `json:"width"`
//	Height int `json:"height"`
//}
//
type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type successResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

// Content-type and body should be already added to req
func (c *Client) sendRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Accept", "application/json; charset=utf-8")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	bodyString := string(bodyBytes)

	fmt.Println(bodyString)

	// Try to unmarshall into errorResponse
	if res.StatusCode != http.StatusOK {
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			return errors.New(errRes.Message)
		}

		return fmt.Errorf("unknown error, status code: %d", res.StatusCode)
	}


	// Unmarshall and populate v
	fullResponse := successResponse{
		Data: v,
	}

	if err = json.NewDecoder(res.Body).Decode(&fullResponse); err != nil {
		return err
	}

	return nil
}

