package oidcutil

import (
	"errors"
	"github.com/coreos/go-oidc"
	"github.com/hashicorp/go-multierror"
)

type Claims struct {
	Issuer string `json:"iss"`
	Subject string `json:"sub"`
	Email   string `json:"email"`
	EmailVerified bool `json:"email_verified"`
}

func NewClaims(idToken *oidc.IDToken) (*Claims, error) {
	claims := Claims{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	if err := claims.Validate(); err != nil {
		return nil, err
	}
	return &claims, nil
}

func (c Claims) Validate() (err error) {
	if c.Issuer == "" {
		err = multierror.Append(err, errors.New("issuer claim must not be empty"))
	}
	if c.Subject == "" {
		err = multierror.Append(err, errors.New("subject claim must not be empty"))
	}
	if c.Email == "" {
		err = multierror.Append(err, errors.New("email claim must not be empty"))
	}
	return err
}
