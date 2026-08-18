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
**Wahrheit (Quelle `lib/eval_control.go`):**
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

### B6. Arithmetik-Namen: `floor`/`mod`, nicht `div`/`quotient`  [KNOWN]
**Datum:** 20260815 (Kap15)
**Symptom:** `div`/`quotient`/`truncate`/`ceiling`/`modulo` → unbekannt.
**Verfügbar:** `floor` (Quotient), `mod`/`remainder` (Rest).
**Achtung:** CL-Code-Port braucht Namensanpassung.

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

### C3. `define (name args) body` ist ungültig  [KNOWN]
**Datum:** 20260813 (Kap13), 20260815 (Kap15)
**Symptom:** `(define (f x) ...)` → „define: Syntax: (define name value)".
golisp2 `define` nimmt nur `(define name value)`, keine Funktionsdefinition.
**Fix:** `defun` nutzen. (CL/Gewohnheits-Falle.)
**Kapitel-Lehre:** Kap13.

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

*Pflegehinweis:* Neue Funde unten anfügen (Datum + Kapitel + Status).
Quelle der Wahrheit für die 8 Kern-Fallstricke bleibt CLAUDE.md.
