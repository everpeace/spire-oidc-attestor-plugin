package config

import (
	"errors"
	"github.com/hashicorp/go-multierror"
)

type Common struct{
	TrustDomain   string
	IssuerURL     string `hcl:"issuer_url"`
	ClientID      string `hcl:"client_id"`
}

func (c *Common) Validate() (err error) {
	if c.TrustDomain == "" {
		err = multierror.Append(err, errors.New("trust_domain must not be empty"))
	}
	if c.IssuerURL == "" {
		err = multierror.Append(err, errors.New("issuer_id must not be empty"))
	}
	if c.ClientID == "" {
		err = multierror.Append(err, errors.New("client_id must not be empty"))
	}
	return
}
