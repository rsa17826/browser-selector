package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// errShowPicker means "this launch can't happen, ask the user instead".
var errShowPicker = errors.New("show picker")

var varRe = regexp.MustCompile(`\$(\w+)`)

// expandArgs substitutes $name with the matching URL property. An argument that
// is exactly "$browserArgs" expands to the extra command-line arguments as
// separate argv entries. Unknown variables are an error.
func expandArgs(args []string, m map[string]string, extra []string) ([]string, error) {
	var out []string
	for _, a := range args {
		if a == "$browserArgs" {
			out = append(out, extra...)
			continue
		}
		var firstErr error
		res := varRe.ReplaceAllStringFunc(a, func(s string) string {
			name := s[1:]
			if name == "browserArgs" {
				return strings.Join(extra, " ")
			}
			v, ok := m[name]
			if !ok {
				if firstErr == nil {
					firstErr = fmt.Errorf("unknown variable $%s in args %q", name, a)
				}
				return s
			}
			return v
		})
		if firstErr != nil {
			return nil, firstErr
		}
		out = append(out, res)
	}
	return out, nil
}

func fallbackChain(c *Config, name string) []string {
	chain := []string{name}
	cur := name
	for {
		fb := c.Programs[cur].Fallback
		if fb == "" {
			return chain
		}
		for _, n := range chain {
			if n == fb {
				return chain
			}
		}
		chain = append(chain, fb)
		cur = fb
	}
}

func findExecutable(p string) (string, bool) {
	p = os.ExpandEnv(p)
	if strings.Contains(p, "/") {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
			return "", false
		}
		return p, true
	}
	r, err := exec.LookPath(p)
	return r, err == nil
}

func sameFile(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	return err1 == nil && err2 == nil && ra == rb
}

// launch replaces this process with the chosen program. It only returns on
// failure; errShowPicker is wrapped for "program missing" / "program is me".
func launch(c *Config, name string, m map[string]string, extra []string) error {
	if name == "" {
		fmt.Fprintln(os.Stderr, "browser-selector: url blocked")
		return nil
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	var tried []string
	for _, pn := range fallbackChain(c, name) {
		prog := c.Programs[pn]
		var found string
		for _, p := range prog.Path {
			if f, ok := findExecutable(p); ok {
				found = f
				break
			}
			tried = append(tried, p)
		}
		if found == "" {
			fmt.Fprintf(os.Stderr, "browser-selector: program %q not found (tried %v)\n", pn, prog.Path)
			continue
		}
		if sameFile(found, self) {
			return fmt.Errorf("program %q points at the selector itself: %w", pn, errShowPicker)
		}
		args, err := expandArgs(prog.Args, m, extra)
		if err != nil {
			return fmt.Errorf("program %q: %w", pn, err)
		}
		dbg("exec %s %q", found, args)
		return syscall.Exec(found, append([]string{found}, args...), os.Environ())
	}
	return fmt.Errorf("no usable program for %q (tried %v): %w", name, tried, errShowPicker)
}

var wrapperParents = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true, "fish": true,
	"xdg-open": true, "env": true, "gio-launch-desktop": true,
}

func parentPID(pid int) (int, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	i := bytes.LastIndexByte(b, ')')
	if i < 0 {
		return 0, fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	f := strings.Fields(string(b[i+1:]))
	if len(f) < 2 {
		return 0, fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	var pp int
	if _, err := fmt.Sscanf(f[1], "%d", &pp); err != nil {
		return 0, err
	}
	return pp, nil
}

// parentProc describes the process that asked for the URL.
type parentProc struct {
	path    string // executable path
	cmdline string // its arguments joined by single spaces (quoting is lost)
}

func processCmdline(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		fmt.Fprintf(os.Stderr, "browser-selector: cannot read command line of process %d: %v\n", pid, err)
		return ""
	}
	return strings.Join(strings.Split(strings.TrimRight(string(b), "\x00"), "\x00"), " ")
}

// findParentProc is the process that asked for the URL. xdg-open and shells in
// between are skipped, since they're always the immediate parent on Linux and
// would make the property useless for rules.
func findParentProc() parentProc {
	pid := os.Getppid()
	for depth := 0; depth < 8 && pid > 1; depth++ {
		exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if err != nil {
			fmt.Fprintf(os.Stderr, "browser-selector: cannot read parent process %d: %v\n", pid, err)
			return parentProc{}
		}
		if !wrapperParents[filepath.Base(exe)] {
			return parentProc{path: exe, cmdline: processCmdline(pid)}
		}
		pid, err = parentPID(pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "browser-selector: %v\n", err)
			return parentProc{path: exe}
		}
	}
	return parentProc{}
}
