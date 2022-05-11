package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/mdblp/shoreline/token"
	"github.com/tidepool-org/go-common/clients/shoreline"
)

// Config holds the configuration for the Auth Client
type Config struct {
	Address       string `json:"address"`
	ServiceSecret string `json:"serviceSecret"`
	UserAgent     string `json:"userAgent"`
}

// ClientInterface interface that we will implement and mock
type ClientInterface interface {
	GetRestrictedToken(ctx context.Context, id string) (*RestrictedToken, error)
	Authenticate(req *http.Request) *shoreline.TokenData
}

// Client holds the state of the Auth Client
type Client struct {
	config         *Config
	httpClient     *http.Client
	tokenValidator *validator.Validator
}

// CustomClaims contains custom data we want from the token.
type CustomClaims struct {
	Scope    string   `json:"scope"`
	Roles    []string `json:"http://your-loops.com/roles"`
	IsServer bool     `json:"isServer"`
}

// Nothing to validate for tidewhisperer, roles do not matter
// TODO: should we check that the email is verified? info should be in the token
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

func setupAuth0() *validator.Validator {
	//target audience is used to verify the token was issued for a specific domain or url.
	//by default it will be empty but we would (in the future) use this to authorize or deny access to some urls
	targetAudience := []string{}
	if value, present := os.LookupEnv("AUTH0_AUDIENCE"); present {
		targetAudience = []string{value}
	}
	issuerURL, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}
	keyProvider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		keyProvider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		targetAudience,
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
	}

	return jwtValidator
}

// NewClient creates a new Auth Client
func NewClient(config *Config, httpClient *http.Client) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is missing")
	}
	if httpClient == nil {
		return nil, errors.New("http client is missing")
	}
	validator := setupAuth0()

	return &Client{
		config:         config,
		httpClient:     httpClient,
		tokenValidator: validator,
	}, nil
}

// GetRestrictedToken fetches a restricted token from the `auth` service
func (c *Client) GetRestrictedToken(ctx context.Context, id string) (*RestrictedToken, error) {
	if ctx == nil {
		return nil, errors.New("context is missing")
	}
	if id == "" {
		return nil, errors.New("id is missing")
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/restricted_tokens/%s", c.config.Address, id), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	req.Header.Add("X-Tidepool-Service-Secret", c.config.ServiceSecret)
	req.Header.Add("User-Agent", c.config.UserAgent)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status code")
	}

	restrictedToken := &RestrictedToken{}
	if err = json.NewDecoder(res.Body).Decode(restrictedToken); err != nil {
		return nil, err
	}

	return restrictedToken, nil
}

func (client *Client) Authenticate(req *http.Request) *shoreline.TokenData {
	if sessionToken := req.Header.Get("x-tidepool-session-token"); sessionToken != "" {
		tokenData, err := token.UnpackSessionTokenAndVerify(sessionToken, client.config.ServiceSecret)
		//More validations?
		if err != nil {
			log.Print("Error decoding tidepool session token")
			return nil
		}
		return &shoreline.TokenData{UserID: tokenData.UserId, IsServer: tokenData.IsServer}
	} else {
		var parsedToken *validator.ValidatedClaims
		if rawToken, err := jwtmiddleware.AuthHeaderTokenExtractor(req); err != nil {
			log.Print("Error decoding bearer token")
			return nil
		} else if t, err := client.tokenValidator.ValidateToken(req.Context(), rawToken); err != nil {
			log.Print("Error decoding bearer token")
			return nil
		} else {
			parsedToken = t.(*validator.ValidatedClaims)
		}
		uid := strings.Split(parsedToken.RegisteredClaims.Subject, "|")[1]
		return &shoreline.TokenData{UserID: uid, IsServer: false}
	}
}
