# TODO - Aufgabenplanung 20260821

## exec — ERLEDIGT (20260821)
- `env:` Keyword ergänzt (`lib/eval_exec.go`), mehrfach angebbar, Form
  `"KEY=WERT"`. Erbt weiterhin `os.Environ()`, ergänzt nur zusätzliche Vars.
  Tests: `TestExecEnv`, `TestExecEnvInvalidFormat`.
- Mehrere Parameter gingen bereits vorher — `param:` einfach mehrfach angeben,
  jedes Vorkommen ist EIN argv-Eintrag:
  `(exec "prog" param: "-st" param: "heard attac" stdout: out)`.
  Der Fehler im Ursprungsbeispiel kam vom manuellen Quoten in einem
  einzelnen `param:`-String — `exec` läuft ohne Shell, Anführungszeichen
  werden nicht interpretiert, sondern landen wörtlich im Argument.



## golisp2web - epub3-anpassung — ERLEDIGT (20260821)

Kapitelnavigation (Erstes/Vorheriges/Nächstes/Letztes Kapitel) als 4 Buttons
in der Haupt-Toolbar, `golisp2web/lib/mainWindow.py`. Buttons disabled, wenn
aktiver Tab kein epub-Tab ist oder eine Kapitelgrenze erreicht ist. `chapters`
(aus `parseSpine()`) + aktueller Index leben jetzt am Browser-Objekt
(`g2wChapters`/`g2wChapterIdx`), Sprung lädt `epub://book/{chapters[idx]}`
neu. Syntax geprüft (`py_compile`) — GUI-Test selbst nicht möglich (keine
TTY/Display in dieser Session).


