;@Ahk2Exe-SetMainIcon browser selector icon.ico
#NoTrayIcon
#Include *i <AutoThemed>
#Include <Misc>
#Include <base>
; MsgBox(JSON.stringify(A_Args))
; "d:\programs\browser selector\browser selector.ahk" s "a^^a" a^^a

; add option to allow adding args to the started program
; add way to detect escaping to prevent having to 2x escape the args
DetectHiddenWindows(1)
print(A_Args)

settingsFilePath := AnyFile("./settings.json", "../settings.json", "./out/settings.json") || "./settings.json"
schemaFilePath := AnyFile("./settings.schema.jsonc", "../settings.schema.jsonc", "./out/settings.schema.jsonc") || "./settings.schema.jsonc"
settings := JSON.parse(f.read(settingsFilePath, "
(
{
  "$schema": "./settings.schema.jsonc",
  "settings":{
    "closeOnFocusLoss": false,
    "defaultBrowser": "",
    "closeOnEscPressed": true,
    "closeOnEscPressedAnywhere": false,
    "hideEmptyProperties": true,
    "titleBar": "shown",
    "alwaysOnTop": true
  },
  "programs": {
    "__default__": {
      "path": "__default__"
    },
    "selector": {
      "admin": false,
      "args": ["$url", "--force"],
      "path": "./browser selector.exe"
    }
  },
  "rules": [
  ]
}
)"), , 1)

usersettings := settings?.['settings'] ?? {}
browserArgs := []
for a in A_Args {
  if A_Index == 1
    continue
  if [
    '--force',
    '/force',
    '-force'
  ].includes(a)
    continue
  browserArgs.push(a)
}

; if browserArgs.Length
;   MsgBox(JSON.stringify(browserArgs))
win := NeutronWindow().Load("./picker.html")

getSetting(obj, key, default) {
  try return obj.%key%
  try return obj[key]
  return default
}
if ((!A_IsCompiled) || (!getSetting(usersettings, "closeOnFocusLoss", 1))) {
  win.wnd.onblur := ''
}

if getSetting(usersettings, "closeOnEscPressedAnywhere", 0) or getSetting(usersettings, "closeOnEscPressed", 1) {
  Hotkey("~esc", (*) {
    if WinActive("ahk_id " win.hwnd) or getSetting(usersettings, "closeOnEscPressedAnywhere", 0)
      ExitApp()
  })
}
if getSetting(usersettings, "titleBar", 0)
  parentpath := ''
#Include <Neutron>
#Include <OnMouseEvent>
#SingleInstance off
SELFTITLE := A_IsCompiled ? A_ScriptFullPath : A_ScriptFullPath " - AutoHotkey v" A_AhkVersion
try parentpath := ProcessGetPath(ProcessGetParent(WinExist(SELFTITLE)))

; MsgBox(ProcessGetPath(WinGetProcessName(WinExist(SELFTITLE))))
; url := A_Args.has(1) ? A_Args[1] : "https://bloxd.asdasd/"
url := A_Args.has(1) ? A_Args[1] : "###TESTING###"

newprogram("")
for name, data in settings['programs'] {
  if getSetting(data, "visible", 1)
    newprogram(name)
}
newprogram(name, img := '') {
  win.wnd.addprogram(name, img)
}

match := parseurl(url)

AhkUpdateShownUrl(newurl) {
  global url, match
  url := newurl
  match := parseurl(url)
  win.wnd.updateoptions(url, JSON.stringify(match))
}

match.parentprocesspath := parentpath
print(match)
if url {
  data := '`n' url
  for k, thing2 in match.OwnProps() {
    data .= '`n' k " = " thing2
  }
  t := f.read(schemaFilePath)
  t := t.RegExReplace(" .*`"", " " data.Replace("\", "\\").replace('"', "\`"").Replace("`n", "\n") '"')
  f.write(, (t))
}
; t.properties.rules.items.properties.matches.items.items.description := " " data

; if (!parentpath) {
;   showBrowserSelectorWindow()
;   return
;   ; MsgBox("RECURSIVE OPEN DETECTED")
;   ; ExitApp(-1)
; }

joinargs(args) {
  a := ""
  for arg in args {
    a .= ' "' . StrReplace(arg, '"', '\"') . '"'
  }
  return a.RegExReplace("^ ", "")
}

if (A_Args.includes("--force") or A_Args.includes("-force") or A_Args.includes("/force")) {
  showBrowserSelectorWindow()
  return
}
if usersettings.has("defaultBrowser") {
  if usersettings['defaultBrowser'] {
    if settings['programs'].has(usersettings['defaultBrowser']) {
      runprogram(usersettings['defaultBrowser'])
      return
    }
  }
}
for rule in settings['rules'] {
mainloop:
  loop 1 {
    for m in rule['matches'] {
      k := m[1]
      matchtype := m[2]
      thing2 := m[3]
      if !match.HasProp(k)
        return
      if !DoTheyMatch(match.%k%, matchtype, thing2)
        break mainloop
    }
    print("rule found", rule['matches'], match, "opening with", rule['name'] || "URL BLOCKED")
    if !rule['name'] {
      return
    }
    runprogram(rule['name'])
    return
  }
}
DoTheyMatch(thing1, matchtype, thing2) {
  try {
    switch matchtype, 0 {
      case "is":
        tempformat(a) {
          return a.replace("\\", "\").replace("\", "/").trim(' /')
        }
        if tempformat(thing1) != tempformat(thing2)
          return 0
      case "isexact":
        if thing1 != thing2
          return 0
      case "startswith":
        if !thing1.startsWith(thing2)
          return 0
      case "endswith":
        if !thing1.endswith(thing2)
          return 0
      case "matchesregex":
        if !thing1.RegExMatch(thing2)
          return 0
      case "includes":
        if !thing1.includes(thing2)
          return 0
      default:
        print("invalid matchtype", matchtype)
        return 0
    }
  } catch {
    return 0
  }
  return 1
}
; try WinKill("ahk_exe IEChooser.exe")
; run("C:\Windows\System32\F12\IEChooser.exe")
; run("C:\Windows\SysWOW64\F12\IEChooser.exe")
; WinWait("ahk_exe IEChooser.exe")
; WinSetAlwaysOnTop(1, ("ahk_exe IEChooser.exe"))
runprogram(progName) {
  if !progName {
    print("rule has no name, ignoring url")
    return
  }
  programs := [
    progName
  ]
  getfallback(progName) {
    program := settings['programs'][progName]
    if program.Has("fallback") {
      if programs.includes(program['fallback'])
        return
      programs.push(program['fallback'])
      getfallback(program['fallback'])
    }
  }
  getfallback(progName)
  loop programs.Length {
    tempProgName := programs[A_Index]
    if tempProgName == "__default__" {
      try run(match.url)
      return
    }
    program := settings['programs'][tempProgName]
    if Type(program['path']) != "array"
      program['path'] := [
        program['path']
      ]
    FoundProgramPath := ''
    for _path in program['path'] {
      if _path == "__default__" {
        try run(match.url)
        return
      }

      FoundProgramPath := _path
      if FileExist(FoundProgramPath) {
        break
      } else {
        try
          Print("program ", tempProgName, "not found at ", FoundProgramPath, "using fallback", program['path'][A_Index + 1])
        catch {
          Print("program ", tempProgName, "not found at ", FoundProgramPath, "showing program selector")
          showBrowserSelectorWindow()
        }
        continue
      }
    }
    if !FileExist(FoundProgramPath)
      continue
    if path.info(FoundProgramPath).absPath == path.format(A_ScriptFullPath) {
      showBrowserSelectorWindow()
      return
    }
    temp := []
    for arg in program['args'] {
      arg := arg.RegExReplace("\$(\w+)", (reg) {
        ret := reg[1]
        try ret := match.%reg[1]%
        return ret
      })
      temp.Push(arg)
    }
    cmd := joinargs([
      FoundProgramPath,
      temp*
    ])
    print(cmd)

    if program['admin']
      Run('*RunAs ' cmd)
    else
      Run(cmd)
    ExitApp()
  }
}
;
selectedprogram(e, url, savedata) {
  match.url := url
  global win
  win.hide()
  savedata := JSON.parse(savedata, , 0)
  Print(savedata)
  realsavedata := []
  for thing in savedata {
    if !thing.checked
      continue
    realsavedata.push(thing)
  }
  program := e.target.value
  if realsavedata.Length {
    s := JSON.parse(f.read(settingsFilePath), 1)
    s['rules'].Push({
      name: program,
      matches:
        realsavedata.Map(e => [
          e.id,
          e.matchtype,
          e.value,
        ])
    })
    f.write(, JSON.stringify(s))
  }
  runprogram(program)
}

;

if 0 {
  FileInstall("picker.html", "*")
  FileInstall("browser selector icon.ico", "*")
}

parseurl(url) {
  tempurl := url
  tempParsed := {}
  if tempurl.RegExMatch("^(\w):") {
    set("drive", "(\w):", 1)
    tempParsed.protocol := "file"
    set("path", "(.*)[\\/]", 1)
    temp := tempurl.RegExMatch("(.+)\.([^.]+)$")
    tempParsed.FileName := temp[1]
    tempParsed.FileExt := temp[2]
    set("fileName And Ext", ".*")
    return end()
  }
  if set("protocol", "(https?)://", 1) {
    if set("ip", "(?:(?:[1-2][0-9]{2}|[0-9]|[1-9][0-9])\.){3}(?:[1-2][0-9]{2}|[0-9]|[1-9][0-9])\b") {
    } else {
      set("fulldomain", "[^:/]+(?=:|/|$)")
    }
    try {
      port := tempurl.RegExMatch("^:(\d{1,5})(\b|$)")[1]
      if port and port <= 65535 {
        set("port", ":(" port ")", 1)
      }
    }
    set("path", "/[^ #?]+")
    replace("^/", "")
    set("perams", "\?[^#]+")
    set("hash", "#.*$")
    return end()
  }
  if set("protocol", "(\w+):", 1) {
    replace(":(//)?")
    set("data", ".*")
    return end()
  }
  end() {
    if tempurl
      tempParsed.leftovers := tempurl
    parsed := {
      url: url,
      protocol: tempParsed.protocol ?? '',
      fulldomain: tempParsed.fulldomain ?? '',
      tld: tempParsed?.fulldomain?.RegExMatch("\.(\w+)$")[1] ?? '',
      subdomain: tempParsed?.fulldomain?.RegExMatch("(.*)(?:\.\w+){2}$")[1] ?? '',
      maindomainonly: tempParsed?.fulldomain?.RegExMatch("(\w+)\.\w+$")[1] ?? '',
      maindomain: tempParsed?.fulldomain?.RegExMatch("(\w+\.\w+)$")[1] ?? '',
      path: tempParsed.path ?? '',
      perams: tempParsed.perams ?? '',
      port: tempParsed.port ?? '',
      hash: tempParsed.hash ?? '',
      fileName: tempParsed.fileName ?? '',
      drive: tempParsed.drive ?? '',
      %"FileName And Ext"%: tempParsed.%"FileName And Ext"% ?? '',
      FileExt: tempParsed.FileExt ?? '',
      browserArgs: browserArgs.map(e => "`"" e.replace("`"", "^`"") "`"").join(' '),
    }
    ; A_Clipboard := JSON.stringify(parsed)
    ; A_Clipboard := JSON.stringify(parsed.Keys())
    return parsed
  }
  replace(reg, with := '') {
    try {
      match := tempurl.RegExMatch('^' reg)[0]
      if !match
        return 0
      tempurl := tempurl.RegExReplace('^' reg, with)
      return 1
    }
    return 0
  }
  set(val, reg, regval := 0) {
    try {
      match := tempurl.RegExMatch('^' reg)
      if !match[0]
        return 0
      tempParsed.%val% := match[regval]
      tempurl := tempurl.RegExReplace('^' reg, '')
      return 1
    }
    return 0
  }
  return end()
}
showBrowserSelectorWindow() {
  win.wnd.addoptions(url, JSON.stringify(match), JSON.stringify(usersettings))
  win.Show()
  if getSetting(usersettings, "alwaysOnTop", 1) {
    WinSetAlwaysOnTop(1, win.hWnd)
  }
}

AhkStopAlt() {
  Send("{Esc}")
}
showBrowserSelectorWindow()