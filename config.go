package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

var matchTypes = []string{"is", "matchesregex", "isexact", "startswith", "endswith", "includes"}

type Settings struct {
	DefaultBrowser      string `json:"defaultBrowser"`
	HideEmptyProperties bool   `json:"hideEmptyProperties"`
	CloseOnFocusLoss    bool   `json:"closeOnFocusLoss"`
	CloseOnEscPressed   bool   `json:"closeOnEscPressed"`
}

// Schema-declared defaults, same idea as `localStorage.option ??= default`:
// a missing optional setting takes its documented default.
func defaultSettings() Settings {
	return Settings{
		HideEmptyProperties: true,
		CloseOnFocusLoss:    true,
		CloseOnEscPressed:   true,
	}
}

type StringList []string

func (s *StringList) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*s = StringList{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("must be a string or an array of strings: %w", err)
	}
	*s = many
	return nil
}

type Program struct {
	Args     []string   `json:"args"`
	Path     StringList `json:"path"`
	Visible  bool       `json:"visible"`
	Fallback string     `json:"fallback"`
}

func (p *Program) UnmarshalJSON(b []byte) error {
	type plain Program
	v := plain{Visible: true}
	if err := strictDecode(b, &v); err != nil {
		return err
	}
	*p = Program(v)
	return nil
}

// A modifier rewrites the url before it is opened. It is run as
// path args... <url> and whatever it prints on stdout becomes the new url.
type Modifier struct {
	Path StringList `json:"path"`
	Args []string   `json:"args"`
}

type Rule struct {
	Matches   [][]string `json:"matches"`
	Name      string     `json:"name"`
	Modifiers []string   `json:"modifiers,omitempty"`
}

type Config struct {
	Path         string
	Settings     Settings
	Programs     map[string]Program
	Modifiers    map[string]Modifier
	ProgramOrder []string
	Rules        []Rule
}

func strictDecode(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func configPath() (string, error) {
	if p := os.Getenv("BROWSER_SELECTOR_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "browser-selector", "settings.json"), nil
}

const defaultConfigJSON = `{
  "settings": {
    "closeOnFocusLoss": false,
    "defaultBrowser": "",
    "closeOnEscPressed": true,
    "hideEmptyProperties": true
  },
  "programs": {
    "firefox": { "path": ["firefox"], "args": ["$url"] },
    "chromium": { "path": ["chromium"], "args": ["$url"] }
  },
  "rules": []
}
`

func loadConfig() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, []byte(defaultConfigJSON), 0o644); err != nil {
			return nil, err
		}
		b = []byte(defaultConfigJSON)
	} else if err != nil {
		return nil, err
	}
	return parseConfig(p, b)
}

