#Requires AutoHotkey v2.0
#SingleInstance Force

; must include
SetWorkingDir(A_ScriptDir)
#Include <admin>
#Include *i <AutoThemed>
; HKCu\Software\Microsoft\Windows\Shell\Associations\UrlAssociations\http
#Include <base> ; Array, Map, String
#Include <Misc> ; print, range, swap, ToString, RegExMatchAll, Highlight, MouseTip, WindowFromPoint, ConvertWinPos, WinGetInfo, GetCaretPos?, IntersectRect, File as F, JSON, WinHasClass, input, path, SendDll, rerange, globalMouseMove, joinObjs. makeSameLength - will add all at fromt and all at back currently is only both, fastDirCopy?, fastCopyFile?, callFuncWithOptionalArgs?

compile(script) {
  try DirCreate("./out")
  try FileDelete(A_ScriptDir '/out/' script '.exe')
  try FileDelete(A_ScriptDir '/' script '.exe')
  RunWait('"' path.join(A_AhkPath, '../../Compiler\Ahk2Exe.exe') '" /in "' A_ScriptDir '/' script '.ahk" /out "' A_ScriptDir '/out/' script '.exe" /base "' A_AhkPath '"')
}
compile("browser selector")
compile("setup")