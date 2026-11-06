package mailservers

import (
	"slices"
	"testing"
)

func TestByMX(t *testing.T) {
	for _, tc := range []struct{ mx, imap, smtp string }{
		{"mx.c3.icoremail.net.", "c3.icoremail.net", "c3.icoremail.net"},
		{"MX2.C3.icoremail.net", "c3.icoremail.net", "c3.icoremail.net"},
		{"mx.icoremail.net.", "imap.icoremail.net", "smtp.icoremail.net"},
		{"icoremail.net.", "imap.icoremail.net", "smtp.icoremail.net"},
		{"a.mx.c3.icoremail.net.", "imap.icoremail.net", "smtp.icoremail.net"},
		{"mxn.mxhichina.com.", "imap.qiye.aliyun.com", "smtp.qiye.aliyun.com"},
		{"qiye163mx01.mxmail.netease.com.", "imap.qiye.163.com", "smtp.qiye.163.com"},
		{"hzmx01.mxmail.netease.com.", "imap.qiye.163.com", "smtp.qiye.163.com"},
		{"mx.ym.163.com.", "imap.ym.163.com", "smtp.ym.163.com"},
		{"mx.263xmail.com.", "imap.263.net", "smtp.263.net"},
		{"mx.entmail.sina.com.", "imap.sina.net", "smtp.sina.net"},
		{"in1-smtp.messagingengine.com.", "imap.fastmail.com", "smtp.fastmail.com"},
		{"mx.zoho.com.cn.", "imappro.zoho.com.cn", "smtppro.zoho.com.cn"},
		{"mx.zoho.com.", "imappro.zoho.com", "smtppro.zoho.com"},
		{"mail.posteo.de.", "posteo.de", "posteo.de"}, // a parent of the MX host is a domain upstream knows
	} {
		p, ok := ByMX("corp.example", tc.mx)
		if !ok || p.IMAP[0].Host != tc.imap || p.SMTP[0].Host != tc.smtp {
			t.Errorf("%s: got %v %+v, want %s / %s", tc.mx, ok, p, tc.imap, tc.smtp)
		}
	}
	for _, mx := range []string{"mx.unknown.example.", "mxmail.netease.com.", "other.mxmail.netease.com.", "notzoho.com.", ""} {
		if p, ok := ByMX("corp.example", mx); ok {
			t.Errorf("%s matched %s", mx, p.ID)
		}
	}
}

func TestByDomain(t *testing.T) {
	p, ok := ByDomain("AOL.de.")
	if !ok || p.IMAP[0].Host != "imap.aol.com" {
		t.Fatalf("aol.de: %v %+v", ok, p)
	}
	if _, ok := ByDomain("no-such-domain.example"); ok {
		t.Fatal("found an unknown domain")
	}
	// A placeholder takes the domain asked about, in a copy.
	p, ok = ByDomain("inbox.lv")
	if !ok || p.IMAP[0].Host != "mail.inbox.lv" || p.Name != "inbox.lv" {
		t.Fatalf("inbox.lv: %v %+v", ok, p)
	}
	if providers[byDomain["inbox.lv"]].IMAP[0].Host != "mail.%EMAILDOMAIN%" {
		t.Fatal("resolve changed the shared data")
	}
}

func TestPersonalZohoKeepsUpstreamServers(t *testing.T) {
	if p, ok := ByDomain("zoho.com"); !ok || p.IMAP[0].Host != "imap.zoho.com" {
		t.Fatalf("zoho.com: %v %+v", ok, p)
	}
}

func TestLogin(t *testing.T) {
	for username, want := range map[string]string{
		"%EMAILADDRESS%":                 "a@b.example",
		"%EMAILLOCALPART%":               "a",
		"%EMAILLOCALPART%.%EMAILDOMAIN%": "a.b.example",
		"fixed":                          "fixed",
	} {
		if got := Login(Server{Username: username}, "a@b.example"); got != want {
			t.Errorf("%s: got %s, want %s", username, got, want)
		}
	}
}

func TestDataIsSafeToServe(t *testing.T) {
	if len(providers) < 100 {
		t.Fatalf("only %d providers", len(providers))
	}
	seen := map[string]bool{}
	for _, p := range providers {
		if len(p.IMAP) == 0 || len(p.SMTP) == 0 {
			t.Errorf("%s has no IMAP or SMTP", p.ID)
		}
		for _, s := range slices.Concat(p.IMAP, p.SMTP) {
			if s.Host == "" || s.Port == 0 || s.Security != "tls" && s.Security != "starttls" {
				t.Errorf("%s: unsafe or incomplete server %+v", p.ID, s)
			}
		}
		for _, d := range p.Domains {
			if seen[d] {
				t.Errorf("%s: domain %s is served twice", p.ID, d)
			}
			seen[d] = true
		}
	}
}
