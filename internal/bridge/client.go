package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	mu          sync.Mutex
	url         string
	httpClient  *http.Client
	lastPayload []byte
}

type ProfilesResponse struct {
	Current  string   `json:"current"`
	Profiles []string `json:"profiles"`
}

func NewClient(url string) *Client {
	return &Client{
		url: strings.TrimSpace(url),
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (c *Client) SetURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.url = strings.TrimSpace(url)
	c.lastPayload = nil
}

func (c *Client) URL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.url
}

func (c *Client) Send(ctx context.Context, state State) (bool, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return false, err
	}

	c.mu.Lock()
	url := c.url
	if bytes.Equal(payload, c.lastPayload) {
		c.mu.Unlock()
		return false, nil
	}
	c.mu.Unlock()

	if url == "" {
		return false, fmt.Errorf("server URL is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return false, fmt.Errorf("server returned %s", res.Status)
	}

	c.mu.Lock()
	c.lastPayload = payload
	c.mu.Unlock()
	return true, nil
}

func (c *Client) Check(ctx context.Context) error {
	c.mu.Lock()
	url := c.url
	c.mu.Unlock()

	if url == "" {
		return fmt.Errorf("server URL is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 500 {
		return fmt.Errorf("server returned %s", res.Status)
	}
	return nil
}

func (c *Client) FetchConfig(ctx context.Context) (*OverlayConfig, error) {
	raw, err := c.FetchRawConfig(ctx)
	if err != nil {
		return nil, err
	}

	var cfg OverlayConfig
	if err := decodeRawConfig(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Client) FetchRawConfig(ctx context.Context) (map[string]json.RawMessage, error) {
	c.mu.Lock()
	baseURL := c.url
	c.mu.Unlock()

	configURL, err := configEndpoint(baseURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("config returned %s", res.Status)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) SaveOverlayConfig(ctx context.Context, cfg *OverlayConfig, profile string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	raw, err := c.FetchRawConfig(ctx)
	if err != nil {
		return err
	}
	if raw == nil {
		raw = make(map[string]json.RawMessage)
	}

	controller, err := json.Marshal(cfg.Controller)
	if err != nil {
		return err
	}
	buttons, err := json.Marshal(cfg.Buttons)
	if err != nil {
		return err
	}
	raw["controller"] = controller
	raw["buttons"] = buttons

	body, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	configURL, err := configEndpoint(c.URL())
	if err != nil {
		return err
	}
	if strings.TrimSpace(profile) != "" {
		configURL, err = withProfileQuery(configURL, profile)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, configURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("config save returned %s", res.Status)
	}
	return nil
}

func (c *Client) FetchProfiles(ctx context.Context) (*ProfilesResponse, error) {
	endpoint, err := profileListEndpoint(c.URL())
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("profiles returned %s", res.Status)
	}
	var profiles ProfilesResponse
	if err := json.NewDecoder(res.Body).Decode(&profiles); err != nil {
		return nil, err
	}
	return &profiles, nil
}

func (c *Client) SwitchProfile(ctx context.Context, profile string) (string, error) {
	endpoint, err := profileSwitchEndpoint(c.URL())
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]string{"name": strings.TrimSpace(profile)})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("profile switch returned %s", res.Status)
	}
	var payload struct {
		Current string `json:"current"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Current, nil
}

func configEndpoint(raw string) (string, error) {
	return apiEndpoint(raw, "/api/config")
}

func profileListEndpoint(raw string) (string, error) {
	return apiEndpoint(raw, "/api/config/profiles")
}

func profileSwitchEndpoint(raw string) (string, error) {
	return apiEndpoint(raw, "/api/config/profile")
}

func apiEndpoint(raw, endpoint string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("server URL is invalid")
	}
	switch {
	case strings.HasSuffix(parsed.Path, "/api/input/gamepad"):
		parsed.Path = strings.TrimSuffix(parsed.Path, "/api/input/gamepad") + endpoint
	case strings.Contains(parsed.Path, "/api/input/"):
		parsed.Path = parsed.Path[:strings.Index(parsed.Path, "/api/input/")] + endpoint
	default:
		parsed.Path = strings.TrimRight(parsed.Path, "/") + endpoint
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func withProfileQuery(raw, profile string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("profile", strings.TrimSpace(profile))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func decodeRawConfig(raw map[string]json.RawMessage, cfg *OverlayConfig) error {
	if cfg == nil {
		return fmt.Errorf("config target is nil")
	}
	if value, ok := raw["controller"]; ok {
		if err := json.Unmarshal(value, &cfg.Controller); err != nil {
			return err
		}
	}
	if value, ok := raw["buttons"]; ok {
		if err := json.Unmarshal(value, &cfg.Buttons); err != nil {
			return err
		}
	}
	return nil
}
