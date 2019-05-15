package oidcutil

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/coreos/go-oidc"
	"log"
)

type Verifier struct {
	verifier *oidc.IDTokenVerifier
	verifiedEmailClaimCheck bool
}

func NewIdTokenVerifier(verifier *oidc.IDTokenVerifier, verifiedEmailClaimCheck bool) *Verifier {
	return &Verifier{
		verifier: verifier,
		verifiedEmailClaimCheck: verifiedEmailClaimCheck,
	}
}

func (v *Verifier) Verify(ctx context.Context, rawIDToken string) (*TokenWrapper, error) {
	idToken, err := v.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	claims, err := NewClaims(idToken)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	if v.verifiedEmailClaimCheck && !claims.EmailVerified {
		err = errors.New("email_verified claim must be true")
		log.Fatal(err)
		return nil, err
	}

	var c json.RawMessage
	_ = idToken.Claims(&c)
	log.Printf(
		"DEBUG: fetched id token successfuly: rawIdToken=%v, idToken=%+v, claims=%+v, allclaims=%s",
		rawIDToken, idToken, claims, c,
	)

	return &TokenWrapper{
		IDToken: idToken,
		RawIDToken: rawIDToken,
		Claims: claims,
	}, nil
}