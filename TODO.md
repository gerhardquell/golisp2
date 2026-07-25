# TODO

## Erledigt 2026-07-25 — GOLISP2 Semantik/Syntax-Kurzform (Option B: zwei Dateien)

**Ziel:** Wenn wir andere KIs einsetzen wollen, brauchen diese einen Überblick über die Semantik und Syntax von GOLISP2, aber auch Erläuterungen zu den Schwächen.

**Motivation:** Wenn GOLISP2 erst analysiert werden muß, wird ein Vorgang häufig wiederholt, was nicht notwendig sein sollte.

**Ergebnis (Option B + eigener Schwächen-Abschnitt):**
- `doc/ki/referenz.md` (316 Zeilen) — KI-Form: Tabellen, Präfixe, tokenoptimiert
- `doc/golisp2-cheatsheet.md` (771 Zeilen) — Mensch-Form: Beispiele, Erklärungen
- Beide haben §10 "Schwächen (bewusst)" als eigenen Abschnitt (12 Punkte:
  kein Package, kein CLOS, kein Condition-System, progv lex/dyn, kein Compile-File,
  kein Small-Int-Cache, macrolet nicht-rekursiv, kein Continuations/MOP,
  load-in-defun, kein GC-Tuning, kein Typ-System, kein LOOP)

## Nächste Aufgaben

### Aufgabe — Mini-Test-Framework (Nachfolger von `assert=`)

**Ziel:** `assert=` aus `tests/test-helpers.lisp` zu echtem Framework ausbauen —
Testsuiten, Fehler sammeln statt abbrechen, Abschluss-Report.
Orientierung an FiveAM-Konzepten, kein Port.

**Motivation:** Heute bricht ein FAIL den Datei-Load ab; nachfolgende Tests
laufen nicht. Es gibt kein Zählen, kein Gruppieren, keinen Report.

**Skizze (reines Lisp):**

```lisp
(deftest mengen-ops
  (is (equal? '(1 2 3) (union '(1 2) '(2 3))))
  (is (equal? '(1 3) (set-difference '(1 2 3) '(2)))))

(run-tests)   ; alle Suiten, Report: n PASS / m FAIL, Fehlerdetails
```

**Optionen:** `deftest`-Registry, `(run-tests 'suite)` für Einzelsuiten,
Exit-relevantes Ergebnis für `golisp2 -t`, expected-failure-Markierung.

### Aufgabe — Condition-lite (Fehler mit Kontext)

**Ziel:** Fehler mit strukturiertem Kontext statt nur String —
Signalisieren, Abfangen, optional Restarts. Orientierung am CL-Condition-System, stark reduziert.

**Motivation:** `error`/`catch` liefern heute nur Message-Strings. Aufrufer
können Fehlerarten nicht programmatisch unterscheiden (z. B. „Datei nicht
gefunden" vs. „Parse-Fehler"), geschweige denn Recovery anbieten.

**Skizze:**

```lisp
(define-condition 'file-error '(io-error) '((path :reader file-error-path)))
(signal 'file-error :path "x.lisp")
(handler-case (load "x.lisp")
  (file-error (e) (format t "fehlt: ~a" (file-error-path e))))
```

**Optionen:** Condition-Hierarchie (einfacher Typ-Tag mit Eltern),
`handler-bind`-lite, einfache Restarts (`retry`/`use-value`) — nur wenn
konkreter Bedarf. Bewusst nicht: volles CL-Restart-Protokoll, MOP-Integration.
