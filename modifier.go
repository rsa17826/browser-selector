package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

const modifierTimeout = 5 * time.Second

// applyModifiers runs the named modifiers in order, feeding each one the
// previous one's output. Names were checked when the config was loaded.
func applyModifiers(c *Config, names []string, url string) (string, error) {
	for _, n := range names {
		out, err := runModifier(n, c.Modifiers[n], url)
		if err != nil {
			return "", err
		}
		dbg("modifier %q: %q -> %q", n, url, out)
		url = out
	}
	return url, nil
}

func runModifier(name string, mod Modifier, url string) (string, error) {
	var exe string
	for _, p := range mod.Path {
		if f, ok := findExecutable(p); ok {
			exe = f
			break
		}
	}
	if exe == "" {
		return "", fmt.Errorf("modifier %q: none of %v found", name, []string(mod.Path))
	}
	ctx, cancel := context.WithTimeout(context.Background(), modifierTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, append(slices.Clone(mod.Args), url)...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return "", fmt.Errorf("modifier %q timed out after %s", name, modifierTimeout)
	}
	if err != nil {
		return "", fmt.Errorf("modifier %q failed: %w", name, err)
	}
	res := strings.TrimSpace(string(out))
	if res == "" {
		return "", fmt.Errorf("modifier %q printed nothing for %q", name, url)
	}
	return res, nil
}
