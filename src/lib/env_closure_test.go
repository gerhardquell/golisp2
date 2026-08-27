//**********************************************************************
//  lib/env_closure_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260805
//**********************************************************************
// Closure x Frame-Form: das Kreuzprodukt, das die bestehenden Suiten
// nicht abdeckten.
//
// Historie: Frame-Envs wurden in einem sync.Pool recycelt, sobald kein
// Env.shared-Bit gesetzt war. shared markierte aber nur den DIREKT
// gefangenen Frame, während die Erreichbarkeit einer Closure transitiv
// über parent läuft. Fing eine Closure einen NACHKOMMEN-Frame, wurde der
// Zwischen-Frame recycelt und die Closure zeigte über parent in einen
// fremden Frame — verlorene Bindung oder Endlos-Zyklus in Env.Get.
// Zehn von elf frame-besitzenden Formen waren betroffen.
//
// Das Pooling ist entfernt; die Frame-Lebensdauer gehört dem Go-GC. Diese
// Tests bewachen seitdem die Semantik: eine Closure behält ihren
// lexikalischen Scope, egal was zwischen Bau und Aufruf passiert.
//
// Sollwerte sind SBCL-Ergebnisse, hart eingetragen: go test darf nicht
// von einer installierten SBCL abhängen. Der SBCL-Ausdruck steht je Fall
// im Kommentar, damit der Sollwert nachprüfbar bleibt.
//
// Siehe TODO.md Abschnitt 1 + 5.
//**********************************************************************

package lib

import (
  "os"
  "os/exec"
  "path/filepath"
  "strings"
  "testing"
)

// churnSrc definiert eine Funktion, die k frische Frames erzeugt und
// wieder verlässt. Zur Pooling-Zeit war das nötig, damit der Pool den
// fälschlich freigegebenen Frame an einen fremden Frame auslieferte —
// ohne Churn schlug der Fehler als Endlos-Zyklus in Env.Get zu (siehe
// TestClosureScopeNoStackOverflow). Bleibt als Stressor stehen: er kostet
// wenig und hält den Regressionsfall reproduzierbar, falls jemand das
// Pooling je wieder einführt.
const churnSrc = `(defun churn (k) (if (> k 0) (let ((a k) (b k)) (churn (- k 1))) 'ok))`

// identSrc: Identität als LAMBDA. Nötig für die let/let*-Fälle: dort wurde
// der Frame nicht per defer freigegeben, sondern beim Tail-Übergang zu
// einem Frame mit anderem parent. Der Übergang braucht einen echten
// Lambda-Aufruf in Tail-Position. Ohne ihn bleibt der äußere let-Frame
// unangetastet und der Fall war grün, ohne etwas zu beweisen — genau das
// hat die RED-Verifikation aufgedeckt. Nicht entfernen.
const identSrc = `(defun ident (x) x)`

// churnCount: wie viele Frames zwischen Closure-Bau und Closure-Aufruf
// durch den Pool laufen. 300 ist empirisch bestimmt und großzügig — der
// Fehler tritt schon bei wenigen auf.
const churnCount = "300"

// evalClosureCase baut die Closure, lässt churnCount Frames durch den
// Pool laufen und ruft die Closure dann auf. Alles in einem Env, damit
// der Pool-Zustand zwischen Bau und Aufruf derselbe ist.
func evalClosureCase(t *testing.T, expr string) (*Cell, error) {
  t.Helper()
  src := churnSrc + "\n" + identSrc + "\n" +
    "(define c " + expr + ")\n" +
    "(churn " + churnCount + ")\n" +
    "(funcall c)"
  return evalStr(src)
}

// closureCase: eine Frame-Form, der Ausdruck der die Closure liefert,
// und das SBCL-Ergebnis.
type closureCase struct {
  name string
  expr string
  want string
}

