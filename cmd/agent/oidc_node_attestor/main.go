package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/everpeace/oidc_attestor_plugin/pkg"
	plugin "github.com/everpeace/oidc_attestor_plugin/pkg/agent/nodeattestor"
	goplugin "github.com/hashicorp/go-plugin"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spiffe/spire/proto/agent/nodeattestor"
)

func main() {
	var versionMode = flag.Bool("version", false, "Show version")
	flag.Parse()
	if *versionMode {
		fmt.Printf("oidc_node_attestor %s (revision: %s)", pkg.Version, pkg.Revision)
		return
	}

	oidcNodeAttestorPlugin := plugin.New()
	sigCh := make(chan os.Signal, 1)
	doneCh := make(chan struct{})
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		for {
			<-sigCh
			log.Print("plugin received interrupt signal, shutdown oidc client")
			ctx, _:= context.WithTimeout(context.Background(), 5*time.Second)
			oidcNodeAttestorPlugin.Shutdown(ctx)
			doneCh<- struct{}{}
		}
	}()

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

	<-doneCh
}
