// Package schema holds the types shared by the generated data, the generator
// and the public packages.
package schema

// Server is one way to reach a mail server.
type Server struct {
	Type     string   `json:"type,omitempty"` // only for servers that are not IMAP, SMTP or POP3
	Host     string   `json:"host,omitempty"`
	Port     int      `json:"port,omitempty"`
	Security string   `json:"security,omitempty"` // tls, starttls or plain
	Username string   `json:"username,omitempty"` // a template; see Login in the root package
	Auth     []string `json:"auth,omitempty"`     // ISPDB names: password-cleartext, password-encrypted, OAuth2, none
	URL      string   `json:"url,omitempty"`
}

// Enable says what the user must do first, such as turning IMAP on in webmail.
type Enable struct {
	URL          string            `json:"url,omitempty"`
	Instructions map[string]string `json:"instructions,omitempty"` // by language; "" when untagged
}

// Provider is what the root package serves: never a plaintext server.
type Provider struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Domains []string `json:"domains,omitempty"`
	MX      []string `json:"mx,omitempty"` // see the root package's ByMX
	IMAP    []Server `json:"imap"`
	SMTP    []Server `json:"smtp"`
	HelpURL string   `json:"helpUrl,omitempty"`
	Enable  *Enable  `json:"enable,omitempty"`
}

// OAuth2 holds a provider's endpoints; clients register their own ID with it.
type OAuth2 struct {
	Issuer   string `json:"issuer,omitempty"`
	Scope    string `json:"scope,omitempty"`
	AuthURL  string `json:"authUrl,omitempty"`
	TokenURL string `json:"tokenUrl,omitempty"`
}

// Doc is a link to the provider's documentation.
type Doc struct {
	URL         string            `json:"url"`
	Description map[string]string `json:"description,omitempty"`
}

// FullProvider is Provider with every server kept, plaintext ones included,
// and the rest of what ISPDB says.
type FullProvider struct {
	Provider
	ShortName  string   `json:"shortName,omitempty"`
	POP3       []Server `json:"pop3,omitempty"`
	Other      []Server `json:"other,omitempty"` // Exchange, OWA, Graph…
	OAuth2     *OAuth2  `json:"oauth2,omitempty"`
	WebmailURL string   `json:"webmailUrl,omitempty"`
	Docs       []Doc    `json:"docs,omitempty"`
}
