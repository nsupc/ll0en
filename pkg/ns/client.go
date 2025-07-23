package ns

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	client  http.Client
	user    string
	limiter *rate.Limiter
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	err := c.limiter.Wait(context.Background())
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", c.user)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %s", resp.Status)
	}

	return resp, nil
}

func NewClient(user string, maxRequests int) *Client {
	client := Client{
		client:  http.Client{Timeout: 5 * time.Second},
		user:    user,
		limiter: rate.NewLimiter(rate.Limit(maxRequests/30), maxRequests),
	}

	return &client
}