// brokenCases: Formen, die einen eigenen Frame anlegen und ihn freigeben,
// ohne ihn als shared zu markieren. Alle zehn müssen VOR dem Fix
// fehlschlagen — sonst testet der Fall nicht, was er zu testen behauptet.
var brokenCases = []closureCase{
  {
    // (ident ...) muss stehen bleiben: der Tail-Call ist der Auslöser,
    // nicht das verschachtelte let. Siehe identSrc.
    // SBCL: (flet ((ident (x) x))
    //         (funcall (let ((n 5)) (ident (let ((m 6)) (lambda () (list m n))))))) => (6 5)
    name: "let",
    expr: "(let ((n 5)) (ident (let ((m 6)) (lambda () (list m n)))))",
    want: "(6 5)",
  },
  {
    // SBCL: analog mit let* => (6 5)
    name: "let_star",
    expr: "(let* ((n 5)) (ident (let* ((m 6)) (lambda () (list m n)))))",
    want: "(6 5)",
  },
  {
    // SBCL: (funcall (funcall (lambda (n) (let ((m 6)) (lambda () (list m n)))) 5)) => (6 5)
    name: "applyLambda_funcall",
    expr: "(funcall (lambda (n) (let ((m 6)) (lambda () (list m n)))) 5)",
    want: "(6 5)",
  },
  {
    // SBCL: (funcall (flet ((f () 1)) (let ((n 5)) (lambda () (list n (f)))))) => (5 1)
    name: "flet",
    expr: "(flet ((f () 1)) (let ((n 5)) (lambda () (list n (f)))))",
    want: "(5 1)",
  },
  {
    // SBCL: (funcall (do ((i 0 (+ i 1))) ((> i 0) (let ((n 5)) (lambda () (list n i)))))) => (5 1)
    name: "do",
    expr: "(do ((i 0 (+ i 1))) ((> i 0) (let ((n 5)) (lambda () (list n i)))))",
    want: "(5 1)",
  },
  {
    // SBCL: (funcall (do* ((i 0 (+ i 1))) ((> i 0) (let ((n 5)) (lambda () (list n i)))))) => (5 1)
    name: "do_star",
    expr: "(do* ((i 0 (+ i 1))) ((> i 0) (let ((n 5)) (lambda () (list n i)))))",
    want: "(5 1)",
  },
  {
    // SBCL: (funcall (multiple-value-bind (a b) (values 1 2) (let ((n 5)) (lambda () (list n a))))) => (5 1)
    name: "multiple_value_bind",
    expr: "(multiple-value-bind (a b) (values 1 2) (let ((n 5)) (lambda () (list n a))))",
    want: "(5 1)",
  },
  {
    // SBCL: (funcall (macrolet ((m () 1)) (let ((n 5)) (lambda () (list n (m)))))) => (5 1)
    name: "macrolet",
    expr: "(macrolet ((m () 1)) (let ((n 5)) (lambda () (list n (m)))))",
    want: "(5 1)",
  },
  {
    // SBCL: (funcall (symbol-macrolet ((s 1)) (let ((n 5)) (lambda () (list n s))))) => (5 1)
    name: "symbol_macrolet",
    expr: "(symbol-macrolet ((s 1)) (let ((n 5)) (lambda () (list n s))))",
    want: "(5 1)",
  },
  {
    // CL-Äquivalent mit special declaration; hier golisp2-progv.
    name: "progv",
    expr: "(progv '(*x*) '(1) (let ((n 5)) (lambda () (list n *x*))))",
    want: "(5 1)",
  },
}

// safeCases: Formen, die entweder keinen eigenen Frame anlegen oder ihn
// korrekt markieren. Charakterisierungstests — sie halten das IST fest
// und müssen vor UND nach dem Fix grün sein.
var safeCases = []closureCase{
  {
    // labels übergibt makeLambda den localEnv (CL: labels-Funktionen
    // sehen sich gegenseitig) und markiert ihn dadurch als shared.
    name: "labels_mit_defs",
    expr: "(labels ((f () 1)) (let ((n 5)) (lambda () (list n (f)))))",
    want: "(5 1)",
  },
  {
    // Ohne Definitionen ist der labels-Frame leer — nichts zu verlieren.
    name: "labels_ohne_defs",
    expr: "(labels () (let ((n 5)) (lambda () (list n))))",
    want: "(5)",
  },
  {
    name: "block",
    expr: "(block b (let ((n 5)) (lambda () n)))",
    want: "5",
  },
  {
    name: "catch",
    expr: "(catch 'c (let ((n 5)) (lambda () n)))",
    want: "5",
  },
  {
    name: "unwind_protect",
    expr: "(unwind-protect (let ((n 5)) (lambda () n)) 1)",
    want: "5",
  },
  {
    name: "tagbody",
    expr: "(let ((r ())) (tagbody (let ((n 5)) (setq r (lambda () n)))) r)",
    want: "5",
  },
  {
    name: "while",
    expr: "(let ((r ()) (i 0)) (while (< i 1) (setq i 1) (let ((n 5)) (setq r (lambda () n)))) r)",
    want: "5",
  },
}

