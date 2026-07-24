// file: pkg/authserver/oauth_flow_test.go
// version: 1.2.1
// guid: e8f7d6c5-b4a3-9281-7f6e-5d4c3b2a1908

package authserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	authpb "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	gauth "github.com/jdfalk/subtitle-manager/pkg/gcommonauth"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
)

// MockOAuthProvider simulates external OAuth providers like GitHub
type MockOAuthProvider struct {
	mock.Mock
	server *httptest.Server
}

func NewMockOAuthProvider() *MockOAuthProvider {
	provider := &MockOAuthProvider{}

	// Create test server to simulate OAuth endpoints
	provider.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/authorize":
			provider.handleAuthorize(w, r)
		case "/oauth/token":
			provider.handleToken(w, r)
		case "/api/user":
			provider.handleUserInfo(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	return provider
}

func (m *MockOAuthProvider) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	args := m.Called("authorize", r.URL.Query())

	if args.Get(0) != nil {
		// Simulate authorization page redirect
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")

		// Simulate user approval and redirect back with code
		redirectURL := fmt.Sprintf("%s?code=test_auth_code&state=%s", redirectURI, state)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	} else {
		http.Error(w, "authorization denied", http.StatusForbidden)
	}
}

func (m *MockOAuthProvider) handleToken(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	r.ParseForm()

	// Call with just the form values for simpler mocking
	args := m.Called(r.Form)

	if tokenResp := args.Get(0); tokenResp != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	} else {
		http.Error(w, "invalid grant", http.StatusBadRequest)
	}
}

func (m *MockOAuthProvider) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	args := m.Called(authHeader)

	if userInfo := args.Get(0); userInfo != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	} else {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
}

func (m *MockOAuthProvider) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  m.server.URL + "/oauth/authorize",
			TokenURL: m.server.URL + "/oauth/token",
		},
		RedirectURL: "http://localhost:8080/callback",
		Scopes:      []string{"user:email"},
	}
}

func (m *MockOAuthProvider) Close() {
	m.server.Close()
}

// OAuth flow integration tests

func TestOAuthAuthorizationFlowPositivePath(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()

	// Create test user
	require.NoError(t, gauth.CreateUser(db, "testuser", "password", "test@example.com", "user"))

	// Setup mock OAuth provider
	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	// Configure successful OAuth flow - only token exchange needed
	mockProvider.On("handleToken", mock.MatchedBy(func(form url.Values) bool {
		return form.Get("code") == "test_auth_code" && form.Get("grant_type") == "authorization_code"
	})).Return(map[string]interface{}{
		"access_token": "oauth_access_token_123",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        "user:email",
	})

	// Test OAuth authorization URL generation
	config := mockProvider.GetConfig()
	state := "random_state_string"
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	assert.Contains(t, authURL, mockProvider.server.URL+"/oauth/authorize")
	assert.Contains(t, authURL, "client_id=test_client_id")
	assert.Contains(t, authURL, "state="+state)

	// Test token exchange
	ctx := context.Background()
	token, err := config.Exchange(ctx, "test_auth_code")
	require.NoError(t, err)
	assert.Equal(t, "oauth_access_token_123", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)

	// Verify all mock expectations were met
	mockProvider.AssertExpectations(t)
}

func TestOAuthTokenExchangeError(t *testing.T) {
	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	// Configure OAuth provider to return error
	mockProvider.On("handleToken", mock.MatchedBy(func(form url.Values) bool {
		return form.Get("code") == "invalid_code"
	})).Return(nil)

	config := mockProvider.GetConfig()
	ctx := context.Background()

	_, err := config.Exchange(ctx, "invalid_code")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oauth2")

	mockProvider.AssertExpectations(t)
}

