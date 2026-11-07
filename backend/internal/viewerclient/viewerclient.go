// Package viewerclient calls watchd: a viewer states what it watches, a publisher reads who watches it
// (internal/watchsvc).
package viewerclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/mirrorme/internal/watchsvc"
)

// Timeout bounds one call, well inside the poll interval the calls ride on.
const Timeout = 5 * time.Second

// Client calls watchd and holds nothing between calls.
// Can be called concurrently.
type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: Timeout}}
}

// State replaces everything this member watches with viewing, under the member the token names.
// An empty viewing states that the member watches nothing.
// Errors if watchd does not answer or refuses the statement, with the reason it gave.
func (c *Client) State(base, token string, viewing []watchsvc.Viewing) error {
	assert.Assert(base != "", "a statement goes to a named service")
	assert.Assert(token != "", "a statement carries the token naming its member")

	if viewing == nil {
		viewing = []watchsvc.Viewing{}
	}
	body, err := json.Marshal(map[string]any{"viewing": viewing})
	assert.IsNil(err, "a viewing set is plain data that always encodes")

	return c.call(http.MethodPut, base+"/viewers", token, bytes.NewReader(body), nil)
}

// Viewers answers who watches path and what each of them measures, in member order.
// Errors if watchd does not answer or refuses the read, such as for a path the token publishes nothing on.
func (c *Client) Viewers(base, token, path string) ([]watchsvc.Viewer, error) {
	assert.Assert(base != "", "a read goes to a named service")
	assert.Assert(token != "", "a read carries the token naming its member")
	assert.Assert(path != "", "a read names the path whose viewers it asks for")

	var answer struct {
		Viewers []watchsvc.Viewer `json:"viewers"`
	}
	address := base + "/viewers?path=" + url.QueryEscape(path)
	if err := c.call(http.MethodGet, address, token, nil, &answer); err != nil {
		return nil, err
	}
	return answer.Viewers, nil
}

// call makes one request and decodes its answer into into, where into is not nil.
func (c *Client) call(method, address, token string, body io.Reader, into any) error {
	request, err := http.NewRequest(method, address, body)
	if err != nil {
		return fmt.Errorf("addressing %s: %w", address, err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, address, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var refusal struct {
			Error string `json:"error"`
		}
		json.NewDecoder(io.LimitReader(resp.Body, 4<<10)).Decode(&refusal)
		return fmt.Errorf("%s %s: status %d: %s", method, address, resp.StatusCode, refusal.Error)
	}
	if into == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		return fmt.Errorf("reading the answer of %s %s: %w", method, address, err)
	}
	return nil
}
