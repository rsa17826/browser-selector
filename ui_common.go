package main

import (
	"fmt"
	"os"
)

// The Wayland app id, which Hyprland reports as the window class.
const appID = "org.browserselector.Picker"

type uiResult struct {
	program string
	url     string
	closed  bool // the window was dismissed without choosing anything
}

var debugOn = os.Getenv("BROWSER_SELECTOR_DEBUG") != ""

func dbg(format string, args ...any) {
	if debugOn {
		fmt.Fprintf(os.Stderr, "browser-selector[debug]: "+format+"\n", args...)
	}
}
