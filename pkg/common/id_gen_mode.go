package common

import (
	"fmt"
	"github.com/everpeace/oidc_attestor_plugin/pkg"
	"github.com/everpeace/oidc_attestor_plugin/pkg/oidcutil"
	"log"
	"net/url"
	"path"
	"strings"
)

type IDGenMode struct {
	Mode Mode `hcl:"mode"`
}

type Mode string
var (
	IDGenModeIssuerAndSubject Mode = "issuer_and_subject"
	IDGenModeEmail            Mode = "email"
	agentPathPrefix                = path.Join("spire", "agent", pkg.PluginName)
)

func (m IDGenMode) Validate() (err error) {
	if !(m.Mode == IDGenModeIssuerAndSubject || m.Mode == IDGenModeEmail) {
		err = fmt.Errorf("mode must be one of %s",
			strings.Join([]string{string(IDGenModeIssuerAndSubject), string(IDGenModeEmail)}, ","),
		)
	}
	return
}

func (m IDGenMode) GenerateSpiffeId(trustDomain string, claims *oidcutil.Claims) string {
	var spiffePath string
	switch m.Mode {
	case IDGenModeIssuerAndSubject:
		spiffePath = path.Join(
			agentPathPrefix, url.PathEscape(claims.Issuer), url.PathEscape(claims.Subject),
		)
	case IDGenModeEmail:
		spiffePath = path.Join(
			agentPathPrefix, url.PathEscape(claims.Email),
		)
	}
	log.Printf("DEBUG: spiffePath=%s", spiffePath)
	id := &url.URL{
		Scheme: "spiffe",
		Host:   trustDomain,
		Path:   spiffePath,
	}

	return id.String()
}
