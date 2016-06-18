package fakes

import (
	"metrics-collector/cf"
)

type FakeCfClient struct {
	accessToken     string
	dopplerEndpiont string
	tokens          cf.Tokens
	endpoints       cf.Endpoints
}

func NewFakeCfClient(token string, doppler string) cf.CfClient {
	return &FakeCfClient{
		accessToken:     token,
		dopplerEndpiont: doppler,
	}
}

func (f *FakeCfClient) Login() error {
	f.tokens.AccessToken = f.accessToken
	f.endpoints.DopplerEndpoint = f.dopplerEndpiont
	return nil
}

func (f *FakeCfClient) GetTokens() cf.Tokens {
	return f.tokens
}

func (f *FakeCfClient) GetEndpoints() cf.Endpoints {
	return f.endpoints
}
