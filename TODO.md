# Aufgabe: 20260813

**Status:** ERLEDIGT — 20260813

Gefunden beim Bau eines externen golisp2-Programms (sixhat-Projekt,
6-Hüte-Methode mit KI-Ensemble via sigo). Beide Punkte reproduzierbar,
aktuell nur im sixhat-Projekt umgangen — Fix gehört hierher.

## 1. `(car '())` wirft Fehler statt `nil` — bricht `dolist`/`dotimes`

**Beobachtung:** Im aktuellen Build wirft `(car '())` einen Fehler
("car: Liste erwartet") statt `()` zu liefern.

**Auswirkung:** `embed/stdlib.lisp` erwartet an dieser Stelle offenbar ein
toleranteres `car`. Die Stdlib-Makros `dolist`/`dotimes` brechen dadurch,
sobald die Bindungsliste nur 2 Elemente hat (`(var lst)`, ohne
Ergebnisform) — der `caddr`-Zugriff auf die fehlende dritte Position
läuft über ein `car` auf `()`.

**Reproduktion (Kurzform):**
```lisp
(dolist (x '(1 2 3)) (print x))   ; bricht in diesem Build
```

**Fix:** `fnCar`/`fnCdr` (`lib/primitives.go`) akzeptieren jetzt `NIL`-Typ
zusätzlich zu `LIST`/`LAMBDA`/`MACRO`, liefern `MakeNil()` — CL-Semantik
hergestellt. `dolist`/`dotimes` mit 2-Element-Bindung laufen dadurch ohne
weitere Änderung an `embed/stdlib.lisp` korrekt durch. Tests:
`TestPrimitiveListEdges` (`lib/primitives_test.go`, angepasst),
`TestDolist2ElementBinding`/`TestDotimes2ElementBinding`
(`lib/stdlib_iter_test.go`, neu). `go test ./... -count=1`: 315 grün.

## 2. `(sigo-models)` liefert Name+Alias paarweise in einer flachen Liste

**Beobachtung:** `(sigo-models)` gibt für dasselbe Modell zwei
aufeinanderfolgende Einträge zurück — vollen Namen und Kurz-Alias, z.B.
`"gemini-3.1-flash-lite-image" "gem31-fltimg" "minimax-m2.7" ...`.

**Auswirkung:** Code, der z.B. "die ersten 3 verfügbaren Modelle" nehmen
will (`(take 3 (sigo-models))`), bekommt teils dasselbe Modell doppelt
statt 3 verschiedener Modelle — nicht offensichtlich beim Lesen des
Codes, nur beim Testen der tatsächlichen Werte auffällig.

**Workaround (nur im sixhat-Projekt):** `sixhat-jeder-zweite` filtert auf
jeden ersten Eintrag eines Paares, bevor `take 3` greift.

**Befund:** `fnSigoModels` (`lib/sigorest.go`) gibt 1:1 weiter, was der
externe sigoREST-Dienst unter `/v1/models` liefert — kein eigener
Aufbereitungsschritt in golisp2. Die Paarung entsteht also im sigoREST-
Dienst selbst (eigener Prozess, nicht Teil dieses Repos), nicht in
golisp2.

**Umsetzung:** Hinweis in `doc/sigo.md` ergänzt — `(sigo-models)` kann
Alias+Kanonischen-Namen als getrennte Listeneinträge liefern, Aufrufer
müssen selbst dedupen. Kein Code-Fix hier, Ursache liegt im externen
sigoREST-Dienst außerhalb dieses Repos.

## 3. Kein Primitiv zum Lesen von Freitext-Eingaben während eines laufenden Skripts

**Beobachtung:** Es gibt kein `read-line`/`prompt`-Äquivalent. `read`
parst nur einen bereits übergebenen String zu Lisp-Daten, liest nicht von
stdin. `system` liefert nur den Exit-Code, kein stdout/stdin-Durchgriff.
Nur der `-i`-REPL (go-prompt) liest interaktiv — das ist der Mensch, der
Lisp-Ausdrücke tippt, kein Primitiv, das ein laufendes Programm selbst
aufrufen kann.

**Wichtige Einschränkung (vorab geklärt):** golisp2s SWANK-Server
(`lib/swank/`) unterstützt kein Reverse-RPC "vom Client lesen" — nur
`connection-info`, `listener-eval`, `simple-completions`, `load-file`.
Ein neues `read-line`-Primitiv, das `os.Stdin` des golisp2-Prozesses
liest, funktioniert deshalb NICHT über eine Emacs/Sly-SWANK-Verbindung
(Emacs leitet Tastatureingaben nicht an den Stdin des Server-Prozesses
weiter). Nutzen hat es nur für eigenständige Terminal-Skripte
(`golisp2 skript.lisp`, direkt an einem TTY laufend, ähnlich `runREPL`s
`isTerminal`-Check in `main.go`).

**Umsetzung:** neues Primitiv `(read-line)` in `lib/primitives.go` —
liest eine Zeile von `os.Stdin` über einen package-level `bufio.Reader`
(erhalten zwischen Aufrufen), liefert String ohne Parsing. Einschränkung
(nur Datei-Argument-/Shebang-Modus, nicht SWANK/stdin-Default) als
Kommentar im Code und in `doc/cli.md` dokumentiert. Tests:
`TestReadLinePrimitive`, `TestReadLinePrimitiveEOF`,
`TestReadLinePrimitiveArgs` (`lib/stdin_read_test.go`, neu, via
`os.Pipe()`). Manuell verifiziert im Datei-Argument-Modus.
