# TODO

## Aufgabe — `defsystem`-lite (ASDF-orientiert, eigenes Ökosystem)

**Ziel:** Deklarative Systemdefinition + dependency-geordnetes, idempotentes Laden —
als Fundament eines eigenen GoLisp2-Ökosystems. Orientierung an ASDF-Konzepten,
keine ASDF-Kompatibilität (kein Compile-File, kein CLOS, keine Packages).

**Motivation:** Load-Reihenfolge und Abhängigkeiten sind heute implizit
(gps.lisp ↔ gps2.lisp ↔ test-helpers). DefLoc + Redef-Log + `makunbound`
liefern bereits die Infrastruktur dafür.

**Skizze (reines Lisp, ~150–250 Zeilen):**

```lisp
(defsystem gps2
  :depends-on (test-helpers)
  :components ("pn-gps1/gps2.lisp"))

(load-system 'gps2)        ; Deps topologisch, jede Datei einmal
(system-symbols 'gps2)     ; via DefLoc: welche Symbole definiert das System?
(unload-system 'gps2)      ; via makunbound
```

**Optionen:** `:redefine :allow` als Hülle (wie gps2-tests manuell),
`(loaded-systems)`-Abfrage, Fehler bei Zyklen in `:depends-on`.

**Bewusst nicht:** Quicklisp-Analogon (kein Ökosystem zum Verteilen),
ASDF-Kompat (bräuchte Packages + CLOS), Compile-File (Interpreter).

## Aufgabe — Mini-Test-Framework (Nachfolger von `assert=`)

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

## Aufgabe — Condition-lite (Fehler mit Kontext)

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

## Aufgabe - GOLISP2 Semantik und Syntax in Kurzform 

**Ziel:** Wenn wir andere KIs einsetzen wollen, brauchen diese einen Überblick über die Semantik und Syntax von GOLISP2, aber auch Erläuterungen zu den Schwächen.

**Motivation:** Wenn GOLISP2 erst analysiert werden muß, wird ein Vorgang häufig wiederholt, was nicht notwendig sein sollte. 

**Optionen:** Die Zusammenfassung soll einmal für KIs in tokenoptimierter Form erfolgen und für die Menschen ausführlicher.
