package cf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		SendRequest(ctx context.Context, req *http.Request) (*http.Response, error)
		ApiUrl(pathAndQuery string) string
		ApiContextClient
		AuthContextClient
	}
	AuthenticatedClient struct {
		Retriever
	}
)

var _ Retriever = &CtxClient{}
var _ Retriever = AuthenticatedClient{}

func (r PagedResourceRetriever[T]) GetAllPages(ctx context.Context, pathAndQuery string) ([]T, error) {
	pageNumber := 1
	var resources []T

	//nolint:staticcheck // QF1008: embedded field access is intentional for API design
	url := r.Retriever.ApiUrl(pathAndQuery)

	for url != "" && ctx.Err() == nil {
		page, err := r.getPage(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("failed getting page %d: %w", pageNumber, err)
		}
		resources = append(resources, page.Resources...)
		url = page.Pagination.Next.Url
		pageNumber++
	}
	return resources, ctx.Err()
}

func (r PagedResourceRetriever[T]) GetPage(ctx context.Context, pathAndQuery string) (Response[T], error) {
	//nolint:staticcheck // QF1008: embedded field access is intentional for API design
	return r.getPage(ctx, r.Retriever.ApiUrl(pathAndQuery))
}

func (r PagedResourceRetriever[T]) getPage(ctx context.Context, url string) (Response[T], error) {
	return ResourceRetriever[Response[T]](r).get(ctx, url)
}

func (r ResourceRetriever[T]) Get(ctx context.Context, pathAndQuery string) (T, error) {
	//nolint:staticcheck // QF1008: embedded field access is intentional for API design
	return r.get(ctx, r.Retriever.ApiUrl(pathAndQuery))
}

func (r ResourceRetriever[T]) get(ctx context.Context, url string) (T, error) {
	var response T

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return response, err
	}

	return r.sendAndDeserialise(ctx, req)
}

func (r ResourceRetriever[T]) sendAndDeserialise(ctx context.Context, req *http.Request) (T, error) {
	var response T
	resp, err := Retriever(r).SendRequest(ctx, req)
	if err != nil {
		return response, fmt.Errorf("failed %s-ing %T: %w", req.Method, response, err)
	}

	defer func() { _ = resp.Body.Close() }()
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return response, fmt.Errorf("failed unmarshalling %T: %w", response, err)
	}
	return response, nil
}

func (r ResourceRetriever[T]) Post(ctx context.Context, pathAndQuery string, bodyStuct any) (T, error) {
	//nolint:staticcheck // QF1008: embedded field access is intentional for API design
	return r.post(ctx, r.Retriever.ApiUrl(pathAndQuery), bodyStuct)
}

func (r ResourceRetriever[T]) post(ctx context.Context, url string, bodyStuct any) (T, error) {
	body, err := json.Marshal(bodyStuct)
	if err != nil {
		var result T
		return result, fmt.Errorf("failed post: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		var result T
		return result, fmt.Errorf("post %s failed: %w", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	return r.sendAndDeserialise(ctx, req)
}

func (r AuthenticatedClient) SendRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	err := r.addAuth(ctx, req)
	if err != nil {
		return nil, err
	}
	return r.Retriever.SendRequest(ctx, req)
}

func (c *CtxClient) SendRequest(_ context.Context, req *http.Request) (*http.Response, error) {
	c.setUserAgent(req)
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
		return resp, fmt.Errorf("%s request failed: %w", req.Method, NewCfError(req.RequestURI, req.RequestURI, statusCode, respBody))
	}
	return resp, nil
}

func (c *CtxClient) ApiUrl(pathAndQuery string) string {
	return c.conf.API + pathAndQuery
}

func (r AuthenticatedClient) addAuth(ctx context.Context, req *http.Request) error {
	//nolint:staticcheck // QF1008: embedded field access is intentional for API design
	tokens, err := r.Retriever.GetTokens(ctx)
	if err != nil {
		return fmt.Errorf("get token failed: %w", err)
	}

	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)
	return nil
}

func isError(statusCode int) bool {
	return statusCode >= 300
}
