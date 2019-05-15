package nodeattestor

import (
	"context"
	"errors"
	"github.com/everpeace/oidc_attestor_plugin/pkg"
	"github.com/everpeace/oidc_attestor_plugin/pkg/common"
	"github.com/everpeace/oidc_attestor_plugin/pkg/oidcutil"
	"github.com/spiffe/spire/proto/agent/nodeattestor"
	"log"
	"sync"
	"time"

	spc "github.com/spiffe/spire/proto/common"
	spi "github.com/spiffe/spire/proto/common/plugin"
)

var _ nodeattestor.Plugin = &Plugin{}

type Plugin struct {
	mtx *sync.RWMutex

	config *Config
	client *oidcutil.Client
}

func New() *Plugin {
	return &Plugin{
		mtx: &sync.RWMutex{},
	}
}

func (p *Plugin) FetchAttestationData(stream nodeattestor.FetchAttestationData_PluginStream) error {
	log.Print("DEBUG: start FetchAttestationData")

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if err := p.assertConfigured(); err != nil {
		return errors.New("plugin not configured")
	}

	ctx, _ := context.WithTimeout(context.Background(), 60 * time.Second)
	t, err := p.client.Authenticate(ctx)

	if err != nil {
		return err
	}

	spiffeId := p.config.GenerateSpiffeId(p.config.TrustDomain, t.Claims)
	log.Printf("DEBUG: spiffeID=%s", spiffeId)

	resp := &nodeattestor.FetchAttestationDataResponse{
		AttestationData: &spc.AttestationData{
			Type: pkg.PluginName,
			Data: []byte(t.RawIDToken),
		},
		SpiffeId: spiffeId,
	}
	if err := stream.Send(resp); err != nil {
		log.Fatal("failed sending FetchAttestationDataResponse", err)
		return err
	}

	log.Print("DEBUG: start FetchAttestationData")
	return nil
}

func (p *Plugin) Configure(ctx context.Context, req *spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	log.Print("DEBUG: start Configure")
	config, err := NewConfig(req)
	if err != nil {
		return nil, err
	}
	log.Printf("DEBUG: loaded configuration successfuly: %+v", config)

	p.config = config
	verifiedEmailClaimCheck := false
	if p.config.Mode == common.IDGenModeEmail {
		verifiedEmailClaimCheck = true
	}
	client, err := oidcutil.NewClient(
		p.config.IssuerURL, p.config.ClientID, p.config.ClientSecret, verifiedEmailClaimCheck,
	)
	if err != nil {
		return nil, err
	}
	p.client = client

	log.Print("DEBUG: finish Configure")
	return &spi.ConfigureResponse{}, nil
}

func (p *Plugin) GetPluginInfo(context.Context, *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

func (p *Plugin) Shutdown(ctx context.Context) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	if p.client == nil {
		return
	}
	childCtx, _ := context.WithCancel(ctx)
	if err := p.client.Shutdown(childCtx); err != nil {
		log.Fatal(err)
	}
}

func (p *Plugin) assertConfigured() error {
	if p.config == nil || p.client == nil {
		return errors.New("plugin not configured")
	}
	return nil
}
