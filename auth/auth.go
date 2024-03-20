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
	Username string
	Email    string
}

func New(ctx context.Context, cfg config.SpiritAuthConfig) (*Auth, error) {
	auth, err := authentication.New(
		ctx,
		cfg.Domain,
		authentication.WithClientID(cfg.ClientID),
		authentication.WithClientSecret(cfg.ClientSecret),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize the auth0 API client: %+v", err)
	}
	return &Auth{
		auth,
	}, nil
}

type Auth struct {
	auth *authentication.Authentication
}

// / Try to sign up the requested credentials
func (a *Auth) RequestSignUp(
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
