package appstate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	http *http.Client
	url  string
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	copy := *httpClient
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{http: &copy, url: strings.TrimRight(baseURL, "/")}
}

func (c *Client) Changes(ctx context.Context, cursor int64) (Changes, error) {
	var result Changes
	err := c.request(ctx, http.MethodGet, "/v1/changes?after="+strconv.FormatInt(cursor, 10), &result)
	return result, err
}

func (c *Client) Invalidate(ctx context.Context, appID string) error {
	return c.request(ctx, http.MethodPost, "/v1/apps/"+url.PathEscape(appID)+"/invalidate", nil)
}

func (c *Client) request(ctx context.Context, method, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.url+path, nil)
	if err != nil {
		return err
	}
	// #nosec G704 -- destination is the configured internal observer URL.
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("observer returned HTTP %d", resp.StatusCode)
	}
	if result != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(result)
	}
	return nil
}
