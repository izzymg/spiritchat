package auth

import (
	"context"
	"errors"
	"fmt"
	"spiritchat/config"
	"strings"

	"github.com/auth0/go-auth0/authentication"
	"github.com/auth0/go-auth0/authentication/database"
)

var ErrInvalidUsername = errors.New("invalid username")
var ErrInvalidEmail = errors.New("invalid email")
var ErrInvalidPassword = errors.New("invalid password")

type UserData struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Auth interface {
	RequestSignUp(
		ctx context.Context,
		username string, email string, password string,
	) (*UserData, error)
}

type OAuth struct {
	auth *authentication.Authentication
}

// / Try to sign up the requested credentials
func (a *OAuth) RequestSignUp(
	ctx context.Context,
	username string, email string, password string,
) (*UserData, error) {
	res, err := a.auth.Database.Signup(ctx, database.SignupRequest{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {

		if strings.Contains(err.Error(), "invalid_pasword") {
			return nil, ErrInvalidPassword
		}
		if strings.Contains(err.Error(), "invalid_username") {
			return nil, ErrInvalidUsername
		}
		if strings.Contains(err.Error(), "invalid_email") {
			return nil, ErrInvalidEmail
		}

		return nil, err
	}
	return &UserData{
		Username: res.Username,
		Email:    res.Email,
	}, nil
}

func NewOAuth(ctx context.Context, cfg config.SpiritAuthConfig) (*OAuth, error) {
	auth, err := authentication.New(
		ctx,
		cfg.Domain,
		authentication.WithClientID(cfg.ClientID),
		authentication.WithClientSecret(cfg.ClientSecret),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize the auth0 API client: %+v", err)
	}
	return &OAuth{
		auth,
	}, nil
}
