package nodeattestor

import (
	"errors"
	"fmt"
	"github.com/everpeace/oidc_attestor_plugin/pkg/common"
	"github.com/everpeace/oidc_attestor_plugin/pkg/config"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
	spi "github.com/spiffe/spire/proto/common/plugin"
)

type Config struct {
	config.Agent     `hcl:",squash"`
	common.IDGenMode `hcl:",squash"`
}

func (c *Config) Validate() (err error) {
	if _err := c.Agent.Validate(); _err != nil {
		err = multierror.Append(err, _err)
	}

	if _err := c.IDGenMode.Validate(); _err != nil {
		err = multierror.Append(err, _err)
	}

	return
}

func NewConfig(req *spi.ConfigureRequest) (*Config, error) {
	config := &Config{}
	if err := hcl.Decode(config, req.Configuration); err != nil {
		return nil, fmt.Errorf("failed to decode configuration file: %v", err)
	}

	if req.GlobalConfig == nil {
		return nil, errors.New("global configuration is required")
	}
	config.TrustDomain = req.GlobalConfig.TrustDomain
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