// TestClosureKeepsScopeAcrossFrameReuse: eine Closure muss ihren
// lexikalischen Scope behalten, auch wenn zwischen Bau und Aufruf fremde
// Frames durch den Env-Pool laufen.
func TestClosureKeepsScopeAcrossFrameReuse(t *testing.T) {
  for _, tc := range brokenCases {
    t.Run(tc.name, func(t *testing.T) {
      got, err := evalClosureCase(t, tc.expr)
      if err != nil {
        t.Fatalf("%s: Closure hat Scope verloren: %v\nwollte: %s", tc.name, err, tc.want)
      }
      if got.String() != tc.want {
        t.Fatalf("%s: got %s, want %s", tc.name, got.String(), tc.want)
      }
    })
  }
}

// TestClosureScopeUnaffectedForms: Charakterisierung der Formen, die
// keinen eigenen Frame besitzen oder ihn korrekt markieren. Bricht einer
// dieser Fälle beim Fix, ist der Fix zu breit geraten.
func TestClosureScopeUnaffectedForms(t *testing.T) {
  for _, tc := range safeCases {
    t.Run(tc.name, func(t *testing.T) {
      got, err := evalClosureCase(t, tc.expr)
      if err != nil {
        t.Fatalf("%s: unerwarteter Fehler: %v", tc.name, err)
      }
      if got.String() != tc.want {
        t.Fatalf("%s: got %s, want %s", tc.name, got.String(), tc.want)
      }
    })
  }
}

// TestClosureScopeNoStackOverflow: derselbe Fehler OHNE churn. Der
// recycelte Frame zeigt zurück in die eigene Kette, Env.Get pendelt
// endlos. Go's "fatal error: stack overflow" ist NICHT vom recover() in
// evalWithCtx abfangbar — der Fall muss deshalb in einem Subprozess
// laufen, sonst reißt er die ganze Suite mit.
func TestClosureScopeNoStackOverflow(t *testing.T) {
  bin, err := filepath.Abs("../../build/golisp2")
  if err != nil {
    t.Fatalf("abs: %v", err)
  }
  if _, err := os.Stat(bin); err != nil {
    t.Skipf("./build/golisp2 fehlt (./build.sh) — Subprozess-Test übersprungen")
  }

  const src = `(begin
    (defun h (f) f)
    (defun g () (let ((n 5)) (h (let ((m 6)) (lambda () (list m n))))))
    (funcall (g)))`

  out, err := exec.Command(bin, "-e", src).CombinedOutput()
  text := string(out)

  if strings.Contains(text, "fatal error") || strings.Contains(text, "stack overflow") {
    t.Fatalf("unrecoverbarer Absturz statt Ergebnis (6 5):\n%s",
      firstLines(text, 5))
  }
  if err != nil {
    t.Fatalf("golisp2 -e schlug fehl: %v\n%s", err, firstLines(text, 5))
  }
  if got := strings.TrimSpace(text); got != "(6 5)" {
    t.Fatalf("got %q, want \"(6 5)\"", got)
  }
}

// firstLines schneidet Ausgabe für Fehlermeldungen zu — Go-Stacktraces
// sind mehrere Megabyte.
func firstLines(s string, n int) string {
  lines := strings.Split(s, "\n")
  if len(lines) > n {
    lines = lines[:n]
  }
  return strings.Join(lines, "\n")
}
