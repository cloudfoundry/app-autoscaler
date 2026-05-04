package models

const GrantTypePassword = "password"

type UAACreds struct {
	URL               string `yaml:"url" json:"url"`
	ClientID          string `yaml:"client_id" json:"clientID"`
	ClientSecret      string `yaml:"client_secret" json:"clientSecret"`
	GrantType         string `yaml:"grant_type" json:"grantType"`
	Username          string `yaml:"username" json:"username"`
	Password          string `yaml:"password" json:"password"`
	SkipSSLValidation bool   `yaml:"skip_ssl_validation" json:"skipSSLValidation"`
}

func (c UAACreds) IsNotEmpty() bool {
	return c.URL != ""
}

func (c UAACreds) IsPasswordGrant() bool {
	return c.GrantType == GrantTypePassword
}
