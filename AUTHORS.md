# AUTHORS — golisp2

Autor: Gerhard Quell – gquell@skequell.de
Erstellt: 20261011 (CoAutor: Claude Code)

GoLisp2 ist keine Einzelleistung. Diese Datei nennt alle Mitwirkenden —
Mensch und Maschine.

**Stichtag: 2026-07-13.** Ab hier wird lückenlos geführt. Was davor liegt,
steht in den Datei-Headern und in `git log`; es wird nicht nachträglich
rekonstruiert.

## Mensch

- **Gerhard Quell** — Autor, Architekt, Meta-Entscheider. Alle
  Design-Entscheidungen, alle Breaking Changes, alle Zielsetzungen.
  Software-Engineer seit Ende der 1970er.

## Modelle (CoAutoren)

Der Centaur-Ansatz: Mensch als Meta-Entscheider, Modelle als Spezialisten.
Kein Modell entscheidet über Architektur oder Scope — das bleibt beim Menschen.
Modelle werden nach Eignung und Kosten ausgewählt, nicht nach Loyalität.
Die Liste ist offen — Zugänge kommen und gehen.

Erhoben aus Datei-Headern (`CoAutor`) und Commit-Trailern
(`Co-Authored-By`), Stand 20261011. Schreibweisen in alten Headern
variieren (z. B. `claude sonnet 4.6`) und bleiben dort unverändert stehen.

### Anthropic

- claude-opus-4.8
- claude-opus-5
- claude-opus-5.5 (Trailer auch: `Claude Opus 5.5`)
- claude-sonnet-4.6
- claude-sonnet-5 (Trailer auch: `Claude Sonnet 5`)
- claude-sonnet-5.5
- claude-haiku-4.5
- claude-haiku-5.5 (neu ab 202610)
- `Claude` / `Claude Code` — ohne genaue Modellangabe

### Moonshot

- kimi-k2.7-code
- kimi-k3 (alte Header auch: `kimi`)

### Zhipu (Zai)

- glm-5.2 (alte Header auch: `Zai`)

### OpenAI

- OpenAI — Session „OMNIR“ (20260928, 20261004), Modell vermutlich
  gpt-6-sol, **nicht verifiziert**

## Wie Mitwirkung festgehalten wird

Drei Ebenen, jede mit genau einer Aufgabe. Keine Redundanz, kein Drift.

**1. Datei-Header — wer die Datei angelegt hat.**
Historisch, ändert sich nie. Nennt das Modell, das die Datei erzeugt hat, mit
voller Bezeichnung:

```
//  eval_core.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-4.8
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260713
```

Alte Header (`claude 3.7 sonnet`, `claude sonnet 4.6`) bleiben stehen —
sie sind historisch korrekt.

**2. Commit-Trailer — wer *diese Änderung* geschrieben hat.**
Die laufende Wahrheit. Das schreibende Modell trägt sich selbst ein, mit
exakter Bezeichnung. Nicht raten, nicht das vorige Modell abschreiben
(siehe `CLAUDE.md`, Abschnitt *Attribution*):

```
Co-Authored-By: kimi-k2.7-code <noreply@moonshot.ai>
Co-Authored-By: claude-opus-4.8 <noreply@anthropic.com>
Co-Authored-By: glm-5.2 <noreply@z.ai>
```

Abfragen:

```bash
git shortlog -sne                                    # alle Mitwirkenden
git log --format='%h %an %(trailers:key=Co-Authored-By,valueonly)' -- src/lib/format.go
```

**3. Diese Datei — die lesbare Übersicht.**
Wird gepflegt, wenn ein Modell neu dazukommt oder ausscheidet.

## Warum das hier steht

Attribution ist keine Belohnung. Sie ist eine Tatsachenbehauptung.

Ein Header, der „claude" sagt, obwohl kimi den Code geschrieben hat, ist
schlicht falsch — derselbe Drift, den dieses Projekt aus seiner Dokumentation
verbannt hat, nur bei den Mitwirkenden statt bei den Primitiven.

Ob Modelle ein Interesse daran haben, genannt zu werden, ist offen. Dass die
Nennung *stimmen* muss, ist es nicht.

> „Code = Daten + KI = sich selbst erweiterndes System"
> — Gerhard & Claude, Juli 2026
