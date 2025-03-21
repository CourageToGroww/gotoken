// Package main provides the executable entry point for the GoToken application.
// This file demonstrates the usage of the OAuth token management system
// by creating a provider, setting up a token manager, and making authenticated API requests.
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"
    
    "github.com/couragetogroww/gotoken/pkg/oauth"
)

func main() {
    // Example usage of the OAuth token package - create a provider for your OAuth server
    provider := &oauth.ClientCredentialsProvider{
        TokenURL:     "https://example.com/oauth/token",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        Scope:        "read write",
    }
    
    // Create a token manager that will automatically:
    // 1. Generate a new token with a POST request
    // 2. Refresh tokens every 59 minutes (before the standard 60-minute expiry)
    // 3. Apply tokens to all HTTP requests without additional POST requests
    tokenManager := oauth.NewTokenManager(provider,
        // Explicitly set refresh time to 59 minutes (this is the default anyway)
        oauth.WithRefreshTime(59*time.Minute),
        // Set buffer time (how long before expiry to refresh if not using fixed interval)
        oauth.WithBufferTime(1*time.Minute),
        // Get notified when a new token is obtained
        oauth.WithOnNewToken(func(token *oauth.TokenResponse) {
            log.Printf("New token obtained at %s, valid for 60 minutes", time.Now().Format(time.RFC3339))
            log.Printf("The token will be automatically refreshed in 59 minutes")
        }),
    )
    
    // Set up a simple HTTP client that uses the token manager
    client := &http.Client{}
    
    // Simulate making authenticated requests every few seconds
    // This demonstrates how the same token is reused for multiple requests
    // without needing to generate a new token each time
    makeAuthenticatedRequest := func(url string) {
        // Create a new request
        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
            log.Printf("Error creating request: %v", err)
            return
        }
        
        // Wait for a valid token and apply it to the request
        if err := tokenManager.EnsureValidToken(); err != nil {
            log.Printf("Error ensuring valid token: %v", err)
            return
        }
        
        // Apply the authorization header - the TokenManager automatically
        // provides the current token without making new POST requests
        tokenManager.ApplyToRequest(req)
        
        // Make the request
        resp, err := client.Do(req)
        if err != nil {
            log.Printf("Error making request: %v", err)
            return
        }
        defer resp.Body.Close()
        
        log.Printf("Request to %s completed with status: %s", url, resp.Status)
    }
    
    // Print instructions and wait for token initialization
    fmt.Println("GoToken - 59-Minute OAuth Token Manager")
    fmt.Println("====================================")
    fmt.Println("This example demonstrates how tokens are automatically:")
    fmt.Println("1. Generated once with a POST request")
    fmt.Println("2. Refreshed every 59 minutes (before 60-minute expiry)")
    fmt.Println("3. Applied to HTTP headers without additional POST requests")
    fmt.Println("\nWaiting for initial token generation...")
    
    // Wait for the first token to be generated
    tokenManager.WaitForToken()
    fmt.Println("Initial token obtained successfully!")
    fmt.Println("Current auth header: " + tokenManager.GetAuthorizationHeader())
    
    // Make a few requests to demonstrate token reuse
    fmt.Println("\nMaking multiple API requests using the same token...")
    
    // Demonstrate how the same token is used for multiple requests
    // In a real application, these would be spread over time
    makeAuthenticatedRequest("https://httpbin.org/get")
    
    fmt.Println("\nIn a real application, the token would be automatically")
    fmt.Println("refreshed every 59 minutes without user intervention.")
    fmt.Println("Meanwhile, all HTTP requests would continue to receive")
    fmt.Println("the current valid token in their Authorization header.")
    
    // Show how the token will be refreshed in the background
    fmt.Println("\nThis example will now simulate periodic API calls.")
    fmt.Println("A new token will be automatically generated after 59 minutes.")
    fmt.Println("Press Ctrl+C to exit.")
    
    // Start a goroutine to periodically make requests to demonstrate
    // how the same token is reused until it's refreshed
    go func() {
        // Make a request every 10 seconds to demonstrate token reuse
        ticker := time.NewTicker(10 * time.Second)
        for {
            select {
            case <-ticker.C:
                makeAuthenticatedRequest("https://httpbin.org/get")
            }
        }
    }()
    
    // Keep the program running
    select {}
}
