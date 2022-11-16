package models

type UAACreds struct {
	ClientID     string `yaml:"client_id" json:"clientID"`
	ClientSecret string `yaml:"client_secret" json:"clientSecret"`
	URL          string `yaml:"url" json:"url"`
}
