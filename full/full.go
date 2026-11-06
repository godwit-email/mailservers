// Package full serves everything we know about each provider, plaintext servers
// included, unlike the root package. Only programs that import it carry its data.
package full

import (
	_ "embed"
	"encoding/json"

	"github.com/godwit-email/mailservers/internal/schema"
)

type (
	Provider = schema.FullProvider
	Server   = schema.Server
	OAuth2   = schema.OAuth2
	Doc      = schema.Doc
)

//go:embed full.json
var data []byte

// Providers returns a fresh copy of every provider, sorted by ID. Hosts keep
// their %EMAILDOMAIN% and {cluster} placeholders.
func Providers() []Provider {
	var list []Provider
	if err := json.Unmarshal(data, &list); err != nil {
		panic("mailservers/full: " + err.Error()) // generated and tested in this module
	}
	return list
}
