// Package oauth provides OAuth token acquisition and management functionality.
// This file implements the TokenManager which handles token lifecycle management,
// refreshing tokens before they expire (typically every 59 minutes for 60-minute tokens)
// and automatically passing them to HTTP request headers without requiring
// new POST requests for each API call.
package oauth

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// TokenManager handles OAuth token lifecycle management
// It automatically refreshes tokens before expiry (every 59 minutes for 60-minute tokens)
// and provides methods to apply tokens to HTTP requests.
type TokenManager struct {
	provider     TokenProvider
	currentToken *TokenResponse
	mutex        sync.RWMutex
	refreshTimer *time.Timer
	tokenReady   chan struct{}
	bufferTime   time.Duration // Time before expiry to refresh the token (default: 1 minute)
	refreshTime  time.Duration // Time between token refreshes (default: 59 minutes for 60-minute tokens)
	onNewToken   func(token *TokenResponse)
}

// NewTokenManager creates a new token manager with the given provider
// By default, it will refresh tokens every 59 minutes (for 60-minute tokens)
func NewTokenManager(provider TokenProvider, options ...TokenManagerOption) *TokenManager {
	tm := &TokenManager{
		provider:    provider,
		tokenReady:  make(chan struct{}),
		bufferTime:  60 * time.Second,     // Default buffer time is 60 seconds
		refreshTime: 59 * time.Minute,     // Default refresh time is 59 minutes (for 60-minute tokens)
	}
   
	// Apply options
	for _, option := range options {
		option(tm)
	}
   
	go tm.run()
	return tm
}

// TokenManagerOption defines a function type for configuring the TokenManager
type TokenManagerOption func(*TokenManager)

// WithBufferTime sets the buffer time before token expiry to refresh
func WithBufferTime(duration time.Duration) TokenManagerOption {
	return func(tm *TokenManager) {
		tm.bufferTime = duration
	}
}

// WithRefreshTime sets a custom refresh time for tokens
// For 60-minute tokens, the recommended value is 59 minutes
func WithRefreshTime(duration time.Duration) TokenManagerOption {
	return func(tm *TokenManager) {
		tm.refreshTime = duration
	}
}

// WithOnNewToken sets a callback function that will be called when a new token is obtained
func WithOnNewToken(callback func(token *TokenResponse)) TokenManagerOption {
	return func(tm *TokenManager) {
		tm.onNewToken = callback
	}
}

// run starts the token refresh loop
// This continuously monitors token expiration and automatically refreshes tokens
// before they expire, by default at 59 minutes for 60-minute tokens.
func (tm *TokenManager) run() {
	for {
		// Generate a new token with a POST request
		token, err := tm.provider.GetNewToken()
		if err != nil {
			log.Printf("Error refreshing token: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Update the current token securely
		tm.mutex.Lock()
		tm.currentToken = token
		tm.mutex.Unlock()

		// Signal that the token is ready for use in HTTP headers
		select {
		case <-tm.tokenReady:
		default:
			close(tm.tokenReady)
		}
	   
		// Call the onNewToken callback if set
		if tm.onNewToken != nil {
			tm.onNewToken(token)
		}

		// Determine when to refresh the token
		var refreshTime time.Duration
		if tm.refreshTime > 0 {
			// Use the explicit refresh time (default 59 minutes)
			refreshTime = tm.refreshTime
			log.Printf("Token will be refreshed in %v (using fixed refresh interval)", refreshTime)
		} else {
			// Calculate based on token's expiry minus buffer time
			refreshTime = time.Duration(token.ExpiresIn)*time.Second - tm.bufferTime
			if refreshTime < 0 {
				refreshTime = 5 * time.Second // If token is about to expire, refresh soon
			}
			log.Printf("Token will be refreshed in %v (using token expiry time)", refreshTime)
		}
	   
		// Wait until it's time to refresh
		tm.refreshTimer = time.NewTimer(refreshTime)
		<-tm.refreshTimer.C
		log.Printf("Refreshing token after %v", refreshTime)
	   
		// Create a new tokenReady channel for the next token
		tm.mutex.Lock()
		tm.tokenReady = make(chan struct{})
		tm.mutex.Unlock()
	}
}

// WaitForToken blocks until a token is available
func (tm *TokenManager) WaitForToken() {
	<-tm.tokenReady
}

// GetToken returns the current access token
func (tm *TokenManager) GetToken() string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	if tm.currentToken == nil {
		return ""
	}
	return tm.currentToken.AccessToken
}

// GetFullToken returns the complete token response
func (tm *TokenManager) GetFullToken() *TokenResponse {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	return tm.currentToken
}

// EnsureValidToken ensures a valid token is available
func (tm *TokenManager) EnsureValidToken() error {
	tm.WaitForToken()
	return nil
}

// GetAuthorizationHeader returns a complete authorization header value
func (tm *TokenManager) GetAuthorizationHeader() string {
	token := tm.GetFullToken()
	if token == nil {
		return ""
	}
	return token.TokenType + " " + token.AccessToken
}

// ApplyToRequest adds the authorization header to an HTTP request
func (tm *TokenManager) ApplyToRequest(req *http.Request) {
	authHeader := tm.GetAuthorizationHeader()
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
}
