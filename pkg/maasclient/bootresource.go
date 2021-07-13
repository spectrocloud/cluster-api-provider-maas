package maasclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type BootResource struct {
	Id int `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	Architecture string `json:"architecture"`
	SubArches string `json:"subarches"`
	ResourceURI string `json:"resource_uri"`
	Title string `json:"title"`
}

func (c *Client) ListBootResources(ctx context.Context) ([]*BootResource, error) {
	q := url.Values{}
	var res []*BootResource
	if err := c.send(ctx, http.MethodGet, "/boot-resources/", q, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) BootResourcesImporting(ctx context.Context) (*bool, error) {
	q := url.Values{}
	q.Add("op", "is_importing")
	var res *bool
	if err := c.send(ctx, http.MethodGet, "/boot-resources/", q, &res); err != nil {
		return nil, err
	}
	return res, nil
}

type UploadBootResourceInput struct {
	Name string
	Architecture string
	Digest string
	Size string
	Title string
	File string
}

func (c *Client) UploadBootResource(ctx context.Context, input UploadBootResourceInput) (*BootResource, error) {
	q := url.Values{}

	q.Add("name", input.Name)
	q.Add("architecture", input.Architecture)
	q.Add("sha256", input.Digest)
	q.Add("size", input.Size)
	q.Add("title", input.Title)


	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	err := writeMultiPartFiles(writer, map[string]string{
		filepath.Base(input.File): input.File,
	})
	if err != nil {
		return nil, err
	}

	err = writeMultiPartParams(writer, q)
	if err != nil {
		return nil, err
	}
	writer.Close()

	var res *BootResource
	if err := c.sendRequestWithBody(ctx, http.MethodPost, "/boot-resources/", writer.FormDataContentType(),q, buf, &res); err != nil {
		return nil, err
	}

	return res, nil
}


func writeMultiPartFiles(writer *multipart.Writer, files map[string]string) error {
	for fileName, filePath := range files {
		fw, err := writer.CreateFormFile(fileName, fileName)
		if err != nil {
			return err
		}
		file, err := os.Open(filePath)
		if err != nil {
			return nil
		}
		io.Copy(fw, file)
	}
	return nil
}

func writeMultiPartParams(writer *multipart.Writer, params url.Values) error {
	for key, values := range params {
		for _, value := range values {
			fw, err := writer.CreateFormField(key)
			if err != nil {
				return err
			}
			buffer := bytes.NewBufferString(value)
			io.Copy(fw, buffer)
		}
	}
	return nil

}

func (c *Client) GetBootResource(ctx context.Context, id string) (*BootResource, error) {
	q := url.Values{}
	var res *BootResource
	if err := c.send(ctx, http.MethodGet, fmt.Sprintf("/boot-resources/%s/", id), q, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) DeleteBootResource(ctx context.Context, id string) error {
	q := url.Values{}
	var res interface{}
	if err := c.send(ctx, http.MethodDelete, fmt.Sprintf("/boot-resources/%s/", id), q, &res); err != nil {
		return err
	}
	return nil
}

