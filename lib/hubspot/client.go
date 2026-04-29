package hubspot

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/belong-inc/go-hubspot"
)

// maxSearchLimit is the upstream cap on per-page search results documented by
// the HubSpot CRM Search API and reflected in the SDK comment on
// SearchOptions.Limit ("the maximum number of entries per page is 200"). It is
// shared by all SearchX methods on Client.
const maxSearchLimit = 200

// Client wraps the HubSpot SDK with thin domain-specific helpers.
type Client struct {
	sdk *hubspot.Client
}

type clientConfig struct {
	baseURL    *url.URL
	httpClient *http.Client
	err        error
}

// ClientOption configures a Client at construction time.
type ClientOption func(*clientConfig)

// WithBaseURL overrides the HubSpot API base URL. A parse error surfaces from
// NewClient.
func WithBaseURL(rawURL string) ClientOption {
	return func(cfg *clientConfig) {
		u, err := url.Parse(rawURL)
		if err != nil {
			cfg.err = fmt.Errorf("parse base url %q: %w", rawURL, err)
			return
		}
		cfg.baseURL = u
	}
}

// WithHTTPClient overrides the HTTP client used for API requests.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(cfg *clientConfig) {
		cfg.httpClient = httpClient
	}
}

// NewClient constructs a HubSpot client wrapper using a private app token.
func NewClient(accessToken string, opts ...ClientOption) (*Client, error) {
	if accessToken == "" {
		return nil, errors.New("hubspot access token is required")
	}

	cfg := &clientConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.err != nil {
		return nil, cfg.err
	}

	sdkOpts := make([]hubspot.Option, 0, 2)
	if cfg.baseURL != nil {
		sdkOpts = append(sdkOpts, hubspot.WithBaseURL(cfg.baseURL))
	}
	if cfg.httpClient != nil {
		sdkOpts = append(sdkOpts, hubspot.WithHTTPClient(cfg.httpClient))
	}

	sdk, err := hubspot.NewClient(hubspot.SetPrivateAppToken(accessToken), sdkOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{sdk: sdk}, nil
}
