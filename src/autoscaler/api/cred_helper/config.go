package cred_helper

import "github.com/hashicorp/go-plugin"

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CREDENTIALS_PLUGIN",
	MagicCookieValue: "somerandomstring",
}
