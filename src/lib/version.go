//**********************************************************************
//  lib/version.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261018
//**********************************************************************
// Versionsnummer (von Hand, SemVer) plus Commit/Datum/dirty aus den
// VCS-Infos des Go-Builds. Einzige Quelle für CLI, REPL, SWANK, Client.
//**********************************************************************

package lib

import (
  "fmt"
  "runtime/debug"
  "strings"
)

// Version von golisp2 – von Hand gepflegt (SemVer)
const Version = "0.8.0"

// lisp-implementation-version: (lisp-implementation-version) → Version als String (CL)
func fnLispImplementationVersion(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("lisp-implementation-version: keine Argumente erwartet")
  }
  return MakeStr(Version), nil
}

// VersionText liefert "<name> <Version> (<commit>, <datum>[, dirty])";
// ohne VCS-Infos (z. B. go test) nur "<name> <Version>".
func VersionText(name string) string {
  info, ok := debug.ReadBuildInfo()
  if !ok {
    return name + " " + Version
  }
  return versionTextAus(name, info.Settings)
}

func versionTextAus(name string, settings []debug.BuildSetting) string {
  var rev, zeit string
  dirty := false
  for _, s := range settings {
    switch s.Key {
    case "vcs.revision":
      rev = s.Value
    case "vcs.time":
      zeit = s.Value
    case "vcs.modified":
      dirty = s.Value == "true"
    }
  }
  if rev == "" {
    return name + " " + Version
  }
  teile := []string{rev[:min(7, len(rev))]}
  if len(zeit) >= 10 {
    teile = append(teile, zeit[:10])
  }
  if dirty {
    teile = append(teile, "dirty")
  }
  return name + " " + Version + " (" + strings.Join(teile, ", ") + ")"
}
