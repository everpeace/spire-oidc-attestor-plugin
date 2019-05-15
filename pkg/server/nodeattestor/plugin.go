package nodeattestor

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc"
	"github.com/everpeace/oidc_attestor_plugin/pkg"
	"github.com/everpeace/oidc_attestor_plugin/pkg/common"
	"github.com/everpeace/oidc_attestor_plugin/pkg/oidcutil"
	spc "github.com/spiffe/spire/proto/common"
	spi "github.com/spiffe/spire/proto/common/plugin"
	"github.com/spiffe/spire/proto/server/nodeattestor"
	"log"
	"strings"
	"sync"
)

var _ nodeattestor.Plugin = &Plugin{}
type Plugin struct {
	mtx *sync.RWMutex

	config *Config
	provider *oidc.Provider
	verifier *oidcutil.Verifier
}

func New() *Plugin {
	return &Plugin{
		mtx: &sync.RWMutex{},
	}
}

func (p *Plugin) Attest(stream nodeattestor.Attest_PluginStream) error {
	log.Print("DEBUG: start Attest")

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if err := p.assertConfigured(); err != nil {
		return errors.New("plugin not configured")
	}

	returnInvalid := func() error {
		err := stream.Send(&nodeattestor.AttestResponse{
			Valid:        false,
		})
		if err != nil {
			log.Fatal("failed sending AttestResponse{Valid: true}:", err)
			return err
		}
		return nil
	}

	req, err := stream.Recv()
	if err != nil {
		log.Fatal(err)
		return returnInvalid()
	}

	rawIDToken := string(req.AttestationData.Data)
	t, err := p.verifier.Verify(context.Background(), rawIDToken)

	if err != nil {
		log.Fatal(err)
		return returnInvalid()
	}

	var selectors []*spc.Selector
	switch p.config.Mode {
	case common.IDGenModeIssuerAndSubject:
		selectors = []*spc.Selector {
			{
				Type:  pkg.PluginName,
				Value: fmt.Sprintf("issuer:%s", t.Claims.Issuer),
			},
			{
				Type:  pkg.PluginName,
				Value: fmt.Sprintf("subject:%s", t.Claims.Subject),
			},
		}
	case common.IDGenModeEmail:
		selectors = []*spc.Selector {
			{
				Type:  pkg.PluginName,
				Value: fmt.Sprintf("issuer:%s", t.Claims.Issuer),
			},
			{
				Type:  pkg.PluginName,
				Value: fmt.Sprintf("email:%s", t.Claims.Email),
			},
			{
				Type:  pkg.PluginName,
				Value: fmt.Sprintf("email_verified:%v", t.Claims.EmailVerified),
			},
		}
	}

	// should we whitelisted??
	resp := &nodeattestor.AttestResponse{
		Valid:        true,
		BaseSPIFFEID: p.config.GenerateSpiffeId(p.config.TrustDomain, t.Claims),
		Selectors: selectors,
	}

	if err := stream.Send(resp); err != nil {
		log.Fatalf(
			"failed sending AttestResponse: err=%+v, resp=%+v",
			err, resp,
		)
		return err
	}
	strSelectors := ""
	for _, s:= range resp.GetSelectors() {
		strSelectors += fmt.Sprintf("%+v,", s)
	}
	strings.TrimRight(strSelectors, ",")
	log.Printf(
		"INFO: %s node attest finished: Spiffe-id: %s, selectors: %s",
		pkg.PluginName,
		resp.GetBaseSPIFFEID(),
		strSelectors,
	)
	log.Print("DEBUG: finish Attest")
	return nil
}

func (p *Plugin) Configure(ctx context.Context, req *spi.ConfigureRequest) (*spi.ConfigureResponse, error) {
	log.Print("DEBUG: start Configure")

	p.mtx.Lock()
	defer p.mtx.Unlock()

	config, err := NewConfig(req)
	if err != nil {
		return &spi.ConfigureResponse{ErrorList: []string{err.Error()}}, nil
	}
	log.Printf("DEBUG: loaded configuration successfuly: %+v", config)

	p.config = config
	provider, err := oidc.NewProvider(context.Background(), config.IssuerURL)
	if err != nil {
		return &spi.ConfigureResponse{ErrorList: []string{err.Error()}}, nil
	}
	p.provider = provider
	verifiedEmailClaimCheck := false
	if p.config.Mode == common.IDGenModeEmail {
		verifiedEmailClaimCheck = true
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: config.ClientID})
	p.verifier = oidcutil.NewIdTokenVerifier(idTokenVerifier, verifiedEmailClaimCheck)

	log.Print("DEBUG: finish Configure")
	return &spi.ConfigureResponse{}, nil
}

func (p *Plugin) GetPluginInfo(context.Context, *spi.GetPluginInfoRequest) (*spi.GetPluginInfoResponse, error) {
	return &spi.GetPluginInfoResponse{}, nil
}

func (p *Plugin) assertConfigured() error {
	if p.config == nil || p.provider == nil || p.verifier == nil {
		return errors.New("plugin not configured")
	}
	return nil
}
