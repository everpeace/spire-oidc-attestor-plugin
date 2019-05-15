package config

import (
	"errors"
	"github.com/hashicorp/go-multierror"
)

type Agent struct {
	Common       `hcl:",squash"`
	ClientSecret string    `hcl:"client_secret"`
}

func (c *Agent) Validate() (err error) {
	err = c.Common.Validate()
	if c.ClientSecret == "" {
		err = multierror.Append(err, errors.New("client_secret must not be empty"))
	}
	return
}
