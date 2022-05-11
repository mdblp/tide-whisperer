package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/tidepool-org/go-common/clients/shoreline"
)

type ClientMock struct {
	RestrictedToken *RestrictedToken
	ServerToken     string
	Unauthorized    bool
	UserID          string
	IsServer        bool
}

func NewMock(token string) *ClientMock {
	now := time.Now()
	rtoken := &RestrictedToken{
		ExpirationTime: now.Add(time.Hour * 24),
		CreatedTime:    now,
		ModifiedTime:   &now,
	}
	return &ClientMock{
		RestrictedToken: rtoken,
		ServerToken:     token,
		Unauthorized:    false,
		UserID:          "123.456.789",
		IsServer:        true,
	}
}

func (c *ClientMock) GetRestrictedToken(ctx context.Context, id string) (*RestrictedToken, error) {
	c.RestrictedToken.ID = id
	c.RestrictedToken.UserID = id
	return c.RestrictedToken, nil
}

func (client *ClientMock) Authenticate(req *http.Request) *shoreline.TokenData {
	if client.Unauthorized {
		return nil
	}

	if sessionToken := req.Header.Get("x-tidepool-session-token"); sessionToken != "" {
		return &shoreline.TokenData{UserID: client.UserID, IsServer: client.IsServer}
	} else if req.Header.Get("authorization") != "" {
		return &shoreline.TokenData{UserID: client.UserID, IsServer: client.IsServer}
	}
	return nil
}
