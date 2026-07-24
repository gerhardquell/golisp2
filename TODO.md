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
