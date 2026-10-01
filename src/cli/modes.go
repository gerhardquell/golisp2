//**********************************************************************
//  cli/modes.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************
// Ausführungsmodi: -e, stdin, REPL (verschoben aus main.go).
//**********************************************************************

package cli

import (
  "errors"
  "fmt"
  "io"
  "os"
  "strings"
  "syscall"
  "unsafe"

  "golisp2/src/lib"
)

// runExpression parses and executes expressions from -e. A single form's
// result is printed; for multiple forms only side effects are emitted so
// scripts like (exec ...) (println out) (println cd) produce clean output.
// Returns exit code 0 on success, 1 on error.
func runExpression(expr string, env *lib.Env, out io.Writer, errOut io.Writer) int {
  cells, err := lib.ReadAll(expr)
  if err != nil {
    fmt.Fprintf(errOut, "ERR read: %v\n", err)
    return 1
  }
  if cells == nil || cells.Type == lib.NIL {
    return 0
  }

  // Count forms so a single -e expression still prints its value.
  formCount := 0
  for c := cells; c != nil && c.Type == lib.LIST; c = c.Cdr {
    if c.Car != nil {
      formCount++
    }
  }

  var result *lib.Cell
  cur := cells
  for cur != nil && cur.Type == lib.LIST {
    form := cur.Car
    if form == nil {
      cur = cur.Cdr
      continue
    }
    result, err = lib.Eval(form, env)
    if err != nil {
      var le *lib.LispError
      if errors.As(err, &le) {
        fmt.Fprintf(errOut, "ERR: %s\n", le.Msg)
      } else {
        fmt.Fprintf(errOut, "ERR: %v\n", err)
      }
      return 1
    }
    cur = cur.Cdr
  }
  if formCount == 1 {
    fmt.Fprintln(out, result)
  }
  return 0
}

// countParens counts open parentheses in a string
func countParens(s string) int {
  count := 0
  inString := false
  escape := false
  for _, ch := range s {
    if escape {
      escape = false
      continue
    }
    if ch == '\\' && inString {
      escape = true
      continue
    }
    if ch == '"' && !escape {
      inString = !inString
      continue
    }
    if !inString {
      if ch == '(' {
        count++
      } else if ch == ')' {
        count--
      }
    }
  }
  return count
}

// runStdin reads from stdin, collects complete expressions, executes them
// Returns exit code 0 on success, 1 on error
func runStdin(env *lib.Env) int {
  var buffer strings.Builder
  hasError := false

  // Read all input
  data, err := io.ReadAll(os.Stdin)
  if err != nil {
    fmt.Fprintf(os.Stderr, "ERR: cannot read stdin: %v\n", err)
    return 1
  }

  input := string(data)
  lines := strings.Split(input, "\n")

  for _, line := range lines {
    buffer.WriteString(line)
    buffer.WriteString("\n")

    // Check if expression is complete
    if countParens(buffer.String()) <= 0 && strings.TrimSpace(buffer.String()) != "" {
      expr := strings.TrimSpace(buffer.String())
      if expr != "" {
        cells, err := lib.ReadAll(expr)
        if err != nil {
          fmt.Fprintf(os.Stderr, "ERR read: %v\n", err)
          hasError = true
          buffer.Reset()
          continue
        }
        for cur := cells; cur != nil && cur.Type == lib.LIST; cur = cur.Cdr {
          form := cur.Car
          if form == nil {
            continue
          }
          result, err := lib.Eval(form, env)
          if err != nil {
            var le *lib.LispError
            if errors.As(err, &le) {
              fmt.Fprintf(os.Stderr, "ERR: %s\n", le.Msg)
            } else {
              fmt.Fprintf(os.Stderr, "ERR: %v\n", err)
            }
            hasError = true
            continue
          }
          fmt.Println(result)
        }
      }
      buffer.Reset()
    }
  }

  // Handle any remaining expression
  if strings.TrimSpace(buffer.String()) != "" {
    fmt.Fprintf(os.Stderr, "ERR: unbalanced expression\n")
    hasError = true
  }

  if hasError {
    return 1
  }
  return 0
}

// isTerminal checks if file descriptor is a terminal using TCGETS ioctl
func isTerminal(fd int) bool {
  var termios syscall.Termios
  _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCGETS, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
  return err == 0
}

// runREPL starts the interactive REPL with go-prompt
func runREPL(env *lib.Env) int {
  // Check if stdin is a terminal
  if !isTerminal(int(os.Stdin.Fd())) {
    fmt.Fprintln(os.Stderr, "ERR: Interactive mode requires a terminal (TTY)")
    fmt.Fprintln(os.Stderr, "Hint: Use echo 'expr' | ./golisp2 for pipe mode, or ./golisp2 -e 'expr' for expressions")
    return 1
  }

  fmt.Println("GoLisp 0.2  –  Ctrl+D oder (exit) zum Beenden")
  fmt.Println("Multiline: offene Klammern → Fortsetzung mit ..")
  rl := lib.NewReadline("golisp2> ", env.Symbols)
  defer rl.Close()

  for {
    line, err := rl.Read()
    if err != nil {
      break
    }
    if line == "" {
      continue
    }
    if line == "(exit)" || line == "exit" {
      break
    }

    cell, err := lib.Read(line)
    if err != nil {
      fmt.Fprintln(os.Stderr, "ERR read:", err)
      continue
    }
    result, err := lib.Eval(cell, env)
    if err != nil {
      var le *lib.LispError
      if errors.As(err, &le) {
        fmt.Fprintln(os.Stderr, "ERR:", le.Msg)
      } else {
        fmt.Fprintln(os.Stderr, "ERR:", err)
      }
      continue
    }
    fmt.Println("=>", result)
  }
  fmt.Println("Tschüss!")
  return 0
}
