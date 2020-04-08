package auth

import (
	"net/http"
	"strings"
	"time"
)

// RestrictedToken holds data for a restricted token from the `auth` service
type RestrictedToken struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	Paths          *[]string  `json:"paths,omitempty"`
	ExpirationTime time.Time  `json:"expirationTime"`
	CreatedTime    time.Time  `json:"createdTime"`
	ModifiedTime   *time.Time `json:"modifiedTime,omitempty"`
}

// Authenticates determines whether the req has a valid restricted token.
func (r *RestrictedToken) Authenticates(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	if time.Now().After(r.ExpirationTime) {
		return false
	}
	if r.Paths != nil {
		escapedPath := req.URL.EscapedPath()
		for _, path := range *r.Paths {
			if path == escapedPath || strings.HasPrefix(escapedPath, strings.TrimSuffix(path, "/")+"/") {
				return true
			}
		}
		return false
	}
	return true
}
