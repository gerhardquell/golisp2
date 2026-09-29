## Global Constraints

- Repo `/u/lisp-projekte/golisp2`. Befehle vom Repo-Root aus.
- **Neue Dateien:** Einrückung 2 Spaces, keine Tabs. **Bestehende Dateien:** Stil der Umgebung übernehmen — `src/lib/primitives.go` ist mit Tabs eingerückt, dort also Tabs. **Kein `gofmt`** laufen lassen.
- Kommentare deutsch, sparsam.
- Fehlermeldungen Go: `fmt.Errorf("funktionsname: beschreibung")`.
- Datei-Header nur für neu angelegte Dateien: Autor `Gerhard Quell - gquell@skequell.de`, CoAutor = **dein eigenes, exaktes Modell** (nicht aus diesem Plan abschreiben), Copyright `2026 Gerhard Quell - SKEQuell`, Erstellt `20260929`. Bestehende Header nie ändern.
- Commit-Trailer: das schreibende Modell trägt sich **selbst** ein, exakte Bezeichnung, z. B. `Co-Authored-By: <dein Modell> <noreply@anthropic.com>`. Nicht raten, nicht aus diesem Plan übernehmen.
- Temporäres nach `./tmp/`, **nie** `/tmp`.
- Primitiven existieren nur, wenn `BaseEnv()` (`src/lib/primitives.go`) sie registriert.
- Keine zweite Typ-Tabelle: Typnamen und Hierarchie stehen **nur** in `*type-parents*` (`types.lisp`).
- Keine Datei über 1000 Zeilen.
- Vorhandene Prädikate (`number?`, `punkt?` usw.) werden **nicht** verändert.
- Achtung stdlib-`append`: `(append list item)` hängt **ein** Element an, ist nicht CL-`append`. Nicht zum Verketten benutzen.

---
