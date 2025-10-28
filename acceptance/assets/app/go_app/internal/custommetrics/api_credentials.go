package api

import "context"

type CustomMetricsCredentials struct {
	Username string `mapstructure:"username" json:"username"`
	Password string `mapstructure:"password" json:"password"`
	URL      string `mapstructure:"url" json:"url"`
	MtlsURL  string `mapstructure:"mtls_url" json:"mtls_url"`
}

func (c CustomMetricsCredentials) BasicAuthentication(ctx context.Context, operationName string) (BasicAuthentication, error) {
	credentials := BasicAuthentication{
		Username: c.Username,
		Password: c.Password,
	}

	return credentials, nil
}
