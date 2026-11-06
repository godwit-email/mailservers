// Command classify compares two min.json files. It lists what changed and exits
// 0 if a provider or address was only added, or 2 if anything an existing address
// relies on changed, which a person must review: the data says where passwords go.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/godwit-email/mailservers/internal/schema"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: classify old.json new.json")
	}
	before, after := load(os.Args[1]), load(os.Args[2])
	var added, changed []string
	for key, v := range after {
		switch old, ok := before[key]; {
		case !ok:
			added = append(added, key)
		case old != v:
			changed = append(changed, key+" -> "+v)
		}
	}
	for key := range before {
		if _, ok := after[key]; !ok {
			changed = append(changed, key+" removed")
		}
	}
	sort.Strings(added)
	sort.Strings(changed)
	for _, s := range added {
		fmt.Println("added:", s)
	}
	for _, s := range changed {
		fmt.Println("changed:", s)
	}
	if len(changed) > 0 {
		os.Exit(2)
	}
}

// load maps each domain and MX rule to the servers it leads to.
func load(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var list []schema.Provider
	if err := json.Unmarshal(data, &list); err != nil {
		log.Fatal(err)
	}
	out := map[string]string{}
	for _, p := range list {
		servers, err := json.Marshal([][]schema.Server{p.IMAP, p.SMTP})
		if err != nil {
			log.Fatal(err)
		}
		for _, d := range p.Domains {
			out["domain "+d] = string(servers)
		}
		for _, m := range p.MX {
			out["mx "+m] = string(servers)
		}
	}
	return out
}
