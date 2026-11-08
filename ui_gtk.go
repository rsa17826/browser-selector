//go:build !nogui

package main

import (
	"fmt"
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const pickerCSS = `
window { background: #1b1b1b; color: #ddd; }
label { color: #ddd; }
entry { background: #333; color: #ddd; min-height: 0; padding: 0 2px; border-radius: 0; }
entry.good { background: #036800; }
entry.bad { background: #680000; }
dropdown button { background: #2a2a2a; color: #ddd; min-height: 0; padding: 0 2px; border-radius: 0; }
checkbutton { min-height: 0; padding: 0 2px; }
button.program { background: #333; color: #ddd; border: 3px solid #555; border-radius: 0; min-height: 0; padding: 0; margin: 0; }
button.program:hover { background: #444; }
.notice { background: #680000; padding: 2px 4px; }
`

type pickerRow struct {
	key   string
	orig  string // the url's real value for this key, what the pattern is tested against
	box   *gtk.Box
	check *gtk.CheckButton
	drop  *gtk.DropDown
	entry *gtk.Entry
}

func (r *pickerRow) recheck() {
	ok, err := doTheyMatch(r.orig, matchTypes[r.drop.Selected()], r.entry.Text())
	if ok {
		r.entry.AddCSSClass("good")
		r.entry.RemoveCSSClass("bad")
	} else {
		r.entry.AddCSSClass("bad")
		r.entry.RemoveCSSClass("good")
	}
	if err != nil {
		r.entry.SetTooltipText(err.Error())
	} else {
		r.entry.SetTooltipText("")
	}
}

// runUI shows the picker and blocks until it is closed. Nothing else runs
// while it is open, so choosing a program never has to kill anything: the
// window is a normal GTK window of this process.
func runUI(cfg *Config, url string, parent parentProc, notice string) (uiResult, error) {
	res := uiResult{closed: true}
	// NonUnique: a second picker must be its own window, not a message to the first.
	app := gtk.NewApplication(appID, gio.ApplicationNonUnique)
	app.ConnectActivate(func() {
		buildPicker(app, cfg, url, parent, notice, &res)
	})
	// Pass only argv[0]: the url and extra args are ours, not GTK's.
	if code := app.Run([]string{os.Args[0]}); code != 0 {
		return res, fmt.Errorf("gtk application exited with code %d", code)
	}
	dbg("picker closed: program=%q closed=%v", res.program, res.closed)
	return res, nil
}

func buildPicker(app *gtk.Application, cfg *Config, rawURL string, parent parentProc, notice string, res *uiResult) {
	provider := gtk.NewCSSProvider()
	provider.LoadFromData(pickerCSS)
	gtk.StyleContextAddProviderForDisplay(gdk.DisplayGetDefault(), provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

	win := gtk.NewApplicationWindow(app)
	win.SetTitle("Browser Selector")
	win.SetDefaultSize(640, 520)

	root := gtk.NewBox(gtk.OrientationVertical, 0)

	noticeLabel := gtk.NewLabel(notice)
	noticeLabel.AddCSSClass("notice")
	noticeLabel.SetWrap(true)
	noticeLabel.SetXAlign(0)
	noticeLabel.SetVisible(notice != "")
	root.Append(noticeLabel)

	urlEntry := gtk.NewEntry()
	urlEntry.SetText(rawURL)
	root.Append(urlEntry)

	m := parseURL(rawURL, parent)
	rows := make([]*pickerRow, 0, len(matchKeys))
	rowsBox := gtk.NewBox(gtk.OrientationVertical, 0)
	for _, key := range matchKeys {
		r := &pickerRow{key: key, orig: m[key]}
		r.box = gtk.NewBox(gtk.OrientationHorizontal, 0)
		r.check = gtk.NewCheckButton()
		label := gtk.NewLabel(key)
		label.SetXAlign(0)
		label.SetSizeRequest(150, -1)
		r.drop = gtk.NewDropDownFromStrings(matchTypes)
		r.entry = gtk.NewEntry()
		r.entry.SetHExpand(true)
		r.entry.SetText(r.orig)
		r.entry.ConnectChanged(r.recheck)
		r.drop.NotifyProperty("selected", r.recheck)
		r.recheck()
		r.box.Append(r.check)
		r.box.Append(label)
		r.box.Append(r.drop)
		r.box.Append(r.entry)
		rowsBox.Append(r.box)
		rows = append(rows, r)
	}
	// Empty rows are hidden when the url is loaded or changed, not while
	// typing, otherwise clearing a pattern would make its row vanish.
	applyHidden := func() {
		for _, r := range rows {
			r.box.SetVisible(!(cfg.Settings.HideEmptyProperties && r.orig == ""))
		}
	}
	applyHidden()
	root.Append(rowsBox)

	urlEntry.ConnectChanged(func() {
		m := parseURL(urlEntry.Text(), parent)
		for _, r := range rows {
			r.orig = m[r.key]
			r.entry.SetText(r.orig)
			r.recheck()
		}
		applyHidden()
	})

	choose := func(program string) {
		rule := Rule{Name: program}
		for _, r := range rows {
			if r.check.Active() {
				rule.Matches = append(rule.Matches, []string{r.key, matchTypes[r.drop.Selected()], r.entry.Text()})
			}
		}
		if len(rule.Matches) > 0 {
			if err := cfg.appendRule(rule); err != nil {
				noticeLabel.SetLabel(err.Error())
				noticeLabel.SetVisible(true)
				return
			}
		}
		*res = uiResult{program: program, url: urlEntry.Text()}
		win.Close()
	}
	for _, name := range append([]string{""}, cfg.visiblePrograms()...) {
		name := name
		text := name
		if text == "" {
			text = "BLOCK URL"
		}
		b := gtk.NewButtonWithLabel(text)
		b.AddCSSClass("program")
		b.ConnectClicked(func() { choose(name) })
		root.Append(b)
	}

	scroller := gtk.NewScrolledWindow()
	scroller.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroller.SetChild(root)
	win.SetChild(scroller)

	if cfg.Settings.CloseOnEscPressed {
		keys := gtk.NewEventControllerKey()
		keys.SetPropagationPhase(gtk.PhaseCapture) // before the focused entry sees it
		keys.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
			if keyval == gdk.KEY_Escape {
				win.Close()
				return true
			}
			return false
		})
		win.AddController(keys)
	}
	if cfg.Settings.CloseOnFocusLoss {
		wasActive := false
		win.NotifyProperty("is-active", func() {
			if win.IsActive() {
				wasActive = true
			} else if wasActive {
				win.Close()
			}
		})
	}

	win.Present()
}
