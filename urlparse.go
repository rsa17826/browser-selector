package main

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

// Same keys the AHK version exposes to rules, in the same order.
var matchKeys = []string{
	"url", "protocol", "fulldomain", "tld", "subdomain", "maindomainonly",
	"maindomain", "parentprocesspath", "parentprocesscmdline", "path", "perams", "port", "hash",
	"fileName", "drive", "FileExt", "FileName And Ext",
}

var schemeRe = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):`)

// "host:3000/x" looks like scheme "host" to schemeRe; a port number gives it away.
var portRe = regexp.MustCompile(`^\d+([/?#]|$)`)

// parseURL returns every match key; parts that don't exist for this URL are "".
func parseURL(raw string, parent parentProc) map[string]string {
	m := make(map[string]string, len(matchKeys))
	for _, k := range matchKeys {
		m[k] = ""
	}
	m["url"] = raw
	m["parentprocesspath"] = parent.path
	m["parentprocesscmdline"] = parent.cmdline

	if strings.HasPrefix(raw, "/") {
		fileParts(m, raw)
		return m
	}
	sm := schemeRe.FindStringSubmatch(raw)
	if sm == nil || portRe.MatchString(raw[len(sm[0]):]) {
		// "asd.test.com/asd" or "localhost:3000/x": no scheme, so it's a host.
		// protocol stays "" since we can't know it.
		hostParts(m, raw)
		return m
	}
	scheme := strings.ToLower(sm[1])
	m["protocol"] = scheme
	rest := raw[len(sm[0]):]

	switch scheme {
	case "file":
		p := strings.TrimPrefix(rest, "//")
		if dec, err := url.PathUnescape(p); err == nil {
			p = dec
		}
		fileParts(m, p)
	case "http", "https":
		rest, ok := strings.CutPrefix(rest, "//")
		if !ok {
			return m
		}
		hostParts(m, rest)
	}
	return m
}

func fileParts(m map[string]string, p string) {
	m["protocol"] = "file"
	i := strings.LastIndexByte(p, '/')
	if i >= 0 {
		m["path"] = p[:i]
	}
	base := p[i+1:]
	m["FileName And Ext"] = base
	if d := strings.LastIndexByte(base, '.'); d > 0 {
		m["fileName"], m["FileExt"] = base[:d], base[d+1:]
	} else {
		m["fileName"] = base
	}
}

// hostParts fills in everything after the scheme: host[:port][/path][?query][#hash].
func hostParts(m map[string]string, rest string) {
	if i := strings.IndexByte(rest, '#'); i >= 0 {
		m["hash"] = rest[i:]
		rest = rest[:i]
	}
	if i := strings.IndexByte(rest, '?'); i >= 0 {
		m["perams"] = rest[i:]
		rest = rest[:i]
	}
	authority, path := rest, ""
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		authority, path = rest[:i], rest[i:]
	}
	m["path"] = path
	if i := strings.LastIndexByte(authority, '@'); i >= 0 {
		authority = authority[i+1:]
	}
	host := authority
	if h, p, err := net.SplitHostPort(authority); err == nil {
		host, m["port"] = h, p
	} else {
		host = strings.Trim(authority, "[]")
	}
	host = strings.ToLower(host)
	m["fulldomain"] = host
	if net.ParseIP(host) == nil {
		labels := strings.Split(host, ".")
		if n := len(labels); n >= 2 {
			m["tld"] = labels[n-1]
			m["maindomainonly"] = labels[n-2]
			m["maindomain"] = labels[n-2] + "." + labels[n-1]
			m["subdomain"] = strings.Join(labels[:n-2], ".")
		}
	}
}
