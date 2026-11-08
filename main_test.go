package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseURLHTTP(t *testing.T) {
	m := parseURL("https://calamitymod.wiki.gg/wiki/Bladecrest_Oathsword", parentProc{path: "/usr/bin/x", cmdline: "/usr/bin/x --flag arg"})
	want := map[string]string{
		"protocol": "https", "fulldomain": "calamitymod.wiki.gg", "tld": "gg",
		"subdomain": "calamitymod", "maindomainonly": "wiki", "maindomain": "wiki.gg",
		"path": "/wiki/Bladecrest_Oathsword", "perams": "", "port": "", "hash": "",
		"parentprocesspath": "/usr/bin/x", "parentprocesscmdline": "/usr/bin/x --flag arg",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %q, want %q", k, m[k], v)
		}
	}
}

func TestParseURLPortQueryHash(t *testing.T) {
	m := parseURL("http://user@Example.COM:8080/a/b?x=1&y=2#frag", parentProc{})
	for k, v := range map[string]string{
		"fulldomain": "example.com", "port": "8080", "path": "/a/b",
		"perams": "?x=1&y=2", "hash": "#frag", "subdomain": "",
	} {
		if m[k] != v {
			t.Errorf("%s = %q, want %q", k, m[k], v)
		}
	}
}

func TestParseURLNoScheme(t *testing.T) {
	m := parseURL("asd.test.com/asd", parentProc{})
	for k, v := range map[string]string{
		"protocol": "", "fulldomain": "asd.test.com", "subdomain": "asd",
		"maindomainonly": "test", "maindomain": "test.com", "tld": "com", "path": "/asd",
	} {
		if m[k] != v {
			t.Errorf("%s = %q, want %q", k, m[k], v)
		}
	}
	m = parseURL("localhost:3000/x?a=1", parentProc{})
	if m["protocol"] != "" || m["fulldomain"] != "localhost" || m["port"] != "3000" || m["path"] != "/x" || m["perams"] != "?a=1" {
		t.Errorf("%v", m)
	}
}

func TestParseURLFile(t *testing.T) {
	m := parseURL("/home/me/docs/report.final.pdf", parentProc{})
	for k, v := range map[string]string{
		"protocol": "file", "path": "/home/me/docs", "fileName": "report.final",
		"FileExt": "pdf", "FileName And Ext": "report.final.pdf",
	} {
		if m[k] != v {
			t.Errorf("%s = %q, want %q", k, m[k], v)
		}
	}
	m = parseURL("file:///tmp/a%20b.html", parentProc{})
	if m["FileName And Ext"] != "a b.html" || m["path"] != "/tmp" {
		t.Errorf("file url parsed wrong: %v", m)
	}
}

func TestParseURLOtherSchemes(t *testing.T) {
	m := parseURL("mailto:a@b.c", parentProc{})
	if m["protocol"] != "mailto" || m["fulldomain"] != "" {
		t.Errorf("%v", m)
	}
	if m := parseURL("localhost", parentProc{}); m["protocol"] != "" || m["fulldomain"] != "localhost" || m["tld"] != "" {
		t.Errorf("%v", m)
	}
	if m := parseURL("http://127.0.0.1:3000/", parentProc{}); m["fulldomain"] != "127.0.0.1" || m["tld"] != "" || m["port"] != "3000" {
		t.Errorf("%v", m)
	}
}

func TestMatch(t *testing.T) {
	cases := []struct {
		v, typ, p string
		want      bool
	}{
		{"/wiki/X/", "is", "\\wiki\\x", true},
		{"abc", "isexact", "ABC", false},
		{"abc", "isexact", "abc", true},
		{"https://a", "startswith", "https", true},
		{"a.gg", "endswith", ".gg", true},
		{"hello", "includes", "ell", true},
		{"hello", "matchesregex", "^h.*o$", true},
		{"hello", "matchesregex", "^x", false},
	}
	for _, c := range cases {
		got, err := doTheyMatch(c.v, c.typ, c.p)
		if err != nil || got != c.want {
			t.Errorf("%v: got %v, %v", c, got, err)
		}
	}
	if _, err := doTheyMatch("a", "matchesregex", "("); err == nil {
		t.Error("bad regex must error")
	}
	if _, err := doTheyMatch("a", "nope", "a"); err == nil {
		t.Error("bad match type must error")
	}
}

func TestExpandArgs(t *testing.T) {
	m := map[string]string{"url": "https://x", "fulldomain": "x"}
	got, err := expandArgs([]string{"--new-window", "$url", "--d=$fulldomain", "$browserArgs"}, m, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--new-window", "https://x", "--d=x", "a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	if _, err := expandArgs([]string{"$nope"}, m, nil); err == nil {
		t.Error("unknown variable must error")
	}
}

const testCfg = `{
  "$schema": "./settings.schema.jsonc",
  "settings": {"defaultBrowser": ""},
  "programs": {
    "zeta": {"path": "zeta"},
    "alpha": {"path": ["a1", "a2"], "args": ["$url"], "fallback": "zeta"},
    "hidden": {"path": "h", "visible": false}
  },
  "rules": [
    {"matches": [["fulldomain", "endswith", ".gg"], ["path", "includes", "wiki"]], "name": "alpha"},
    {"matches": [["tld", "is", "bad"]], "name": ""}
  ]
}`

