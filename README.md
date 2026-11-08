# browser-selector

Opens URLs with a browser chosen by rules, or shows a picker when no rule matches.
Port of the AHK browser selector for NixOS + Hyprland. Go standard library only.

## Install (NixOS)

```nix
environment.systemPackages = [ (pkgs.callPackage ./browser-selector { }) ];
```

Make it the default handler:

```sh
xdg-settings set default-web-browser browser-selector.desktop
```

## Config

`~/.config/browser-selector/settings.json` (created on first run; override with
`BROWSER_SELECTOR_CONFIG`). Same format as before, see `settings.schema.jsonc`.
Unknown keys and bad regexes are errors, not ignored. If a rule can't be evaluated
or a program can't be found, the picker opens with the error shown at the top.

The picker is a native GTK4 window (gotk4) with the Wayland app id
`org.browserselector.Picker`. Set `BROWSER_SELECTOR_DEBUG=1` to log what the
program is doing. `uiCommand` was removed with the old web picker; delete it from
your `settings.json` (unknown settings are an error).

## Modifiers

A modifier rewrites the url before it is opened. Define them under `modifiers` and
list them on a rule:

```json
"modifiers": {
  "noLinkFilter": {
    "path": ["python3", "python"],
    "args": ["-c", "import urllib.parse,sys; print(urllib.parse.unquote(sys.argv[1].split('?u=')[-1]))"]
  }
},
"rules": [
  { "name": "firefox", "matches": [["parentprocesspath", "is", "/path/to/steam"]], "modifiers": ["noLinkFilter"] }
]
```

The modifier runs as `path args... <url>` (the url is always the last argument,
`$variables` are not expanded), and its stdout, trimmed, is the new url. Modifiers
run in the listed order, each getting the previous one's output. A modifier that
isn't found, exits non-zero, takes over 5s or prints nothing is an error: the
picker opens with the message. Modifiers only run for rules, and a rule saved from
the picker has none, so add them to `settings.json` by hand.

## Match keys for the parent process

`parentprocesspath` is the executable that opened the url and `parentprocesscmdline`
is its full command line. Arguments are joined with single spaces, so quoting is lost
and an argument containing a space can't be told apart from two. Both skip shells and
`xdg-open` in between, and are empty if the parent can't be read.

## Hyprland window rules

The window class is the app id, set before the window maps, so a rule on it is
reliable:

```lua
hl.window_rule({
  match = { class = [[^org\.browserselector\.Picker$]] },
  float = true, pin = true, center = true,
})
```

## Building

```sh
go get github.com/diamondburned/gotk4/pkg@latest && go mod tidy
go run . https://example.com        # needs GTK4 dev files (nix develop / nix-shell -p gtk4 pkg-config)
```

For Nix, set `vendorHash` in `default.nix` to `lib.fakeHash`, run `nix build`,
and paste the hash it prints. Tests that don't need GTK: `go test -tags nogui ./...`

## Usage

```
browser-selector <url> [extra args...]   # extra args go to $browserArgs
browser-selector <url> --force           # always show the picker
```

## Differences from the AHK version

- No `admin` option, `__default__` program, `titleBar`, `alwaysOnTop` or
  `closeOnEscPressedAnywhere` (window behaviour is the compositor's job).
- `matchesregex` uses Go RE2 syntax. `is` is case-insensitive, the others are case-sensitive.
- `parentprocesspath` skips shells and xdg-open, which are always the direct parent on Linux.
- An unknown `$variable` in `args` is an error instead of turning into plain text.
