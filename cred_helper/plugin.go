package cred_helper

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type CredentialsPlugin struct {
	Impl Credentials
}

func (p *CredentialsPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &CredentialsRPCServer{Impl: p.Impl}, nil
}

func (CredentialsPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &CredentialsRPCClient{client: c}, nil
}
