//go:build nogui

package main

import "errors"

// Build with -tags nogui to compile and test everything except the GTK window
// on a machine without GTK4 (used for the unit tests in CI-less sandboxes).
func runUI(cfg *Config, url string, parent parentProc, notice string) (uiResult, error) {
	return uiResult{}, errors.New("built with -tags nogui: no picker window")
}
