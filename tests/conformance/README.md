# Konformitäts-Suite — golisp2 gegen clisp

**Erstellt:** 20260723 · **Zweck:** Schritt 2 des CL-Konformitätsplans (TODO.md).
Charakterisierungstests: clisp ist Goldstandard, golisp2 wird dagegen gemessen.
**Kein Produktivcode wurde für diese Suite geändert** — sie ist das rote
Netz, gegen das Schritt 3 (Kern-Ausbau) arbeitet.

## Lauf

```bash
tests/conformance/run.sh          # Suite (Exit 0 = PASS oder nur erwartetes Rot)
tests/conformance/run.sh --gold   # Gold via clisp neu erzeugen — bewusste Aktion!
```

## Status-Bedeutung

| Status | Bedeutung |
|--------|-----------|
| PASS | golisp2 verhält sich wie clisp |
| FAIL | Abweichung, **nicht** akzeptiert — Bug oder undokumentierte Design-Entscheidung |
| XFAIL | Abweichung, akzeptiert (Feature fehlt, steht im Plan) — gelistet in `known-failures.txt` |
| XPASS | plötzlich konform, aber noch in `known-failures.txt` → Eintrag löschen, Suite feiert |

## Struktur

- `cases/NN-*.lisp` — ein Fall pro Zeile, Kommentare mit `;;`.
  Fälle einer Datei laufen in **einem** Prozess sequentiell (Zustand bleibt,
  `defun` vor Verwendung). Keine Mehrzeiler, keine case-sensitiven Strings.
- `gold/NN-*.gold` — clisp-Ergebnis pro Zeile (`write-to-string`, downcased;
  `ERROR` bei Condition). **Versioniert**, nie implizit neu erzeugen.
- `driver-clisp.lisp` — Gold-Generator (ein clisp-Prozess pro Datei).
- `known-failures.txt` — exakte Form-Texte der akzeptierten Lücken.
- `run.sh` — Runner. Normalisiert beiderseits: `()` ≡ `nil` (Drucker-
  Konvention), `#<…>` ≡ `#<>` (unreadable Objects).

## Regeln für neue Fälle

1. Fall zuerst in clisp verifizieren (`--gold`), dann gold diffen — Gold darf
   nie blind übernommen werden.
2. Ein Fall = eine Zeile = ein Prozess-unabhängiges Verhalten; Zustand nur
   über Reihenfolge innerhalb einer Datei.
3. Neue fehlende Features: Fall schreiben → FAIL → in `known-failures.txt`
   → XFAIL. Implementierung landet → XPASS → Eintrag löschen.
4. Wert-Abweichungen (kein ERROR) gehören **nie** in `known-failures.txt` —
   die sind laute Bugs.

## Stand Erstlauf (20260723)

`PASS=72 FAIL=18 XFAIL=62`

Die 18 FAIL sind Semantik-Abweichungen in **existierenden** Features
(u. a. `cond`-Einzelelement, `catch`-Semantik, `let`-Parallelität,
`flet`-Parallelität, `setq`-Rückgabe/Multi-Paar, `dolist`/`dotimes`-Resultat).
Details: `doc/cl-inventar.md`, Befundliste in TODO.md.
