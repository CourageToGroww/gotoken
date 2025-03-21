// Package oauth provides OAuth token acquisition and management functionality.
// This file implements the token providers and client credentials flow for OAuth authentication,
// allowing applications to securely obtain access tokens from OAuth servers.
package oauth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// TokenResponse represents a generic OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope,omitempty"`
}

// TokenProvider defines the interface for getting a new OAuth token
type TokenProvider interface {
	GetNewToken() (*TokenResponse, error)
}

// ClientCredentialsProvider implements TokenProvider for the OAuth client credentials flow
type ClientCredentialsProvider struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
	HTTPClient   *http.Client
	ExtraParams  map[string]string
}

// GetNewToken implements the TokenProvider interface for client credentials flow
func (p *ClientCredentialsProvider) GetNewToken() (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.ClientID)
	data.Set("client_secret", p.ClientSecret)
   
	if p.Scope != "" {
		data.Set("scope", p.Scope)
	}
   
	// Add any extra parameters
	for key, value := range p.ExtraParams {
		data.Set(key, value)
	}

	req, err := http.NewRequest("POST", p.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating token request: %v", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := p.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending token request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading token response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("error parsing token response: %v", err)
	}

	return &tokenResp, nil
}
