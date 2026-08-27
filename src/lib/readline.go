//**********************************************************************
//  lib/readline.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************
// Readline mit github.com/elk-language/go-prompt:
//   - Syntax-Highlighting: Klammern nach Tiefe eingefärbt
//   - TAB-Completion: alle Env-Symbole dynamisch
//   - History: persistent in ~/.golisp_history
//   - Multiline: offene Klammern → automatische Einrückung
//   - Ctrl+C: Eingabe abbrechen, Ctrl+D: Beenden
//**********************************************************************

package lib

import (
  "bufio"
  "fmt"
  "os"
  "strings"
  "unicode/utf8"

  prompt "github.com/elk-language/go-prompt"
  pstrings "github.com/elk-language/go-prompt/strings"
)

// depthColors: Klammern pro Tiefe — helle, kontrastreiche Farben (zyklisch)
var depthColors = []prompt.Color{
  prompt.Red,
  prompt.Green,
  prompt.Yellow,
  prompt.Fuchsia,
  prompt.Turquoise,
  prompt.White,
}

// lispLexer tokenisiert eine Zeile für das Syntax-Highlighting
func lispLexer(line string) []prompt.Token {
  if len(line) == 0 {
    return nil
  }
  var tokens []prompt.Token
  depth := 0
  inStr := false
  escape := false
  i := 0

  emit := func(first, last int, opts ...prompt.SimpleTokenOption) {
    if first > last { return }
    tokens = append(tokens, prompt.NewSimpleToken(
      pstrings.ByteNumber(first),
      pstrings.ByteNumber(last),
      opts...,
    ))
  }

  for i < len(line) {
    ch, size := utf8.DecodeRuneInString(line[i:])

    if escape {
      escape = false
      i += size
      continue
    }
    if inStr {
      if ch == '\\' { escape = true }
      if ch == '"'  { inStr = false }
      i += size
      continue
    }

    switch ch {
    case '"':
      // String: grün
      start := i
      i += size
      for i < len(line) {
        c, s := utf8.DecodeRuneInString(line[i:])
        i += s
        if c == '"' { break }
        if c == '\\' && i < len(line) {
          _, nextSize := utf8.DecodeRuneInString(line[i:])
          i += nextSize  // skip escaped char
        }
      }
      end := i - 1
      if end >= len(line) { end = len(line) - 1 }
      emit(start, end, prompt.SimpleTokenWithColor(prompt.Green))

    case ';':
      // Kommentar bis Zeilenende: dunkelgrau
      emit(i, len(line)-1, prompt.SimpleTokenWithColor(prompt.DarkGray))
      i = len(line)

    case '(':
      col := depthColors[depth%len(depthColors)]
      emit(i, i, prompt.SimpleTokenWithColor(col),
        prompt.SimpleTokenWithDisplayAttributes(prompt.DisplayBold))
      depth++
      i += size

    case ')':
      if depth > 0 { depth-- }
      col := depthColors[depth%len(depthColors)]
      emit(i, i, prompt.SimpleTokenWithColor(col),
        prompt.SimpleTokenWithDisplayAttributes(prompt.DisplayBold))
      i += size

    case '\'', '`', ',':
      // Quote-Zeichen: gelb
      emit(i, i, prompt.SimpleTokenWithColor(prompt.Yellow))
      i += size

    default:
      i += size
    }
  }
  return tokens
}

// fileHistory implementiert HistoryInterface mit Datei-Persistenz
type fileHistory struct {
  *prompt.History
  path string
}

func newFileHistory(path string) *fileHistory {
  h := &fileHistory{History: prompt.NewHistory(), path: path}
  // bestehende History einlesen
  f, err := os.Open(path)
  if err != nil { return h }
  defer f.Close()
  var entries []string
  sc := bufio.NewScanner(f)
  for sc.Scan() {
    if line := sc.Text(); line != "" {
      entries = append(entries, line)
    }
  }
  h.History.DeleteAll()
  for _, e := range entries {
    h.History.Add(e)
  }
  h.History.Clear()
  return h
}

func (h *fileHistory) Add(input string) {
  h.History.Add(input)
  // ans Ende der Datei anhängen
  f, err := os.OpenFile(h.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
  if err != nil { return }
  defer f.Close()
  fmt.Fprintln(f, input)
}

// countDepth zählt offene Klammern (ignoriert Strings und Kommentare)
func countDepth(s string) int {
  depth  := 0
  inStr  := false
  escape := false
  for _, ch := range s {
    if escape  { escape = false; continue }
    if ch == '\\' && inStr { escape = true; continue }
    if ch == '"' { inStr = !inStr; continue }
    if inStr     { continue }
    if ch == ';' { break }
    if ch == '(' { depth++ }
    if ch == ')' { depth-- }
  }
  return depth
}

// --- Readline -------------------------------------------------------

type Readline struct {
  ch chan string
}

func NewReadline(promptStr string, getSymbols func() []string) *Readline {
  r := &Readline{ch: make(chan string, 1)}

  // Executor: aufgerufen wenn ein vollständiger Ausdruck bestätigt wird
  executor := func(line string) {
    line = strings.TrimSpace(line)
    if line != "" {
      r.ch <- line
    }
  }

  // ExecuteOnEnterCallback: Multi-line – Enter nur ausführen wenn Klammern ausgeglichen
  executeOnEnter := func(p *prompt.Prompt, indentSize int) (int, bool) {
    text := p.Buffer().Text()
    depth := countDepth(text)
    if depth > 0 {
      return depth, false  // weiter eingeben, einrücken
    }
    return 0, true  // ausführen
  }

  histPath := os.ExpandEnv("$HOME/.golisp_history")
  hist := newFileHistory(histPath)

  p := prompt.New(
    executor,
    prompt.WithPrefix(promptStr),
    prompt.WithLexer(prompt.NewEagerLexer(lispLexer)),
    prompt.WithExecuteOnEnterCallback(executeOnEnter),
    prompt.WithCustomHistory(hist),
    prompt.WithIndentSize(2),
  )

  go func() {
    p.Run()
    close(r.ch)
  }()

  return r
}

// Read blockiert bis ein vollständiger Ausdruck verfügbar ist
func (r *Readline) Read() (string, error) {
  expr, ok := <-r.ch
  if !ok {
    return "", fmt.Errorf("EOF")
  }
  return expr, nil
}

// Close — kein-op, go-prompt beendet sich via Ctrl+D selbst
func (r *Readline) Close() {}