func parseConfig(path string, b []byte) (*Config, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for k := range raw {
		if k != "$schema" && k != "settings" && k != "programs" && k != "rules" && k != "modifiers" {
			return nil, fmt.Errorf("%s: unknown top-level key %q", path, k)
		}
	}
	for _, k := range []string{"settings", "programs", "rules"} {
		if _, ok := raw[k]; !ok {
			return nil, fmt.Errorf("%s: missing required key %q", path, k)
		}
	}
	c := &Config{Path: path, Settings: defaultSettings()}
	if err := strictDecode(raw["settings"], &c.Settings); err != nil {
		return nil, fmt.Errorf("%s: settings: %w", path, err)
	}
	if err := strictDecode(raw["programs"], &c.Programs); err != nil {
		return nil, fmt.Errorf("%s: programs: %w", path, err)
	}
	if err := strictDecode(raw["rules"], &c.Rules); err != nil {
		return nil, fmt.Errorf("%s: rules: %w", path, err)
	}
	if mr, ok := raw["modifiers"]; ok {
		if err := strictDecode(mr, &c.Modifiers); err != nil {
			return nil, fmt.Errorf("%s: modifiers: %w", path, err)
		}
	}
	order, err := orderedKeys(raw["programs"])
	if err != nil {
		return nil, fmt.Errorf("%s: programs: %w", path, err)
	}
	c.ProgramOrder = order
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func orderedKeys(raw json.RawMessage) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	var keys []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, t.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func (c *Config) validate() error {
	for name, m := range c.Modifiers {
		if len(m.Path) == 0 {
			return fmt.Errorf("modifier %q has no path", name)
		}
		for _, mp := range m.Path {
			if mp == "" {
				return fmt.Errorf("modifier %q has an empty path entry", name)
			}
		}
	}
	for name, p := range c.Programs {
		if name == "" {
			return errors.New("program names must not be empty")
		}
		if len(p.Path) == 0 {
			return fmt.Errorf("program %q has no path", name)
		}
		for _, pp := range p.Path {
			if pp == "" {
				return fmt.Errorf("program %q has an empty path entry", name)
			}
		}
		if p.Fallback != "" {
			if _, ok := c.Programs[p.Fallback]; !ok {
				return fmt.Errorf("program %q: fallback %q is not a program", name, p.Fallback)
			}
		}
	}
	if d := c.Settings.DefaultBrowser; d != "" {
		if _, ok := c.Programs[d]; !ok {
			return fmt.Errorf("settings.defaultBrowser %q is not a program", d)
		}
	}
	for i, r := range c.Rules {
		if err := c.validateRule(r); err != nil {
			return fmt.Errorf("rule %d: %w", i+1, err)
		}
	}
	return nil
}

func (c *Config) validateRule(r Rule) error {
	if r.Name != "" {
		if _, ok := c.Programs[r.Name]; !ok {
			return fmt.Errorf("name %q is not a program", r.Name)
		}
	}
	if len(r.Modifiers) > 0 && r.Name == "" {
		return errors.New("a blocking rule (empty name) can't have modifiers")
	}
	for _, mn := range r.Modifiers {
		if _, ok := c.Modifiers[mn]; !ok {
			return fmt.Errorf("modifier %q is not defined in modifiers", mn)
		}
	}
	for _, m := range r.Matches {
		if len(m) != 3 {
			return fmt.Errorf("each match must be [key, matchtype, value], got %d items", len(m))
		}
		if !slices.Contains(matchKeys, m[0]) {
			return fmt.Errorf("unknown match key %q", m[0])
		}
		if !slices.Contains(matchTypes, m[1]) {
			return fmt.Errorf("unknown match type %q", m[1])
		}
		if m[1] == "matchesregex" {
			if _, err := regexp.Compile(m[2]); err != nil {
				return fmt.Errorf("bad regex %q: %w", m[2], err)
			}
		}
	}
	return nil
}

func (c *Config) findRule(m map[string]string) (*Rule, error) {
	for i := range c.Rules {
		r := &c.Rules[i]
		all := true
		for _, mt := range r.Matches {
			ok, err := doTheyMatch(m[mt[0]], mt[1], mt[2])
			if err != nil {
				return nil, fmt.Errorf("rule %d: %w", i+1, err)
			}
			if !ok {
				all = false
				break
			}
		}
		if all {
			return r, nil
		}
	}
	return nil, nil
}

// appendRule re-reads the file so manual edits made while the picker was
// open are kept, and keeps the order of the programs section intact.
func (c *Config) appendRule(r Rule) error {
	if err := c.validateRule(r); err != nil {
		return err
	}
	b, err := os.ReadFile(c.Path)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	var rules []json.RawMessage
	if err := json.Unmarshal(raw["rules"], &rules); err != nil {
		return err
	}
	nr, err := json.Marshal(r)
	if err != nil {
		return err
	}
	rules = append(rules, nr)
	if raw["rules"], err = json.Marshal(rules); err != nil {
		return err
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.Path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, c.Path); err != nil {
		return err
	}
	c.Rules = append(c.Rules, r)
	return nil
}

func (c *Config) visiblePrograms() []string {
	var out []string
	for _, n := range c.ProgramOrder {
		if c.Programs[n].Visible {
			out = append(out, n)
		}
	}
	return out
}
