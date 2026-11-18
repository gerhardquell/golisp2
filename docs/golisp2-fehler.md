# golisp2 — Fehler- und Fallstrick-Archiv

> Sammlung aller bekannten golisp2-Stolperfallen, Dokubugs und
> Dialekt-Limitationen, gesammelt bei der Lispbuch-Überarbeitung.
> Quelle der Wahrheit für Fallstricke bleibt CLAUDE.md (Abschnitt
> „Kritische Fallstricke"); diese Datei archiviert sie TODO-gerecht
> mit Status + Kontext für spätere Nachbereitung.
>
> Status-Legende:
> - **[FIXED]** — im golisp2-Quellcode behoben
> - **[DOKU-FIXED]** — Doku-Fehler korrigiert (Code unverändert)
> - **[OPEN-DOKU]** — Doku lückenhaft/irreführend, Code korrekt
> - **[LIMITATION]** — bewusste Dialekt-Entscheidung, kein Bug
> - **[KNOWN]** — Verhalten bestätigt, beim Arbeiten beachten

Stand: 20260815

---

## A. Korrigierte Doku-Fehler (CLAUDE.md)

### A1. Fehlermodell: `catch` vs `trap`  [DOKU-FIXED]
**Datum:** 20260815 (Kap15)
**Symptom:** CLAUDE.md zeigte `(catch (/ 1 0) (lambda (e) ...))` als
Fehlerfänger. Tatsächlich crasht das — `catch` fängt nur `throw`-Sentinels,
keine errors.
**Korrektur:** CLAUDE.md-Abschnitt „Fehlermodell" auf `(trap ...)` umgestellt
+ Warnung ergänzt. Memory-Eintrag `golisp2-catch-vs-trap-fehlermodell.md`.
**Wahrheit (Quelle `src/lib/eval_control.go`):**
- `catch`/`throw` = CL-dynamischer Kontrollsprung `(catch tag body...)` /
  `(throw tag wert)` — fängt **nur** `*throwValue`, keine errors.
- `trap` = error-handler `(trap body handler)` — Handler kriegt Fehler-String,
  fängt LispError + Go-Primitive-Fehler, lässt throw/return-from durch.
**Kapitel-Lehre:** Kap15 Abschnitt 5 + Aufgabe 5.

---

## B. Bestätigte Dialekt-Limitationen (bewusst, keine Bugs)

### B1. `defstruct` generiert kein `-p`-Prädikat  [LIMITATION]
**Datum:** 20260815 (Kap15)
**Symptom:** `(punkt-p x)` → „unbekanntes Symbol". CL generiert das Prädikat
automatisch, golisp2 nicht.
**Workaround:** selbst bauen —
`(defun X-p (v) (and (list? v) (equal? (car v) (quote X))))`
(Typ-Tag = erstes Symbol der Listen-Repräsentation).
**Grund:** bewusst klein gehalten (Kapitel 1: „Was kann weg?").
**Kapitel-Lehre:** Kap15 Abschnitt defstruct + Aufgabe 2.

### B2. `defstruct` ohne `:include` (Vererbung)  [LIMITATION]
**Datum:** 20260815 (Kap15)
**Symptom:** `(defstruct (sportwagen :include fahrzeug) ...)` →
`make-sportwagen` unbekannt. Vererbung nicht unterstützt.
**Workaround:** Felder manuell koppeln oder flache Struktur.
**Grund:** golisp2 verzichtet auf CLOS-Nähe.
**Kapitel-Lehre:** Kap15 CL-Tabelle + Aufgabe 7.

### B3. `setf` nur eine Place pro Form  [LIMITATION]
**Datum:** 20260815 (Kap15)
**Symptom:** `(setf a 1 b 2)` → „zu viele Argumente". CL erlaubt Multi-Place.
**Workaround:** separate `(setf a 1)` `(setf b 2)`.
**Kapitel-Lehre:** Kap15 Abschnitt setf + Aufgabe 7.

### B4. `setf (nth i ...)` braucht Symbol als Place  [LIMITATION]
**Datum:** 20260815 (Kap15)
**Symptom:** `(setf (nth 2 (nth 1 matrix)) 99)` →
„nth-Place-Argument muss ein Symbol sein". Kein Chaining wie CL.
**Ursache:** immutable cells → golisp2 rebindet äusserste Variable; Rebinding
gibt es nur für Symbole, nicht Ausdrücke.
**Workaround:** Zeile extrahieren, ändern, Matrix rebinden.
**Kapitel-Lehre:** Kap15 Aufgabe 1.

### B5. Keine `rplaca`/`rplacd` (immutable cells)  [LIMITATION]
**Bereits bekannt** (CLAUDE.md Rekursion & Performance). Hier bestätigt:
`rplaca`/`rplacd` = unbekannte Symbole. `setf` simuliert Mutation via Rebinding.
**Kapitel-Lehre:** Kap13 (Immutable Cells), Kap15 CL-Tabelle.

### B6. Arithmetik-Namen: `floor`/`mod`, nicht `div`/`quotient`  [TEILWEISE BEHOBEN]
**Datum:** 20260815 (Kap15), aktualisiert 20260903
**Symptom:** `div`/`quotient`/`modulo` → unbekannt.
**Verfügbar:** `floor`, `ceiling`, `truncate`, `round` (alle CL-Semantik, 2 Werte:
Quotient + Rest), `mod`/`remainder` (Rest).
**Achtung:** CL-Code-Port braucht weiterhin Namensanpassung für `div`/`quotient`/`modulo`.

### B7. Kein `ignore-errors`, kein `handler-case`  [LIMITATION]
**Datum:** 20260815 (Kap15)
**Symptom:** `ignore-errors` unbekannt. Fehlerbehandlung nur via `trap`
(ein Handler, ein String, kein Typ-Dispatch).
**Workaround:** `trap` mit String-Parsing für differnzierte Behandlung, oder
vorab `if`-Prüfung (vorzuziehen).
**Kapitel-Lehre:** Kap15 Abschnitt trap + Aufgabe 5.

---

## C. Bestätigte Verhaltens-Fallstricke (beim Arbeiten beachten)

### C1. `throw` ohne `catch` ist fatal — `trap` fängt nicht  [KNOWN]
**Datum:** 20260815 (Kap15)
**Symptom:** `(throw 'lost 1)` ohne umschliessendes catch → crash.
`trap` fängt es **nicht** (throw ist Kontrollfluss-Sentinel, wird durchgereicht).
**Kapitel-Lehre:** Kap15 Fallstrick 2.

### C2. `catch` fängt keine errors  [KNOWN]
**Datum:** 20260815 (Kap15)
**Symptom:** `(catch 'tag (error "boom"))` → crash (nicht gefangen).
`catch` fängt nur `throw`, nie `error`. Fehler → `trap`.
**Kapitel-Lehre:** Kap15 Fallstrick 1 (der zentrale).

### C3. `define (name args) body` ist ungültig  [FIXED 20260902]
**Datum:** 20260813 (Kap13), 20260815 (Kap15)
**Symptom:** `(define (f x) ...)` → „define: Syntax: (define name value)".
golisp2 `define` nimmt nur `(define name value)`, keine Funktionsdefinition.
**Fix:** `defun` nutzen. (CL/Gewohnheits-Falle.)
**Kapitel-Lehre:** Kap13.
**Fix 20260902:** `evalDefine` akzeptiert jetzt auch
`(define (name params...) body...)` als Zucker für
`(define name (lambda (params...) body...))` — Semantik identisch zu
`defun`. Die Gewohnheits-Falle existiert nicht mehr.

### C4. Mehrfachwerte kollabieren in single-value-Kontext  [KNOWN]
**Datum:** 20260815 (Kap15)
**Symptom:** `(define x (values 1 2 3))` → `x` ist `1`, nicht `(1 2 3)`.
Wer alle Werte will: `multiple-value-bind` / `multiple-value-list`.
**Kapitel-Lehre:** Kap15 Abschnitt Mehrfachwerte + Konzept 5.

### C5. `set!` gibt Symbol (Variablenname) zurueck, nicht den Wert  [KNOWN]
**Datum:** 20260815 (Kap17)
**Symptom:** `(set! x (cons "neu" x))` → Rueckgabe ist das Symbol `x`,
nicht die neue Liste. `(define r (set! x 5))` → `r` = Symbol `x`, nicht `5`.
**Ursache:** golisp2 `set!` (Rebinding) evaluiert zum Namen, nicht zum Wert
(abweichend von CL `setq`, das den Wert liefert; Scheme `set!` ist
unspezifiziert).
**Workaround:** nach `set!` den Variablennamen explizit als Rueckgabe
angeben — in `cond`/Lambda-Bodies via `begin`:
`(begin (set! tasks (cons arg tasks)) tasks)`.
**Kapitel-Lehre:** Kap17 todo-System + Aufgabe 3 (make-todo-system).
**Achtung:** Viele Closure-basierte Zustandssysteme (let-over-lambda)
brauchen diese `begin`-Form — sonst gibt die Aktion den Namen statt des
aktuellen Werts zurueck.

---

## D. Aus CLAUDE.md „Kritische Fallstricke" (Referenz, dort detailliert)

Diese 8 sind in CLAUDE.md dokumentiert und gültig — hier nur Index:

1. Kein `undefine` — letzter Schreiber gewinnt (Primitiv-Überschreibung warnt REDEF).
2. Spezialformen vor Makros geprüft — `defmacro if` ist toter Code.
3. `eq` = Pointer, `equal?` = strukturell; nil ist Singleton.
4. `(eval form)` läuft im globalen Env, nicht Lambda-Scope.
5. `rg` muss `*.lisp` einschließen.
6. SWANK: kein Reverse-RPC, `read-line` nur am TTY.
7. `(car '())`/`(cdr '())` → `()` ab Commit b18ce54 (20260813) [FIXED].
8. `(sigo-models)` kann Alias+Kanonisch doppelt liefern — dedupen.

---

## E. Offene TODOs / Nachbereitung

- **[TODO]** Bei nächster golisp2-Quellcode-Arbeit prüfen, ob `defstruct -p`
  und/oder `:include` leicht nachrüstbar — aktuell per Workaround umgangen.
- **[TODO]** `setf`-Multi-Place und verschachtelte Places evtl. als
  Makro-Erweiterung in golisp2 möglich (Kap14-Wissen) — falls Bedarf.
- **[TODO]** `ignore-errors`-Äquivalent als dünnes Makro über `trap` bauen?
- **[TODO]** Bei Kapitel-Portierung CL→golisp2 stets Arithmetik-Namen
  anpassen (`floor`/`mod` statt `div`/`quotient`/`truncate`).

---

## F. Runde golisp2dfs: Mehrfachwerte, Channels, Gotchas (20261018, 0.5.1)

Fund-Quelle: golisp2dfs (DataFlowSystem, `exp/dfs`). Umgesetzt (Plan
20261018): Mehrfachwerte in Spezialformen, Channels mit Timeout/Schließen,
`spawn` mit Ergebnis-Channel, `chan-select`.

### F1. Mehrfachwerte in Spezialformen (`lock`, `case`, 21 Formen)  [FIXED 0.2.1]
**Symptom:** Spezialformen, die das Ergebnis eines Teilausdrucks inspizieren,
sahen die Mehrfachwert-Zelle statt des Primärwerts: `lock` meldete „kein
Mutex“ (Mutex aus `gethash` o. ä.), `case` nahm still den falschen Zweig.
**Fix:** 21 Spezialformen packen Mehrfachwerte über `Primary()` aus
(u. a. `lock`, `case`, `catch`/`throw`, `do`, `define`, `,@`, `exec`, `load`).
Eine Quelle: `Primary()` in `types.go`.

### F2. `cond`-Klausel ohne Körper wertete den Test doppelt aus  [FIXED 0.5.1]
**Symptom:** `(cond ('foo))` → Fehler „unbekanntes Symbol 'foo'“;
`(cond ((list 1 2)))` rief `(1 2)` als Funktion auf.
**Fix:** Klausel ohne Körper liefert den **Primärwert** des Tests (CL).

### F3. `prog1`/`prog2`/`or` reichten Mehrfachwerte durch  [FIXED 0.5.1]
**Symptom:** `(multiple-value-list (prog1 (values 1 2) 3))` → `(1 2)`.
**Fix:** `prog1`, `prog2` und vordere Formen von `or` liefern nur den
Primärwert (CL). Die letzte Form von `or` bleibt Tail; `and` unverändert.

### F4. `parfunc :timeout` ignorierte Bruchteile  [FIXED 0.5.1]
**Symptom:** `:timeout 0.2` wirkte nicht (kein Timeout), ohne Fehlermeldung.
**Fix:** Bruchteile werden gelesen; Nicht-Zahl oder negativ → Fehler
`parfunc: :timeout erwartet Zahl >= 0 (Sekunden)`.
**Beachten [KNOWN]:** Das Argument wird nicht ausgewertet — nur Zahl-Literal.
**Beachten:** `:timeout 0` heißt bei `parfunc` „kein Timeout“, bei
`chan-recv` und `chan-select` dagegen „sofort zurück, nicht warten“.

### F5. Channels ohne Timeout, try, close, select  [FIXED 0.3.0 / 0.5.0]
**Symptom:** `chan-recv` blockierte ewig (auch auf geschlossenem Channel).
**Fix:** `(chan-recv ch [:timeout sek])` → `(values x t)` / `(values nil nil)`;
neu `chan-try-recv`, `chan-try-send`, `chan-close`, `chan-closed?`
(geschlossen **und** leer); `chan-send` auf geschlossenem Channel → Fehler;
`(chan-select chans [:timeout sek])` → `(values x t i)` / `(values nil nil nil)`.

### F6. `spawn`-Fehler nur auf stderr  [FIXED 0.4.0]
**Symptom:** `(spawn fn)` lieferte `nil`; Ergebnis und Fehler gingen verloren.
**Fix:** `spawn` liefert einen Ergebnis-Channel mit genau einer Nachricht
`(:ok wert)` bzw. `(:error "text")`. Die stderr-Meldung bleibt.

### F7. `puthash`-Argumentreihenfolge  [KNOWN]
`(puthash key table value)` — Tabelle an 2. Stelle, anders als Emacs Lisp
`(puthash key value table)`. Empfohlen: `(setf (gethash k h) v)`.

### F8. `sleep` in Millisekunden, `:timeout` in Sekunden  [KNOWN]
`(sleep ms)` erwartet Millisekunden; `:timeout` (`chan-recv`, `chan-select`,
`parfunc`) Sekunden. `(sleep 0.05)` wirkt praktisch nicht.

### F9. Fehlende CL-Namen, `last`, `member :test`  [LIMITATION]
- `stringp`, `consp`, `string=`, `equal` fehlen → `string?`,
  `(typep x 'cons)`, `equal?`.
- `(member x l)` kennt kein `:test` und vergleicht Strings strukturell.
- `(last l)` liefert das Element, nicht die letzte Zelle.
- `(slurp)` liest stdin (Datei → `file-read`); `(gets)` am Eingabeende
  → Fehler `gets: EOF`, nicht `nil`.

---

## G. Ganzzahl-Typ: zwei Zahlarten (20261116, 0.31.0)

Seit 0.31.0 gibt es Ganzzahl (`int64`) und Float (`float64`). Vorher war jede
Zahl ein `float64`. Regeln: `docs/lisp-semantik.md`, Abschnitt „Zahlen“.

### G1. Ganzzahlen über 2^53 wurden still gerundet  [FIXED 0.31.0]
**Symptom:** `(+ 9007199254740992 1)` → `9007199254740992`.
**Fix:** Ganzzahlen rechnen exakt auf `int64`: → `9007199254740993`.

### G2. Überlauf ist ein Fehler  [KNOWN]
`(* 99999999999 99999999999)` → Fehler `*: Ganzzahl-Überlauf` (vorher
`9.9999999998e+21`). Es gibt kein Bignum. Wer den Näherungswert will, rechnet
mit Floats: `(* 99999999999.0 99999999999)` → `9.9999999998e+21`.

### G3. Floats drucken mit Punkt  [KNOWN]
`(sqrt 16)` → `4.0`, `(* 1.5 2)` → `3.0`, `1e3` → `1000.0` (vorher `4`, `3`,
`1000`). Auch `(format nil "~d" 3.0)` → `"3.0"`. Für eine Ganzzahl:
`(round x)`, `(floor x)` oder `(format nil "~,0:f" x)`.

### G4. `1` und `1.0` sind bei `eql`/`equal?` verschieden  [KNOWN]
`(eql 1 1.0)` und `(equal? 1 1.0)` → `()` (vorher `t`). `(= 1 1.0)` → `t`.
Folgen: `(member 1.0 (list 1 2))` → `()`, `(case 1.0 (1 …))` trifft nicht,
`1` und `1.0` sind zwei Hash-Schlüssel, `(catch 1 (throw 1.0 …))` passt nicht.
`(typep 3.0 'integer)` → `()`.

### G5. Float als Index oder Anzahl ist ein Fehler  [KNOWN]
`(nth 1.7 l)` → Fehler `nth: ganze Zahl erwartet, got 1.7` (vorher `()`).
Das gilt auch für `2.0` und für `aref`, `substring`, `iota`, `dotimes`,
`loop repeat`, `evenp`. Typische Falle: `(nth (/ n 2) l)` geht nur, wenn `n`
gerade ist — `(/ 7 2)` → `3.5`. Abhilfe: `(nth (floor n 2) l)`.

### G6. `(random 1.0)` liefert einen Float  [FIXED 0.31.0]
`(random 1.0)` → Float in [0, 1) (vorher immer `0`). `(random 10)` → Ganzzahl.

### G7. JSON behält die Zahlart  [KNOWN]
`(json-encode 3.0)` → `"3.0"`, `(json-parse "3.0")` → `3.0`,
`(json-parse "3")` → `3` (Ganzzahl). Ganzzahlen bis `int64` bleiben exakt:
`(json-parse "9007199254740993")` → `9007199254740993`.

### G8. Ganzzahl gegen Float über 2^53 ist ungenau  [LIMITATION]
`(= 9007199254740993 9007199254740992.0)` → `t`. Beim gemischten Vergleich
wird die Ganzzahl nach Float gewandelt. Ganzzahl gegen Ganzzahl ist exakt.

### G9. `(random)`, `~va`, `(length "abc")`, `~r` mit Float  [FIXED 0.32.0, 20261117]
- `(random)` ohne Argument liefert einen Float aus [0, 1) (vorher eine große Ganzzahl —
  `(< (random) 0.4)` in den Demos war dadurch fast nie wahr).
- `~va`/`~vs` verbrauchen den Breiten-Parameter vor dem Wert (CL 22.3), wie schon `~vd`:
  `(format nil "~va|" 5 3)` → `"3    |"`.
- `(length "abc")` → `3` (vorher `cdr: Liste erwartet`).
- `~r` ohne Basis mit Float ist ein Fehler (CL verlangt eine Ganzzahl), vorher still gekürzt;
  `~nr` druckt einen Float wie `~a`, wie `~d`.


## H. Dynamische Variablen (20261117, 0.33.0)

Seit 0.33.0 sind `defvar`-/`defparameter`-Namen special wie in CL. Regeln:
`docs/lisp-semantik.md`, Abschnitt „Dynamische Variablen“.

### H1. `let` auf einen `defvar`-Namen bindet dynamisch  [FIXED 0.33.0]
**Symptom:** `(defvar *x* 1) (defun g () *x*) (let ((*x* 2)) (g))` → `1`;
die aufgerufene Funktion sah die Bindung nie.
**Fix:** → `2`. Gilt für `let`, `let*` und Makros, die zu `let` expandieren
(`dolist`, `dotimes`, `loop … for`). `destructuring-bind` ist seit 0.36.0
eine Spezialform und weist special-Namen ab (J5).

### H2. `progv` wirkt jetzt auf aufgerufene Funktionen  [FIXED 0.33.0]
**Symptom:** `(progv '(*x*) '(3) (g))` → `1`; `progv` legte nur ein
lexikalisches Child-Env an und überdeckte lexikalische Bindungen.
**Fix:** → `3`. Eine lexikalische Bindung schützt jetzt wie in CL:
`(let ((x 2)) (progv '(x) '(1) x))` → `2`.

### H3. Lambda-Parameter, `do`, `multiple-value-bind` mit special-Namen  [KNOWN]
`(defvar *x* 1) (defun f (*x*) …)` → Fehler
`defun: Parameter *x* ist special — mit let binden`. Ebenso `do`/`do*`
(`do: Variable *x* ist special …`) und `multiple-value-bind`. CL bände
dynamisch; golisp2 bindet dort nur lexikalisch und meldet es laut.
Abhilfe: anderer Parametername oder `(let ((*x* p)) …)` im Rumpf.

### H4. Lambda vor dem `defvar` bindet weiter lexikalisch  [LIMITATION]
Lambda-Listen werden bei der Definition geprüft, nicht beim Aufruf (kostet
sonst auf dem heißesten Pfad). `(defun alt (*z*) (list *z* (lies)))
(defvar *z* 1) (alt 2)` → `(2 1)`: `alt` bindet `*z*` lexikalisch, `lies`
sieht den globalen Wert. Abhilfe: `defvar` vor den Funktionen. Siehe H8.

### H5. Closure merkt sich die Bindung nicht  [KNOWN]
`(defun mach () (let ((*x* 2)) (lambda () *x*)))  (funcall (mach))` → `1`
(vor 0.33.0: `2`). Wie CL: die Bindung endet mit dem `let`. Wer den Wert
festhalten will, bindet ihn lexikalisch: `(let ((w *x*)) (lambda () w))`.

### H6. Handler beginnen ohne Bindungen  [KNOWN]
HTTP-Handler (`http-serve`, `webserv`), WS-Bridge, GUI-Callbacks und
SWANK-Verbindungen sehen nur die globalen Werte, auch wenn der Server in
einem `let` gestartet wurde. `parfunc`, `spawn`, `pool-map` und `ga-calc`
erben dagegen die Bindungen; ein `setq` im Worker trifft dieselbe Bindung
und ist für den Starter sichtbar.

### H7. `defvar` ohne `*…*` ist ein Fehler; interne Variablen umbenannt  [KNOWN]
`(defvar n 1)` → `defvar: n ohne *…* — special-Namen brauchen *n*`
(ebenso `defparameter`). Zuerst war es nur eine Warnung — dann scheiterten
nach `(defvar x)` `remove`, `handler-case`, `ignore-errors`, `defstruct` und
`typecase`, weil die stdlib zur Laufzeit `(lambda (x) …)` erzeugt. Kein
Teil-Effekt: Nach dem Fehler ist der Name weder gebunden noch special.
Interne Variablen heißen deshalb `*generic-registry*` (vorher
`%generic-registry`), `*swank--insp-…*` (vorher `swank--insp-…`) und
`*tf-err-tag*` (vorher `%tf-err-tag`). Wer sie direkt benutzt hat, muss
nachziehen.

### H8. `let` auf einen lexikalisch gebundenen special-Namen  [FIXED 0.33.0]
**Symptom:** `(defun dbl (*p*) (let ((*p* (* *p* 2))) *p*)) (defvar *p* 0)
(dbl 5)` → `5`: das `let` band dynamisch, gelesen wurde aber der Parameter.
**Fix:** → `10`. Hält ein lexikalischer Frame den Namen (Parameter von vor
dem `defvar`), bindet `let`/`let*` ihn lexikalisch weiter; `progv` bleibt
dynamisch.

### H9. `defparameter` in einer dynamischen Bindung  [FIXED 0.33.0]
**Symptom:** `(let ((*x* 2)) (defparameter *x* 9) (g))` → `2`, danach
`*x*` → `9`: der globale Wert wurde gesetzt, die Bindung nicht.
**Fix:** → `9`, danach `1` — wie CL (`symbol-value`).

### H10. `makunbound` in einer dynamischen Bindung  [FIXED 0.33.0]
`(let ((*x* 2)) (makunbound '*x*))` entfernte still den globalen Wert. Jetzt
Fehler `makunbound: *x* ist dynamisch gebunden`.

## I. Sandbox (20261117, 0.34.0)

Seit 0.34.0 gibt es `make-sandbox`/`sandbox-eval` für KI-Code. Regeln:
`docs/lisp-semantik.md`, Abschnitt „Sandbox“. Jede Sandbox baut ein eigenes
`BaseEnv` — dabei fielen zwei globale Zustände auf, die nur das zuletzt
gebaute Env kannten. Sie betrafen vorher schon jede weitere
SWANK-Verbindung (`handleConn` baut je Verbindung ein `BaseEnv`).

### I1. `~/fun/` suchte im zuletzt gebauten Env  [FIXED 0.34.0]
**Symptom:** `format` merkte sich das Env in der globalen `globalFormatEnv`,
der letzte `RegisterFormat`-Aufruf gewann. Nach `(make-sandbox)` hätte
`(format nil "~/up/" "hi")` die globale Funktion `up` nicht mehr gefunden.
**Fix:** `format` fängt sein Env bei der Registrierung: global → `"HI"`; in
der Sandbox findet `~/fun/` nur Sandbox-Funktionen
(`format: ~/up/: env: unbekanntes Symbol 'up'`).

### I2. `trace`-Zustand war global  [FIXED 0.34.0]
**Symptom:** `RegisterTrace` setzte Root und Originale global zurück. Nach
`(trace 'tr) (make-sandbox)` fand `(untrace 'tr)` das Original nicht mehr.
**Fix:** Zustand je Root-Env → `(untrace 'tr)` → `tr`.

### I3. Zeitlimit hielt bei blockierenden Operationen nicht  [FIXED 0.34.0, Rest LIMITATION]
**Symptom:** Mit `:zeitlimit 1` hing `sandbox-eval` bei `(sleep 3000)` 3 s,
bei `chan-recv`/`chan-send`/`chan-select` ohne Partner und beim
`lock`-Deadlock für immer.
**Fix:** `sandbox-eval` wertet in eigener Goroutine aus und kehrt immer zur
Frist zurück; `sleep` und `chan-*` brechen mit dem Kontext ab.
**Rest:** Was den Abbruch nicht sieht (`lock`, großes `sort`, Regex,
`sigo`-HTTP-Aufruf), läuft im Hintergrund weiter, ein `lock`-Deadlock für
immer. `sigo` hat seinen eigenen Timeout (`GOLISP_SIGO_TIMEOUT`).

### I4. Speicher pro Sandbox bleibt  [LIMITATION]
Jede Sandbox kostet rund 1 MB (eigenes `BaseEnv` + stdlib, gemessen mit
`(memstats)`: 10 Sandboxen ≈ +11 MB) und wird bis Prozessende nicht
freigegeben (Zustandstabelle in `sandbox.go`). Eine Sandbox wiederverwenden
statt eine pro Aufruf. Quoten für Speicher und CPU gibt es nicht.

### I5. Callbacks sehen den Abbruch-Kontext auch außerhalb der Sandbox  [KNOWN]
Verhaltensänderung: Callbacks aus Primitiven (`mapcar`, `sort`, `maphash`,
`format ~/fun/` …) erben jetzt den Abbruch-Kontext des Aufrufers — nötig,
damit das Zeitlimit dort greift. Eine Endlosschleife in einem
`mapcar`-Callback innerhalb eines abgebrochenen `parfunc :timeout` bricht
jetzt ab; vorher lief sie still weiter. Ebenso brechen `sleep` und `chan-*`
im abgebrochenen Worker jetzt mit `eval: cancelled` ab.

### I6. Verschachtelte Sandbox hatte volle Rechte  [FIXED 0.34.0]
**Symptom:** `(make-sandbox :erlaube '(make-sandbox sandbox-eval))` — darin
konnte eine neue Sandbox sich `system` freigeben.
**Fix:** `make-sandbox`/`sandbox-eval` sind nie freigebbar →
`make-sandbox: make-sandbox ist nicht freigebbar`.

### I7. `file-read` auf Gerätedateien  [FIXED 0.34.0]
**Symptom:** `(file-read "/dev/zero")` in der Sandbox las ohne Ende und fraß
Speicher bis zum Absturz.
**Fix:** In der Sandbox liest `file-read` nur reguläre Dateien (auch nicht
`sys-stdin`); außerhalb unverändert.

## J. destructuring-bind und Makro-Lambda-Listen (20261118, 0.36.0)

`destructuring-bind` ist seit 0.36.0 eine Spezialform und bindet über
denselben Weg wie Lambda-Listen. Regeln: `docs/lisp-semantik.md`,
Abschnitt „Lambda-Listen“.

### J1. `destructuring-bind` nur flach  [FIXED 0.36.0]
**Symptom:** Verschachtelte Muster, Punktpaare und `&optional`/`&rest`/`&key`
waren ein Fehler (`destructuring-bind: &rest nicht unterstützt …`).
**Fix:** Lambda-Liste wie CL:
`(destructuring-bind (a (b &optional (c 9)) . r) '(1 (2) 3 4) (list a b c r))`
→ `(1 2 9 (3 4))`.

### J2. Falsche Anzahl ging still durch  [FIXED 0.36.0, Breaking]
**Symptom:** `(destructuring-bind (a b) '(1) b)` → `()`; überzählige Werte
wurden ignoriert.
**Fix:** Fehler `destructuring-bind: zu wenig Werte (brauche 'b')` bzw.
`destructuring-bind: zu viele Werte (2 überzählig)`. `&optional` und `&rest`
erlauben die Abweichung.

### J3. Makros zerlegten nicht verschachtelt  [FIXED 0.36.0]
**Symptom:** `(defmacro mit-paar ((x y) &body body) …)` wurde angenommen, der
Aufruf scheiterte mit `lambda: Parameter muss Atom sein`.
**Fix:** `defmacro` und `macrolet` zerlegen verschachtelt wie CL. Fehler beim
Makro-Aufruf behalten das Präfix `lambda:`
(`lambda: Liste erwartet für (x y), bekam p`).

### J4. Muster in `lambda`/`defun` erst beim Aufruf gemeldet  [FIXED 0.36.0]
**Symptom:** `(lambda ((a b)) a)`, `(lambda (&key ((:kw v) 1)) v)` und
`(lambda (&rest (a b)) a)` wurden angenommen und scheiterten erst beim
Aufruf, teils irreführend (`unbekanntes Schlüsselwort :kw`,
`unbekanntes Symbol 'a'`).
**Fix:** Definitionsfehler mit dem Namen der definierenden Form:
`lambda: Parameter muss Atom sein`, `defun: Parameter muss Atom sein`; in
`defmacro`: `defmacro: verschachteltes Muster nur an Pflichtstellen`.

### J5. special-Name in `destructuring-bind`  [KNOWN]
Vor 0.36.0 expandierte `destructuring-bind` zu `let` und band
`(defvar *x* 1)`-Namen dynamisch. Jetzt Fehler wie bei Lambda-Parametern (H3):
`destructuring-bind: Parameter *x* ist special — mit let binden`.
`macroexpand` liefert für `destructuring-bind` keine `let`-Kette mehr.

### J6. Muster in `&optional`/`&key`/`&rest`  [LIMITATION]
CL erlaubt sie, golisp2 nicht:
`destructuring-bind: verschachteltes Muster nur an Pflichtstellen`. `&whole`
und `&environment` gibt es nicht.

## K. Kleinkram (20261118, 0.36.1)

### K1. `(quote)` und `(if)` gingen still durch  [FIXED 0.36.1, Breaking]
**Symptom:** `(quote)` → `()`, `(if)` und `(if t)` → `()`; überzählige Teile
(`(quote a b)`, `(if t 1 2 3)`) wurden still ignoriert.
**Fix:** Wie CL: `quote` braucht genau 1 Argument, `if` 2 oder 3. Sonst
`quote: Syntax: (quote x)` bzw. `if: Syntax: (if test dann [sonst])`.

### K2. Zweites `maxima-close` war ein Fehler  [FIXED 0.36.1]
**Symptom:** `(maxima-close s)` zweimal → `maxima-close: Session bereits
geschlossen`; der Zustand wurde dabei ohne Lock gelesen (Datenrennen mit
nebenläufigem `maxima-eval`).
**Fix:** Zweites `maxima-close` → `t`. `maxima-eval` nach dem Schließen
meldet weiter `maxima-eval: Session bereits geschlossen`.

### K3. Postgres ignorierte Sandbox-Frist und `parfunc :timeout`  [FIXED 0.36.1]
**Symptom:** `(pg-query db "select pg_sleep(5)")` lief nach Ablauf der Frist
weiter — `sandbox-eval` kehrte zwar zurück, das Statement lief aber im
Hintergrund und auf dem Server bis zum Ende.
**Fix:** `pg-query`/`pg-exec` brechen mit dem Kontext ab (`eval: cancelled`),
der Treiber bricht das Statement auf dem Server ab. Kein eigenes `:timeout`:
die Frist kommt von Sandbox oder `parfunc`.

## L. Kleinkram (20261118, 0.36.2)

### L1. `load` einer leeren Datei lieferte rohes Go-nil  [FIXED 0.36.2]
**Symptom:** `(handler-case (load "leer.lisp") (error (e) …))` →
`panic recovered: … nil pointer dereference in (pair? RAW__1)`;
`(list (load "leer.lisp"))` → `list: Argument 1 ist nil`. Ebenso Dateien
nur mit Leerraum oder Kommentar.
**Fix:** `load` liefert wie bisher den Wert der letzten Form, bei leerer
Datei `()` (wie `sandbox-eval` mit leerem String).

### L2. KI-Referenz: „`3.0` ist `integer`"  [FIXED 0.36.2, nur Doku]
**Symptom:** `ki-referenz.md` (Abschnitt Typen), `referenz_cn.md` und das
Tutorial (`type-of`) beschrieben noch das Float-only-Modell vor 0.31.
**Fix:** `3.0` ist `float`: `(typep 3.0 'integer)` → `()`.

### L3. `(-)` ohne Argumente lieferte `0`  [FIXED 0.36.2]
**Symptom:** `(-)` → `0`; Common Lisp verlangt mindestens ein Argument.
Ebenso `(apply - '())`.
**Fix:** `(-)` → Fehler `-: min. 1 Argument`. `(- x)` negiert weiter,
`(+)` → `0` und `(*)` → `1` bleiben (wie CL). `(/)` war schon ein Fehler
(`/: min. 2 Argumente`).

---

*Pflegehinweis:* Neue Funde unten anfügen (Datum + Kapitel + Status).
Quelle der Wahrheit für die 8 Kern-Fallstricke bleibt CLAUDE.md.
