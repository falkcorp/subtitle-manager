// file: pkg/authserver/server.go
// version: 1.1.0
// guid: d14bd9a1-4b55-4b57-bffc-4adc165ebd1a

package authserver

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	authpb "github.com/falkcorp/gcommon/v2/pkg/commonpb/v2"
	gauth "github.com/jdfalk/subtitle-manager/pkg/gcommonauth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements authpb.AuthServiceServer using the local database
// and gcommonauth helpers.
type Server struct {
	DB *sql.DB
	authpb.UnimplementedAuthServiceServer
}

// NewServer creates a new Server instance.
func NewServer(db *sql.DB) *Server {
	return &Server{DB: db}
}

// Authenticate validates user credentials and issues a session token.
func (s *Server) Authenticate(ctx context.Context, req *authpb.AuthAuthenticateRequest) (*authpb.AuthAuthenticateResponse, error) {
	var userID int64

	if req.HasPassword() {
		creds := req.GetPassword()
		id, err := gauth.AuthenticateUser(s.DB, creds.GetUsername(), creds.GetPassword())
		if err != nil {
			return nil, err
		}
		userID = id
	} else if req.HasApiKey() {
		creds := req.GetApiKey()
		apiKey, err := gauth.ValidateAPIKey(s.DB, creds.GetKey())
		if err != nil {
			return nil, err
		}
		// Extract user ID from gcommon APIKey
		if userIdStr := apiKey.GetUserId(); userIdStr != "" {
			if uid, err := strconv.ParseInt(userIdStr, 10, 64); err == nil {
				userID = uid
			} else {
				return nil, err
			}
		} else {
			return nil, sql.ErrNoRows
		}
	} else {
		return nil, sql.ErrNoRows
	}

	session, err := gauth.GenerateSession(s.DB, userID, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	// Extract session token from gcommon Session
	token := session.GetId()

	// Create UserInfo
	userInfo := &authpb.UserInfo{}
	userInfo.SetUserId(strconv.FormatInt(userID, 10))
	userInfo.SetUsername(credsUsername(req))

	// Create response
	response := &authpb.AuthAuthenticateResponse{}
	response.SetAccessToken(token)
	response.SetTokenType("session")
	response.SetExpiresIn(int32(24 * 60 * 60))
	response.SetUserInfo(userInfo)

	return response, nil
}

func credsUsername(req *authpb.AuthAuthenticateRequest) string {
	if req.HasPassword() {
		return req.GetPassword().GetUsername()
	}
	return ""
}

// ValidateToken verifies a session token and returns expiration info.
func (s *Server) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	var userID int64
	var expires time.Time
	row := s.DB.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token = ?`, req.GetAccessToken())
	if err := row.Scan(&userID, &expires); err != nil {
		if err == sql.ErrNoRows {
			response := &authpb.ValidateTokenResponse{}
			response.SetValid(false)
			return response, nil
		}
		return nil, err
	}
	if time.Now().After(expires) {
		response := &authpb.ValidateTokenResponse{}
		response.SetValid(false)
		return response, nil
	}

	response := &authpb.ValidateTokenResponse{}
	response.SetValid(true)
	response.SetSubject(strconv.FormatInt(userID, 10))
	response.SetExpiresAt(timestamppb.New(expires))
	response.SetExpiresIn(int32(time.Until(expires).Seconds()))

	return response, nil
}
