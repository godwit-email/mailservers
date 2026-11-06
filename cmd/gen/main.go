// Command gen merges overlay/ over upstream ISPDB into min.json and
// full/full.json. With -check it fails if those files are out of date.
package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/godwit-email/mailservers/internal/schema"
)

type xmlConfig struct {
	Provider xmlProvider `xml:"emailProvider"`
	OAuth2   *struct {
		Issuer   string `xml:"issuer"`
		Scope    string `xml:"scope"`
		AuthURL  string `xml:"authURL"`
		TokenURL string `xml:"tokenURL"`
	} `xml:"oAuth2"`
	WebMail struct {
		LoginPage struct {
			URL string `xml:"url,attr"`
		} `xml:"loginPage"`
	} `xml:"webMail"`
}

type xmlProvider struct {
	Domains   []string    `xml:"domain"`
	MX        []string    `xml:"mx"` // our extension to the ISPDB format
	Name      string      `xml:"displayName"`
	ShortName string      `xml:"displayShortName"`
	Incoming  []xmlServer `xml:"incomingServer"`
	Outgoing  []xmlServer `xml:"outgoingServer"`
	Docs      []struct {
		URL   string    `xml:"url,attr"`
		Descr []xmlText `xml:"descr"`
	} `xml:"documentation"`
	Enable []struct {
		URL          string    `xml:"visiturl,attr"`
		Instructions []xmlText `xml:"instruction"`
	} `xml:"enable"`
}

type xmlText struct {
	Lang string `xml:"lang,attr"`
	Text string `xml:",chardata"`
}

type xmlServer struct {
	Type     string   `xml:"type,attr"`
	Host     string   `xml:"hostname"`
	Port     string   `xml:"port"`
	Socket   string   `xml:"socketType"`
	Username string   `xml:"username"`
	Auth     []string `xml:"authentication"`
	URL      string   `xml:"url"`
	OwaURL   string   `xml:"owaURL"`
}

