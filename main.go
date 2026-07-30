//**********************************************************************
//  main.go  - GoLisp REPL
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************

package main

import (
  "errors"
  "flag"
  "fmt"
  "io"
  "os"
  "strings"
  "syscall"
  "unsafe"

  "golisp2/lib"
  "golisp2/lib/swank"
)

func test(env *lib.Env, s string) {
  cell, _ := lib.Read(s)
  result, err := lib.Eval(cell, env)
  if err != nil {
    fmt.Printf("ERR: %v\n", err)
    return
  }
  fmt.Printf("%-45s => %s\n", s, result)
}

func runTests(env *lib.Env) {
  fmt.Println("=== GoLisp Test ===")
  fmt.Println()
  test(env, "(+ 1 2)")
  test(env, "(defun fak (n) (if (= n 0) 1 (* n (fak (- n 1)))))")
  test(env, "(fak 6)")
  test(env, "(mapcar (lambda (x) (* x x)) (list 1 2 3 4 5))")
  test(env, "(parfunc r (* 6 7) (+ 100 23))")
  test(env, "r")
  test(env, "(atom (gensym))")
  test(env, "(= (gensym) (gensym))")
  test(env, `(catch (error "oops") (lambda (e) (string-append "caught: " e)))`)
  test(env, `(catch (+ 1 2) (lambda (e) "fehler"))`)
  test(env, `(defun add-and-show (x y) (define s (+ x y)) s)`)
  test(env, `(add-and-show 3 4)`)
  test(env, `((lambda (x) (define d (* x 2)) d) 5)`)
  test(env, `(>= 5 3)`)
  test(env, `(>= 3 3)`)
  test(env, `(>= 2 3)`)
  test(env, `(<= 2 3)`)
  test(env, `(<= 3 3)`)
  test(env, `(<= 5 3)`)
  // while
  test(env, `(define w 0)`)
  test(env, `(while (< w 3) (set! w (+ w 1)))`)
  test(env, `w`)
  // do: (do ((i 0 (+ i 1)) (s 0 (+ s i))) ((= i 5) s))  → 0+1+2+3+4 = 10
  test(env, `(do ((i 0 (+ i 1)) (s 0 (+ s i))) ((= i 5) s))`)
  // TCO: tiefe Rekursion ohne Stack-Overflow
  test(env, `(defun sum-acc (n acc) (if (= n 0) acc (sum-acc (- n 1) (+ acc n))))`)
  test(env, `(sum-acc 1000000 0)`)
  test(env, `(defun even? (n) (if (= n 0) t (odd?  (- n 1))))`)
  test(env, `(defun odd?  (n) (if (= n 0) () (even? (- n 1))))`)
  test(env, `(even? 1000000)`)
  // equal?
  test(env, `(equal? 1 1)`)
  test(env, `(equal? 1 2)`)
  test(env, `(equal? "a" "a")`)
  test(env, `(equal? (list 1 2) (list 1 2))`)
  test(env, `(equal? (list 1 2) (list 1 3))`)
  test(env, `(equal? (list 1 (list 2 3)) (list 1 (list 2 3)))`)
  test(env, `(equal? () ())`)
  // &optional / &key Parameter
  test(env, `(defun greet (name &optional (greeting "Hallo")) (string-append greeting " " name))`)
  test(env, `(greet "Gerhard")`)
  test(env, `(greet "Gerhard" "Hi")`)
  test(env, `(defun f-key (x &key (y 0)) (+ x y))`)
  test(env, `(f-key 10)`)
  test(env, `(f-key 10 :y 5)`)
  // #' Reader-Makro und funcall
  test(env, `(mapcar #'car '((1 2) (3 4)))`)
  test(env, `(funcall #'+ 3 4)`)
  // flet
  test(env, `(flet ((sq (x) (* x x))) (sq 7))`)
  // labels: gegenseitig rekursive Funktionen
  test(env, `(labels ((even? (n) (if (= n 0) t (odd?  (- n 1)))) (odd? (n) (if (= n 0) () (even? (- n 1))))) (even? 10))`)
  // block / return-from
  test(env, `(block outer (return-from outer 42) 0)`)
  test(env, `(block b (+ 1 (return-from b 99)) 0)`)
  // Gemeinsame Test-Helfer (assert=) — eine Quelle für alle Testdateien
  test(env, `(load "tests/test-helpers.lisp")`)
  // Isolierte stdlib-Tests (defstruct, setf, defvar, union, set-difference, find-all)
  test(env, `(load "tests/stdlib-test.lisp")`)
  // defsystem-lite: Systemdefinition, topo-Laden, unload
  test(env, `(load "tests/defsystem-tests.lisp")`)
  // Norvigs drei bekannte GPS-Fehler (Timeout für rekursiven Fall)
  test(env, `(load "pn-gps1/gps-norvig-bugs.lisp")`)
  // GPS Version 2: state-passing, goroutine-tauglich
  test(env, `(load "pn-gps1/gps2-tests.lisp")`)
  // Genetischer Algorithmus
  test(env, `(define ga (ga-create 'bit1 5 4 (lambda (g) (apply + g))))`)
  test(env, `(ga-init ga)`)
  test(env, `(ga-calc ga)`)
  test(env, `(ga-result ga)`)
  // Test-Framework: alle registrierten Suiten laufen lassen.
  // Exit-Code des Prozesses = Anzahl FAILs (0 = grün), CI-tauglich.
  test(env, `(exit (run-tests))`)
}

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

