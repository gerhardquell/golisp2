### Task 5: Doku-Generator

**Files:**
- Create: `tools/gen-reference.lisp`
- Create: `docs/referenz-generiert.md` (generiertes Artefakt, committet)
- Modify: `docs/ki/referenz.md` (ersetzt bestehenden Inhalt komplett)
- Modify: `CLAUDE.md` (Doku-Tabelle: neuen Eintrag ergänzen)

**Interfaces:**
- Consumes: `(env-symbols)` aus Task 4, `(defined-in "pfad")` aus
  `src/lib/defloc.go` (bereits vorhanden, defsystem-lite), `(bound? sym)`
  (bereits vorhanden) zur Typklassifikation.
- Produces: zwei Markdown-Dateien, keine weiteren Konsumenten in diesem
  Plan.

- [ ] **Step 1: Generator-Skript schreiben**

`tools/gen-reference.lisp` — nutzt ausschließlich verifiziert vorhandene
Primitiven (`env-symbols` aus Task 4, `format`, `mapcar`, `apply`,
`string-append`, `file-write`, `length` — kein `open`/`sort`/`string<`,
die gibt es in diesem Repo nicht):

```lisp
;;**********************************************************************
;;  tools/gen-reference.lisp
;;  Autor    : Gerhard Quell - gquell@skequell.de
;;  CoAutor  : claude-sonnet-5
;;  Copyright: 2026 Gerhard Quell - SKEQuell
;;  Erstellt : 20260827
;;**********************************************************************
;; Generiert docs/referenz-generiert.md aus (env-symbols) -- Vollstaendigkeit
;; ist strukturell garantiert, da direkt aus dem Root-Env gelesen wird.
;; Aufruf: ./build/golisp2 tools/gen-reference.lisp
;;**********************************************************************

(define *ref-out* "docs/referenz-generiert.md")

(define (write-reference)
  (let* ((names (env-symbols))
         (rows (mapcar (lambda (n) (format nil "| `~a` | |~%" n)) names))
         (header (format nil "# GoLisp2 — Generierte Funktionsreferenz~%~%> Automatisch erzeugt aus `(env-symbols)` — nicht von Hand editieren.~%> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`~%~%| Symbol | Beschreibung |~%|---|---|~%"))
         (body (apply string-append rows)))
    (file-write *ref-out* (string-append header body))
    (format t "~a Symbole nach ~a geschrieben~%" (length names) *ref-out*)))

(write-reference)
```

`env-symbols` liefert bereits eine sortierte Liste (`sort.Strings` in
Task 4), deshalb kein zusätzlicher Sortierschritt nötig.

- [ ] **Step 2: Skript ausführen**

```bash
./build/golisp2 tools/gen-reference.lisp
```

Erwartung: `docs/referenz-generiert.md` wird erzeugt, Konsolenausgabe
nennt eine Symbolzahl deutlich über 100.

- [ ] **Step 3: docs/ki/referenz.md aktualisieren**

Bestehenden Inhalt vollständig ersetzen durch eine kompakte Fassung nach
dem bisherigen Aufbau (Eval-Reihenfolge, Spezialformen-Tabelle,
Primitiven-Kurzliste) — Spezialformen-Liste manuell aus
`src/lib/eval_core.go` (case-Zweige, siehe Datei-Header-Kommentar
"Eval-Reihenfolge in `evalList`") übernehmen, Primitiven-Kurzliste aus
`docs/referenz-generiert.md`. Datei-Header aktualisieren:

```
> **Quelle:** `eval_core.go`, `primitives.go`, `embed/stdlib.lisp`,
> generiert via `tools/gen-reference.lisp` (Stand 20260827).
```

- [ ] **Step 4: CLAUDE.md — Doku-Tabelle ergänzen**

Nach der Zeile mit `docs/memory.md` einfügen:

```
| `docs/referenz-generiert.md` | Vollständige Funktionsreferenz, generiert aus `(env-symbols)` | du eine konkrete Funktion nachschlägst |
```

- [ ] **Step 5: Verifizieren**

```bash
go build ./...
go test ./... -count=1
./build/golisp2 -t
wc -l docs/referenz-generiert.md
```

Erwartung: alles grün, generierte Datei hat mehr als 100 Zeilen.

- [ ] **Step 6: Commit**

```bash
git add tools/gen-reference.lisp docs/referenz-generiert.md docs/ki/referenz.md CLAUDE.md
git commit -m "$(cat <<'EOF'
docs(reference): Funktionsreferenz aus (env-symbols) generiert

Ersetzt die manuell gepflegte, bereits 4 Wochen alte docs/ki/referenz.md.
tools/gen-reference.lisp ist bei Bedarf erneut ausfuehrbar (kein
Build-Hook, YAGNI) -- Vollstaendigkeit ist strukturell garantiert, da
direkt aus dem Root-Env gelesen wird statt von Hand gepflegt.

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
EOF
)"
```

---

