package models

type UAACreds struct {
	URL               string `yaml:"url" json:"url"`
	ClientID          string `yaml:"client_id" json:"clientID"`
	ClientSecret      string `yaml:"client_secret" json:"clientSecret"`
	SkipSSLValidation bool   `yaml:"skip_ssl_validation" json:"skipSSLValidation"`
}

func (c UAACreds) IsNotEmpty() bool {
	return c.URL != ""
}
