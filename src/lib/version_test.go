//**********************************************************************
//  lib/version_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261018
//**********************************************************************
// Tests für VersionText / versionTextAus.
//**********************************************************************

package lib

import (
  "regexp"
  "runtime/debug"
  "strings"
  "testing"
)

func TestVersionTextOhneVCS(t *testing.T) {
  if got := versionTextAus("golisp2", nil); got != "golisp2 "+Version {
    t.Fatalf("got %q", got)
  }
  if !strings.HasPrefix(VersionText("golisp2"), "golisp2 "+Version) {
    t.Fatalf("VersionText = %q", VersionText("golisp2"))
  }
}

func TestVersionTextMitVCS(t *testing.T) {
  re := regexp.MustCompile(`^golisp2-client \d+\.\d+\.\d+ \([0-9a-f]{7}, \d{4}-\d{2}-\d{2}(, dirty)?\)$`)
  settings := []debug.BuildSetting{
    {Key: "vcs.revision", Value: "46a91201234567890abcdef"},
    {Key: "vcs.time", Value: "2026-10-18T12:00:00Z"},
    {Key: "vcs.modified", Value: "false"},
  }
  got := versionTextAus("golisp2-client", settings)
  if !re.MatchString(got) || strings.Contains(got, "dirty") {
    t.Fatalf("sauber: %q", got)
  }
  settings[2].Value = "true"
  got = versionTextAus("golisp2-client", settings)
  if !re.MatchString(got) || !strings.HasSuffix(got, ", dirty)") {
    t.Fatalf("dirty: %q", got)
  }
}
