package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type (
	Pagination struct {
		TotalResults int  `json:"total_results"`
		TotalPages   int  `json:"total_pages"`
		First        Href `json:"first"`
		Last         Href `json:"last"`
		Next         Href `json:"next"`
		Previous     Href `json:"previous"`
	}
	Href struct {
		Url string `json:"href"`
	}

	//Response  for example https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#list-processes
	Response[T any] struct {
		Pagination Pagination `json:"pagination"`
		Resources  []T        `json:"resources"`
	}

	ResourceRetriever[T any] struct {
		*Client
	}
)

func (r ResourceRetriever[T]) getAllPages(url string) ([]T, error) {
	pageNumber := 1
	var resources []T
	for url != "" {
		page, err := r.getPage(url)
		if err != nil {
			return nil, fmt.Errorf("failed getting page %d: %w", pageNumber, err)
		}
		resources = append(resources, page.Resources...)
		url = page.Pagination.Next.Url
		pageNumber++
	}
	return resources, nil
}

func (r ResourceRetriever[T]) getPage(url string) (Response[T], error) {
	response := Response[T]{}
	resp, err := r.get(url)
	if err != nil {
		return response, fmt.Errorf("failed getting %T: %w", response, err)
	}
	defer func() { _ = resp.Body.Close() }()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return response, fmt.Errorf("failed unmarshalling %T: %w", response, err)
	}
	return response, nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.sendRequest(req)
}

func (c *Client) post(url string, bodyStuct any) (*http.Response, error) {
	body, err := json.Marshal(bodyStuct)
	if err != nil {
		return nil, fmt.Errorf("failed post: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post %s failed: %w", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	return c.sendRequest(req)
}

func (c *Client) sendRequest(req *http.Request) (*http.Response, error) {
	tokens, err := c.GetTokens()
	if err != nil {
		return nil, fmt.Errorf("get token failed: %w", err)
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)

	resp, err := c.retryClient.Do(req)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if isError(statusCode) {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response[%d]: %w", statusCode, err)
		}
		return nil, fmt.Errorf("%s request failed: %w", req.Method, models.NewCfError(req.RequestURI, req.RequestURI, statusCode, respBody))
	}
	return resp, nil
}

func isError(statusCode int) bool {
	return statusCode >= 300
}