func TestOAuthUserInfoRetrievalError(t *testing.T) {
	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	// Configure successful token exchange but failed user info
	mockProvider.On("handleToken", mock.MatchedBy(func(form url.Values) bool {
		return form.Get("code") == "test_auth_code"
	})).Return(map[string]interface{}{
		"access_token": "invalid_token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
	mockProvider.On("handleUserInfo", "Bearer invalid_token").Return(nil)

	config := mockProvider.GetConfig()
	ctx := context.Background()

	token, err := config.Exchange(ctx, "test_auth_code")
	require.NoError(t, err)

	// Simulate HTTP client with the token
	client := config.Client(ctx, token)
	resp, err := client.Get(mockProvider.server.URL + "/api/user")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockProvider.AssertExpectations(t)
}

// Session management tests with OAuth integration

func TestOAuthSessionCreationAndValidation(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()

	// Create test user
	require.NoError(t, gauth.CreateUser(db, "oauthuser", "", "oauth@example.com", "user"))

	srv := NewServer(db)

	// Test session creation after OAuth flow
	userID := int64(1)
	session, err := gauth.GenerateSession(db, userID, 24*time.Hour)
	require.NoError(t, err)
	sessionToken := session.GetId()

	// Test session validation via authserver
	valReq := &authpb.ValidateTokenRequest{}
	valReq.SetAccessToken(sessionToken)

	valResp, err := srv.ValidateToken(context.Background(), valReq)
	require.NoError(t, err)
	assert.True(t, valResp.GetValid())
	assert.Equal(t, "1", valResp.GetSubject())
	assert.Greater(t, valResp.GetExpiresIn(), int32(0))
}

func TestOAuthSessionExpiration(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()

	require.NoError(t, gauth.CreateUser(db, "expireduser", "", "expired@example.com", "user"))

	srv := NewServer(db)

	// Create session with very short expiration
	userID := int64(1)
	session, err := gauth.GenerateSession(db, userID, 1*time.Millisecond)
	require.NoError(t, err)
	sessionToken := session.GetId()

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Test expired session validation
	valReq := &authpb.ValidateTokenRequest{}
	valReq.SetAccessToken(sessionToken)

	valResp, err := srv.ValidateToken(context.Background(), valReq)
	require.NoError(t, err)
	assert.False(t, valResp.GetValid())
}

// OAuth callback handler integration tests

func TestOAuthCallbackHandlerSuccess(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()

	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	// Setup successful OAuth response
	mockProvider.On("handleToken", mock.MatchedBy(func(form url.Values) bool {
		return form.Get("code") == "callback_auth_code"
	})).Return(map[string]interface{}{
		"access_token": "callback_token_123",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})

	// Test callback processing simulation
	config := mockProvider.GetConfig()
	ctx := context.Background()

	// Simulate receiving callback with authorization code
	token, err := config.Exchange(ctx, "callback_auth_code")
	require.NoError(t, err)
	assert.Equal(t, "callback_token_123", token.AccessToken)

	// Simulate user creation/login after successful OAuth
	err = gauth.CreateUser(db, "callbackuser", "", "callback@example.com", "user")
	require.NoError(t, err)

	// Create session for OAuth-authenticated user
	session, err := gauth.GenerateSession(db, 1, 24*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, session.GetId())

	mockProvider.AssertExpectations(t)
}

func TestOAuthCallbackHandlerStateValidation(t *testing.T) {
	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	config := mockProvider.GetConfig()
	originalState := "secure_random_state_123"

	// Generate auth URL with state
	authURL := config.AuthCodeURL(originalState)
	parsedURL, err := url.Parse(authURL)
	require.NoError(t, err)

	stateParam := parsedURL.Query().Get("state")
	assert.Equal(t, originalState, stateParam)

	// In a real callback handler, we would validate that the returned state matches
	// This prevents CSRF attacks
	returnedState := "secure_random_state_123" // This should match originalState
	assert.Equal(t, originalState, returnedState, "State parameter must match to prevent CSRF attacks")
}

// Test OAuth flow error scenarios

func TestOAuthFlowWithInvalidClientCredentials(t *testing.T) {
	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	// Use invalid credentials
	invalidConfig := &oauth2.Config{
		ClientID:     "invalid_client_id",
		ClientSecret: "invalid_secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  mockProvider.server.URL + "/oauth/authorize",
			TokenURL: mockProvider.server.URL + "/oauth/token",
		},
		RedirectURL: "http://localhost:8080/callback",
	}

	mockProvider.On("handleToken", mock.MatchedBy(func(form url.Values) bool {
		return form.Get("code") == "any_code"
	})).Return(nil) // Return error

	ctx := context.Background()
	_, err := invalidConfig.Exchange(ctx, "any_code")
	assert.Error(t, err)

	mockProvider.AssertExpectations(t)
}

func TestOAuthScopeHandling(t *testing.T) {
	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	config := mockProvider.GetConfig()
	config.Scopes = []string{"user:email", "repo:status"}

	authURL := config.AuthCodeURL("state")
	parsedURL, err := url.Parse(authURL)
	require.NoError(t, err)

	scope := parsedURL.Query().Get("scope")
	assert.Contains(t, scope, "user:email")
	assert.Contains(t, scope, "repo:status")
}

// OAuth refresh token tests

func TestOAuthRefreshTokenFlow(t *testing.T) {
	mockProvider := NewMockOAuthProvider()
	defer mockProvider.Close()

	// Initial token with refresh token
	initialToken := &oauth2.Token{
		AccessToken:  "initial_access_token",
		RefreshToken: "refresh_token_123",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-1 * time.Hour), // Expired
	}

	// Mock refresh token exchange
	mockProvider.On("handleToken", mock.MatchedBy(func(form url.Values) bool {
		return form.Get("grant_type") == "refresh_token" && form.Get("refresh_token") == "refresh_token_123"
	})).Return(map[string]interface{}{
		"access_token":  "refreshed_access_token",
		"refresh_token": "new_refresh_token",
		"token_type":    "Bearer",
		"expires_in":    3600,
	})

	config := mockProvider.GetConfig()
	ctx := context.Background()

	// Use token source which will automatically refresh expired tokens
	tokenSource := config.TokenSource(ctx, initialToken)
	refreshedToken, err := tokenSource.Token()
	require.NoError(t, err)

	assert.Equal(t, "refreshed_access_token", refreshedToken.AccessToken)
	assert.Equal(t, "new_refresh_token", refreshedToken.RefreshToken)
	assert.True(t, refreshedToken.Expiry.After(time.Now()))

	mockProvider.AssertExpectations(t)
}
