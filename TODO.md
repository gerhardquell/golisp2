# TODO 20261011 - Fragen

## 1. Testen eval
Lass uns die eval-Funktion testen:

```
mul
"(+ 4 22)"
(eval mul)
"(+ 4 22)"
```
Eigentlich sollte eval die Addition ausführen.

## Offen — Entscheidungen (Gerhard)
- [ ] `swank--cell-type` (`src/lib/swank/env.go:116`) ist Beinahe-Zwilling
      von `cellTypeName` (`src/lib/celltype.go`). Zusammenlegen ändert die
      Labels von SLIME `describe-symbol` (`"atom"` → `symbol` …).
- [ ] Lispbuch: Remote anlegen? (Repo hat keinen — 6 Commits nur lokal)
- [ ] Lispbuch: `src/ch2/` und `src/chapters/` driften — Kap. 8/12/16/24/27
      in ch2 älter (ohne sort/sqrt-Fixes), Kap. 1 in ch2 neuer. Richtung?
- [ ] Lispbuch-`CLAUDE.md`: Sprachabschnitt veraltet („keine Hash-Maps",
      „KEIN CL-Condition-System").

## Offen — Fehler / Kleinkram
- [ ] `format ~e` / `~g` runden falsch: `(format nil "~e" 1.5)` → `"2e+00"`
      (CL: `"1.5e+0"`).
- [ ] Kommentar an `fnAppend` (`src/lib/primitives.go`) sagt „single-item",
      Tests sagen CL-`append` — Kommentar prüfen.
- [ ] `examples/golisp2web_demo.lisp`: Tabs aus der Umformatierung (2 Spaces).
- [ ] typep-Folgen (Spec-Nicht-Ziele): `deftype`, `check-type`, `subtypep`,
      `typecase`/`etypecase`.
- [ ] Aus 20260928 weiter offen: sigo Schritt 2 (Modell ohne Thinking,
      `zai-glm53-f`), Schritt 3 (`max_tokens`/`temperature`), Cache-Feldnamen
      Moonshot/Gemini/DeepSeek.
- [ ] webserv deferred-minor (20260809): `fnBrowserOpen` verschluckt Fehler,
      `/`-Kollision → Go-Panic, `webserv-smoke` nicht in `-t`.
