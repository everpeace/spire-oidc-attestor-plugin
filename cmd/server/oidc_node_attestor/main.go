package main

import (
	"flag"
	"fmt"
	"github.com/everpeace/oidc_attestor_plugin/pkg"
	plugin "github.com/everpeace/oidc_attestor_plugin/pkg/server/nodeattestor"
	goplugin "github.com/hashicorp/go-plugin"

	"github.com/spiffe/spire/proto/server/nodeattestor"
)

func main() {
	var versionMode = flag.Bool("version", false, "Show version")
	flag.Parse()
	if *versionMode {
		fmt.Printf("oidc_node_attestor %s (revision: %s)", pkg.Version, pkg.Revision)
		return
	}
	oidcNodeAttestorPlugin := plugin.New()
	goplugin.Serve(&goplugin.ServeConfig{
		Plugins: map[string]goplugin.Plugin{
			pkg.PluginName: nodeattestor.GRPCPlugin{
				ServerImpl: &nodeattestor.GRPCServer{
					Plugin: oidcNodeAttestorPlugin,
				},
			},
		},
		HandshakeConfig: nodeattestor.Handshake,
		GRPCServer:      goplugin.DefaultGRPCServer,
	})
}
