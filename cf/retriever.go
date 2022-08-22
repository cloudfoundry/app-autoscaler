package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	PagedResourceRetriever[T any] ResourceRetriever[T]
	ResourceRetriever[T any]      struct{ Retriever }

	Retriever interface {
		SendRequest(req *http.Request) (*http.Response, error)
		ApiUrl(pathAndQuery string) string
		CFClient
	}
	AuthenticatedClient struct {
		Retriever
	}
)

var _ Retriever = &Client{}
var _ Retriever = AuthenticatedClient{}

func (r PagedResourceRetriever[T]) GetAllPages(pathAndQuery string) ([]T, error) {
	pageNumber := 1
	var resources []T

	url := r.Retriever.ApiUrl(pathAndQuery)

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

func (r PagedResourceRetriever[T]) GetPage(pathAndQuery string) (Response[T], error) {
	return r.getPage(r.Retriever.ApiUrl(pathAndQuery))
}

func (r PagedResourceRetriever[T]) getPage(url string) (Response[T], error) {
	return ResourceRetriever[Response[T]](r).get(url)
}

func (r ResourceRetriever[T]) Get(pathAndQuery string) (T, error) {
	return r.get(r.Retriever.ApiUrl(pathAndQuery))
}

func (r ResourceRetriever[T]) get(url string) (T, error) {
	var response T

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return response, err
	}

	resp, err := Retriever(r).SendRequest(req)
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

func (r ResourceRetriever[T]) Post(url string, bodyStuct any) (*http.Response, error) {
	body, err := json.Marshal(bodyStuct)
	if err != nil {
		return nil, fmt.Errorf("failed post: %w", err)
	}
	req, err := http.NewRequest("POST", r.Retriever.ApiUrl(url), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post %s failed: %w", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	return Retriever(r).SendRequest(req)
}

func (r AuthenticatedClient) SendRequest(req *http.Request) (*http.Response, error) {
	err := r.addAuth(req)
	if err != nil {
		return nil, err
	}
	return r.Retriever.SendRequest(req)
}

func (c *Client) SendRequest(req *http.Request) (*http.Response, error) {
	resp, err := c.retryClient.Do(req)
	if err != nil {
		return resp, err
	}

	statusCode := resp.StatusCode
	if isError(statusCode) {
		defer func() { _ = resp.Body.Close() }()
		//TODO use limitReader here
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, fmt.Errorf("failed to read response[%d]: %w", statusCode, err)
		}
		return resp, fmt.Errorf("%s request failed: %w", req.Method, models.NewCfError(req.RequestURI, req.RequestURI, statusCode, respBody))
	}
	return resp, nil
}

func (c *Client) ApiUrl(pathAndQuery string) string {
	return c.conf.API + pathAndQuery
}

func (r AuthenticatedClient) addAuth(req *http.Request) error {
	tokens, err := r.Retriever.GetTokens()
	if err != nil {
		return fmt.Errorf("get token failed: %w", err)
	}

	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)
	return nil
}

func isError(statusCode int) bool {
	return statusCode >= 300
}
