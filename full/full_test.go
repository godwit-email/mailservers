package full

import (
	"slices"
	"testing"
)

func TestFullKeepsWhatTheRootPackageDrops(t *testing.T) {
	var plain, pop3, other, oauth bool
	for _, p := range Providers() {
		plain = plain || slices.ContainsFunc(p.IMAP, func(s Server) bool { return s.Security == "plain" })
		pop3 = pop3 || len(p.POP3) > 0
		other = other || len(p.Other) > 0
		oauth = oauth || p.OAuth2 != nil
	}
	if !plain || !pop3 || !other || !oauth {
		t.Fatalf("plain=%v pop3=%v other=%v oauth=%v", plain, pop3, other, oauth)
	}
}
