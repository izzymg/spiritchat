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
var ErrUserExists = errors.New("that user already exists")

type UserData struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	IsVerified bool   `json:"-"`
}

type Auth interface {
	RequestSignUp(
		ctx context.Context,
		username string, email string, password string,
	) (*UserData, error)
	GetUserFromToken(ctx context.Context, token string) (*UserData, error)
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
		Username:   username,
		Email:      email,
		Password:   password,
		Connection: "Username-Password-Authentication",
	})
	if err != nil {

		if strings.Contains(err.Error(), "invalid_password") {
			return nil, ErrInvalidPassword
		}
		if strings.Contains(err.Error(), "invalid_username") {
			return nil, ErrInvalidUsername
		}
		if strings.Contains(err.Error(), "invalid_email") {
			return nil, ErrInvalidEmail
		}
		if strings.Contains(err.Error(), "invalid_signup") {
			return nil, ErrUserExists
		}

		return nil, err
	}
	return &UserData{
		Username: res.Username,
		Email:    res.Email,
	}, nil
}

func (a *OAuth) GetUserFromToken(ctx context.Context, token string) (*UserData, error) {
	info, err := a.auth.UserInfo(ctx, token)
	if err != nil {
		return nil, err
	}
	return &UserData{
		Username:   info.PreferredUsername,
		Email:      info.Email,
		IsVerified: info.EmailVerified,
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