func TestConfig(t *testing.T) {
	c, err := parseConfig("t.json", []byte(testCfg))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.ProgramOrder, []string{"zeta", "alpha", "hidden"}) {
		t.Errorf("order %v", c.ProgramOrder)
	}
	if !reflect.DeepEqual(c.visiblePrograms(), []string{"zeta", "alpha"}) {
		t.Errorf("visible %v", c.visiblePrograms())
	}
	if !reflect.DeepEqual(fallbackChain(c, "alpha"), []string{"alpha", "zeta"}) {
		t.Error("chain")
	}
	r, err := c.findRule(parseURL("https://calamitymod.wiki.gg/wiki/x", parentProc{}))
	if err != nil || r == nil || r.Name != "alpha" {
		t.Errorf("rule %v %v", r, err)
	}
	r, _ = c.findRule(parseURL("https://example.com/", parentProc{}))
	if r != nil {
		t.Errorf("unexpected rule %v", r)
	}
	r, _ = c.findRule(parseURL("https://x.bad/", parentProc{}))
	if r == nil || r.Name != "" {
		t.Errorf("block rule %v", r)
	}
}

func TestConfigErrors(t *testing.T) {
	bad := map[string]string{
		"unknown setting":       `{"settings":{"titleBar":"x"},"programs":{},"rules":[]}`,
		"missing rules":         `{"settings":{},"programs":{}}`,
		"no path":               `{"settings":{},"programs":{"a":{}},"rules":[]}`,
		"bad fallback":          `{"settings":{},"programs":{"a":{"path":"a","fallback":"q"}},"rules":[]}`,
		"bad key":               `{"settings":{},"programs":{"a":{"path":"a"}},"rules":[{"matches":[["nope","is","x"]],"name":"a"}]}`,
		"bad rule prog":         `{"settings":{},"programs":{"a":{"path":"a"}},"rules":[{"matches":[],"name":"q"}]}`,
		"bad default":           `{"settings":{"defaultBrowser":"q"},"programs":{"a":{"path":"a"}},"rules":[]}`,
		"unknown program field": `{"settings":{},"programs":{"a":{"path":"a","admin":true}},"rules":[]}`,
	}
	for name, js := range bad {
		if _, err := parseConfig("t.json", []byte(js)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestAppendRulePreservesOrder(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(testCfg), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := parseConfig(p, []byte(testCfg))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.appendRule(Rule{Name: "zeta", Matches: [][]string{{"fulldomain", "is", "x.com"}}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if strings.Index(s, `"zeta"`) > strings.Index(s, `"alpha"`) {
		t.Error("programs order changed")
	}
	c2, err := parseConfig(p, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.Rules) != 3 || c2.Rules[2].Name != "zeta" {
		t.Errorf("rules %v", c2.Rules)
	}
	if err := c.appendRule(Rule{Name: "nope"}); err == nil {
		t.Error("invalid rule must be rejected")
	}
}

const modCfg = `{
  "settings": {},
  "programs": {"p": {"path": "p"}},
  "modifiers": {
    "unwrap": {"path": ["nope-not-here", "sh"], "args": ["-c", "echo \"${1##*u=}\"", "sh"]},
    "upper": {"path": "sh", "args": ["-c", "echo \"$1\" | tr a-z A-Z", "sh"]},
    "fails": {"path": "sh", "args": ["-c", "exit 3", "sh"]},
    "silent": {"path": "sh", "args": ["-c", "true", "sh"]},
    "missing": {"path": "definitely-not-installed"}
  },
  "rules": [{"name": "p", "matches": [["tld", "is", "com"]], "modifiers": ["unwrap", "upper"]}]
}`

func TestModifiers(t *testing.T) {
	c, err := parseConfig("t.json", []byte(modCfg))
	if err != nil {
		t.Fatal(err)
	}
	got, err := applyModifiers(c, c.Rules[0].Modifiers, "https://x.com/redir?u=https://real.org/a")
	if err != nil || got != "HTTPS://REAL.ORG/A" {
		t.Errorf("got %q, %v", got, err)
	}
	if got, err := applyModifiers(c, nil, "https://a"); err != nil || got != "https://a" {
		t.Errorf("no modifiers must leave url alone: %q %v", got, err)
	}
	for _, n := range []string{"fails", "silent", "missing"} {
		if _, err := applyModifiers(c, []string{n}, "https://a"); err == nil {
			t.Errorf("%s: expected error", n)
		}
	}
}

func TestModifierConfigErrors(t *testing.T) {
	bad := map[string]string{
		"undefined modifier": `{"settings":{},"programs":{"p":{"path":"p"}},"rules":[{"name":"p","matches":[],"modifiers":["nope"]}]}`,
		"modifier on block":  `{"settings":{},"programs":{},"modifiers":{"m":{"path":"x"}},"rules":[{"name":"","matches":[],"modifiers":["m"]}]}`,
		"no path":            `{"settings":{},"programs":{},"modifiers":{"m":{}},"rules":[]}`,
		"unknown field":      `{"settings":{},"programs":{},"modifiers":{"m":{"path":"x","admin":true}},"rules":[]}`,
	}
	for name, js := range bad {
		if _, err := parseConfig("t.json", []byte(js)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestParentProcCmdline(t *testing.T) {
	// the real process we are running in, read the same way the program does
	got := processCmdline(os.Getpid())
	if !strings.Contains(got, ".test") {
		t.Errorf("cmdline of the test binary = %q", got)
	}
	if strings.ContainsRune(got, 0) {
		t.Errorf("NUL left in %q", got)
	}
	if p := findParentProc(); p.path == "" {
		t.Error("expected to find a parent process")
	}
}
