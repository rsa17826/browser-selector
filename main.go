package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "browser-selector:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	force := false
	var positional []string
	for _, a := range args {
		switch a {
		case "--force", "-force", "/force":
			force = true
		default:
			positional = append(positional, a)
		}
	}
	// First argument is the URL, anything after it is passed on as $browserArgs.
	rawURL := ""
	var extra []string
	if len(positional) > 0 {
		rawURL, extra = positional[0], positional[1:]
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	parent := findParentProc()
	m := parseURL(rawURL, parent)

	if force {
		return pick(cfg, rawURL, parent, extra, "")
	}
	if d := cfg.Settings.DefaultBrowser; d != "" {
		return launchOrPick(cfg, d, rawURL, parent, extra)
	}
	rule, err := cfg.findRule(m)
	if err != nil {
		return pick(cfg, rawURL, parent, extra, err.Error())
	}
	if rule != nil {
		newURL, err := applyModifiers(cfg, rule.Modifiers, rawURL)
		if err != nil {
			return pick(cfg, rawURL, parent, extra, err.Error())
		}
		return launchOrPick(cfg, rule.Name, newURL, parent, extra)
	}
	return pick(cfg, rawURL, parent, extra, "")
}

func launchOrPick(cfg *Config, name, rawURL string, parent parentProc, extra []string) error {
	err := launch(cfg, name, parseURL(rawURL, parent), extra)
	if errors.Is(err, errShowPicker) {
		return pick(cfg, rawURL, parent, extra, err.Error())
	}
	return err
}

func pick(cfg *Config, rawURL string, parent parentProc, extra []string, notice string) error {
	res, err := runUI(cfg, rawURL, parent, notice)
	if err != nil {
		return err
	}
	if res.closed {
		return nil
	}
	return launch(cfg, res.program, parseURL(res.url, parent), extra)
}
