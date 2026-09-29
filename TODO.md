# Aufgaben 20260928

## Ich möchte golisp2 /äquivalent zum CLisp) ergänzen:
- [x] **type** → als `type-of` (CL hat keine Funktion `type`)
- [x] **typep**
    → erledigt 20260929, gemergt + gepusht (Merge 77227c4).
      Spec `docs/superpowers/specs/2026-09-29-typep-design.md`,
      Plan `docs/superpowers/plans/2026-09-29-typep.md`.
      `%cell-type` (Go) + `src/embed/types.lisp`; Kombinatoren
      `or/and/not/member/eql/satisfies`, Bereiche `(integer lo hi)`,
      Structs via `*struct-types*`, Conditions über ihre Elternkette,
      WARN bei Namenskollision.

## Nebenbei erledigt 20260929
- [x] `ash` rundet beim Rechtsshift Richtung −∞ (`(ash -5 -1)` → -3)
- [x] `(- x)` negiert (`(- 5)` → -5)
- [x] `gcd` per Euklid über `mod` (`(gcd 12 18)` → 6)
- [x] `(funcall 'f …)` löst Symbole global auf
- [x] Lispbuch Kap. 11 (+Q&A) und Kap. 10 an golisp2-IST angeglichen,
      Prosa zeitlos, `src/ch2/` ins Lispbuch-Repo aufgenommen

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