func main() {
	upstream := flag.String("upstream", "upstream/ispdb", "ISPDB directory")
	overlay := flag.String("overlay", "overlay", "our own entries, which win over upstream")
	minPath := flag.String("min", "min.json", "output for the root package")
	fullPath := flag.String("full", "full/full.json", "output for the full package")
	check := flag.Bool("check", false, "fail if the outputs are out of date")
	flag.Parse()

	full, err := merge(*overlay, *upstream)
	if err != nil {
		log.Fatal(err)
	}
	stale := false
	for path, v := range map[string]any{*minPath: minimal(full), *fullPath: full} {
		data, err := encode(v)
		if err != nil {
			log.Fatal(err)
		}
		if *check {
			if old, err := os.ReadFile(path); err != nil || !bytes.Equal(old, data) {
				fmt.Fprintf(os.Stderr, "%s is out of date: run go run ./cmd/gen\n", path)
				stale = true
			}
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	if stale {
		os.Exit(1)
	}
}

// merge reads the directories in order: an earlier one wins a file name (which
// is the provider's ID) and each domain.
func merge(dirs ...string) ([]schema.FullProvider, error) {
	var out []schema.FullProvider
	ids, domains := map[string]bool{}, map[string]bool{}
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no XML files in %s (is the submodule checked out?)", dir)
		}
		sort.Strings(files)
		for _, file := range files {
			id := strings.TrimSuffix(filepath.Base(file), ".xml")
			if ids[id] {
				continue
			}
			ids[id] = true
			p, err := read(file, id)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			p.Domains = slices.DeleteFunc(p.Domains, func(d string) bool {
				seen := domains[d]
				domains[d] = true
				return seen
			})
			if len(p.Domains) > 0 || len(p.MX) > 0 {
				out = append(out, p)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func read(file, id string) (schema.FullProvider, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return schema.FullProvider{}, err
	}
	var c xmlConfig
	if err := xml.Unmarshal(data, &c); err != nil {
		return schema.FullProvider{}, err
	}
	x := c.Provider
	p := schema.FullProvider{
		Provider: schema.Provider{
			ID:   id,
			Name: clean(x.Name),
			IMAP: []schema.Server{},
			SMTP: []schema.Server{},
		},
		ShortName:  clean(x.ShortName),
		WebmailURL: clean(c.WebMail.LoginPage.URL),
	}
	for _, list := range [][]string{x.Domains, x.MX} {
		for i, d := range list {
			list[i] = strings.ToLower(clean(d))
		}
	}
	p.Domains, p.MX = dedup(x.Domains), dedup(x.MX)
	for _, s := range append(x.Incoming, x.Outgoing...) {
		srv, err := server(s)
		if err != nil {
			return p, err
		}
		switch s.Type {
		case "imap":
			p.IMAP = append(p.IMAP, srv)
		case "smtp":
			p.SMTP = append(p.SMTP, srv)
		case "pop3":
			p.POP3 = append(p.POP3, srv)
		default:
			srv.Type = s.Type
			p.Other = append(p.Other, srv)
		}
	}
	for _, d := range x.Docs {
		p.Docs = append(p.Docs, schema.Doc{URL: clean(d.URL), Description: texts(d.Descr)})
	}
	if len(p.Docs) > 0 {
		p.HelpURL = p.Docs[0].URL
	}
	if len(x.Enable) > 0 {
		p.Enable = &schema.Enable{URL: clean(x.Enable[0].URL), Instructions: texts(x.Enable[0].Instructions)}
	}
	if o := c.OAuth2; o != nil {
		p.OAuth2 = &schema.OAuth2{Issuer: clean(o.Issuer), Scope: clean(o.Scope), AuthURL: clean(o.AuthURL), TokenURL: clean(o.TokenURL)}
	}
	return p, nil
}

func server(s xmlServer) (schema.Server, error) {
	srv := schema.Server{
		Host:     strings.ReplaceAll(strings.ToLower(clean(s.Host)), "%emaildomain%", "%EMAILDOMAIN%"),
		Username: clean(s.Username),
		URL:      clean(s.URL + s.OwaURL),
	}
	switch socket := clean(s.Socket); socket {
	case "SSL":
		srv.Security = "tls"
	case "STARTTLS":
		srv.Security = "starttls"
	case "plain":
		srv.Security = "plain"
	case "":
	default:
		return srv, fmt.Errorf("unknown socketType %q", socket)
	}
	for _, a := range s.Auth {
		srv.Auth = append(srv.Auth, clean(a))
	}
	if port := clean(s.Port); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return srv, fmt.Errorf("bad port %q", port)
		}
		srv.Port = n
	}
	return srv, nil
}

// minimal keeps the providers reachable over IMAP and SMTP without plaintext.
func minimal(full []schema.FullProvider) []schema.Provider {
	out := []schema.Provider{}
	for _, f := range full {
		p := f.Provider
		p.IMAP, p.SMTP = encrypted(p.IMAP), encrypted(p.SMTP)
		if len(p.IMAP) > 0 && len(p.SMTP) > 0 {
			out = append(out, p)
		}
	}
	return out
}

func encrypted(servers []schema.Server) []schema.Server {
	out := []schema.Server{}
	for _, s := range servers {
		if s.Host != "" && s.Port != 0 && (s.Security == "tls" || s.Security == "starttls") {
			out = append(out, s)
		}
	}
	return out
}

func encode(v any) ([]byte, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	e.SetIndent("", " ") // one field a line, so a sync reads as a diff
	err := e.Encode(v)
	return b.Bytes(), err
}

func texts(list []xmlText) map[string]string {
	if len(list) == 0 {
		return nil
	}
	m := map[string]string{}
	for _, t := range list {
		m[t.Lang] = clean(t.Text)
	}
	return m
}

func dedup(list []string) []string {
	var out []string
	for _, s := range list {
		if s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func clean(s string) string { return strings.Join(strings.Fields(s), " ") }
