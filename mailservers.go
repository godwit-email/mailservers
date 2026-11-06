// Package mailservers finds the IMAP and SMTP servers of an email domain, from
// Thunderbird's ISPDB and our own entries. It makes no network requests: the
// caller looks up MX records and passes them to ByMX.
package mailservers

import (
	_ "embed"
	"encoding/json"
	"path"
	"strings"

	"github.com/godwit-email/mailservers/internal/schema"
)

type (
	Provider = schema.Provider
	Server   = schema.Server
	Enable   = schema.Enable
)

//go:embed min.json
var data []byte

var providers, byDomain = load()

func load() ([]Provider, map[string]int) {
	var list []Provider
	if err := json.Unmarshal(data, &list); err != nil {
		panic("mailservers: " + err.Error()) // generated and tested in this module
	}
	index := map[string]int{}
	for i, p := range list {
		for _, d := range p.Domains {
			index[d] = i
		}
	}
	return list, index
}

// ByDomain returns the provider that hosts domain.
func ByDomain(domain string) (Provider, bool) {
	domain = normalize(domain)
	i, ok := byDomain[domain]
	if !ok {
		return Provider{}, false
	}
	return resolve(providers[i], domain, ""), true
}

// ByMX returns the provider of domain, whose MX host is mx: first by the MX
// rules of our own entries, then, as Thunderbird does, by mx or a parent of it
// being a provider's domain.
//
// A rule with no wildcard matches its host and subdomains. In one with a
// wildcard, * and {cluster} match one label or part of one, and {cluster} fills
// the same placeholder in the provider's servers. The rule with the most labels wins.
func ByMX(domain, mx string) (Provider, bool) {
	domain, mx = normalize(domain), normalize(mx)
	best, bestLabels, cluster := -1, 0, ""
	for i, p := range providers {
		for _, rule := range p.MX {
			if c, ok := match(rule, mx); ok && strings.Count(rule, ".")+1 > bestLabels {
				best, bestLabels, cluster = i, strings.Count(rule, ".")+1, c
			}
		}
	}
	if best >= 0 {
		return resolve(providers[best], domain, cluster), true
	}
	for h := mx; strings.Contains(h, "."); {
		if i, ok := byDomain[h]; ok {
			return resolve(providers[i], domain, ""), true
		}
		_, h, _ = strings.Cut(h, ".")
	}
	return Provider{}, false
}

// Login returns the username s wants for the address email.
func Login(s Server, email string) string {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return s.Username
	}
	return strings.NewReplacer("%EMAILADDRESS%", email, "%EMAILLOCALPART%", email[:at], "%EMAILDOMAIN%", email[at+1:]).Replace(s.Username)
}

func match(rule, host string) (cluster string, ok bool) {
	if !strings.ContainsAny(rule, "*{") {
		return "", host == rule || strings.HasSuffix(host, "."+rule)
	}
	ruleLabels, hostLabels := strings.Split(rule, "."), strings.Split(host, ".")
	if len(ruleLabels) != len(hostLabels) {
		return "", false
	}
	for i, l := range ruleLabels {
		if l == "{cluster}" {
			cluster = hostLabels[i]
			continue
		}
		if ok, err := path.Match(l, hostLabels[i]); err != nil || !ok {
			return "", false
		}
	}
	return cluster, true
}

// resolve fills the placeholders of p's servers, in a copy.
func resolve(p Provider, domain, cluster string) Provider {
	r := strings.NewReplacer("%EMAILDOMAIN%", domain, "{cluster}", cluster)
	fill := func(list []Server) []Server {
		out := make([]Server, len(list))
		for i, s := range list {
			s.Host = r.Replace(s.Host)
			out[i] = s
		}
		return out
	}
	p.Name, p.IMAP, p.SMTP = r.Replace(p.Name), fill(p.IMAP), fill(p.SMTP)
	return p
}

func normalize(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
}