func main() {
  // Define flags
  interactiveFlag := flag.Bool("i", false, "Interaktiver REPL-Modus")
  exprFlag := flag.String("e", "", "Expression direkt ausführen")
  testFlag := flag.Bool("t", false, "Tests ausführen")
  swankFlag := flag.String("swank", "", "SWANK-Server starten (Format: host:port, z.B. 127.0.0.1:4005)")

  flag.Parse()

  // Setup environment
  env := lib.BaseEnv()

  // Standardbibliothek laden (eingebettet, zentral in lib.LoadStdlib)
  if err := lib.LoadStdlib(env); err != nil {
    fmt.Fprintln(os.Stderr, "stdlib Fehler:", err)
    os.Exit(1)
  }

  // SWANK-Server Modus: golisp2 --swank [host:port]
  if *swankFlag != "" {
    addr := *swankFlag
    if !strings.Contains(addr, ":") {
      addr = "127.0.0.1:" + addr
    }
    if err := swank.RunServer(addr); err != nil {
      fmt.Fprintln(os.Stderr, "swank server error:", err)
      os.Exit(1)
    }
    os.Exit(0)
  }

  // Testmodus: golisp2 -t
  if *testFlag {
    runTests(env)
    os.Exit(0)
  }

  // Expression-Modus: golisp2 -e "(expr)"
  if *exprFlag != "" {
    exitCode := runExpression(*exprFlag, env, os.Stdout, os.Stderr)
    os.Exit(exitCode)
  }

  // Interaktiver Modus: golisp2 -i
  if *interactiveFlag {
    exitCode := runREPL(env)
    os.Exit(exitCode)
  }

  // Datei laden: golisp2 script.lisp
  if flag.NArg() > 0 {
    filename := flag.Arg(0)
    cell, err := lib.Read(`(load "` + filename + `")`)
    if err != nil {
      fmt.Fprintln(os.Stderr, "ERR:", err)
      os.Exit(1)
    }
    result, err := lib.Eval(cell, env)
    if err != nil {
      var le *lib.LispError
      if errors.As(err, &le) {
        fmt.Fprintln(os.Stderr, "ERR:", le.Msg)
      } else {
        fmt.Fprintln(os.Stderr, "ERR:", err)
      }
      os.Exit(1)
    }
    fmt.Println(result)
    os.Exit(0)
  }

  // Default: stdin lesen
  exitCode := runStdin(env)
  os.Exit(exitCode)
}
