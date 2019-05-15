package oidcutil

import (
	"context"
	"errors"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"log"
)

type IDTokenSource struct {
	verifier *Verifier
	ts oauth2.TokenSource
}

type TokenWrapper struct {
	IDToken    *oidc.IDToken
	RawIDToken string
	Claims     *Claims
}

func NewIDTokenSource(verifier *oidc.IDTokenVerifier, ts oauth2.TokenSource, verifiedEmailClaimCheck bool) *IDTokenSource {
	return &IDTokenSource{
		verifier: NewIdTokenVerifier(verifier, verifiedEmailClaimCheck),
		ts:       ts,
	}
}

func (s *IDTokenSource) Token() (*TokenWrapper,  error) {
	oauth2Token, err := s.ts.Token()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		err := errors.New("id_token is not a string")
		log.Fatal(err)
		return nil, err
	}
	return s.verifier.Verify(context.Background(), rawIDToken)
}
