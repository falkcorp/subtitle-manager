// file: pkg/authserver/server_test.go
// version: 1.2.1
// guid: c5c8d260-2641-45dc-80d2-4dbb941bdb7e

package authserver

import (
	"context"
	"testing"

	authpb "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	gauth "github.com/jdfalk/subtitle-manager/pkg/gcommonauth"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
)

// TestAuthenticatePassword verifies password-based authentication
func TestAuthenticatePassword(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()

	testutil.MustNoError(t, "create user", gauth.CreateUser(db, "u1", "pass", "e@example.com", "user"))

	srv := NewServer(db)
	req := &authpb.AuthAuthenticateRequest{}
	creds := &authpb.PasswordCredentials{}
	creds.SetUsername("u1")
	creds.SetPassword("pass")
	req.SetPassword(creds)
	resp, err := srv.Authenticate(context.Background(), req)
	testutil.MustNoError(t, "authenticate", err)
	testutil.MustNotEqual(t, "token", "", resp.GetAccessToken())

	valReq := &authpb.ValidateTokenRequest{}
	valReq.SetAccessToken(resp.GetAccessToken())
	val, err := srv.ValidateToken(context.Background(), valReq)
	testutil.MustNoError(t, "validate", err)
	testutil.MustEqual(t, "valid", true, val.GetValid())
}

// TestAuthenticateAPIKey verifies API key authentication
func TestAuthenticateAPIKey(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()

	testutil.MustNoError(t, "create user", gauth.CreateUser(db, "u2", "pass", "", "user"))
	key, err := gauth.GenerateAPIKey(db, 1)
	testutil.MustNoError(t, "generate key", err)

	srv := NewServer(db)

	// Create API key credentials
	apiKeyCreds := &authpb.APIKeyCredentials{}
	apiKeyCreds.SetKey(key.GetId())

	// Create the request with API key credentials
	req := &authpb.AuthAuthenticateRequest{}
	req.SetApiKey(apiKeyCreds)

	resp, err := srv.Authenticate(context.Background(), req)
	testutil.MustNoError(t, "authenticate", err)
	testutil.MustNotEqual(t, "token", "", resp.GetAccessToken())
}
