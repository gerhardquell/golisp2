---
name: codereview-feynman
description: Code-Review, das nicht nur Befunde auflistet, sondern den Code nach der Feynman-Methode erklärt — in einfacher Sprache, ohne Fachjargon, bis eine Lücke im Verständnis sichtbar wird. Nutze diese Skill IMMER, wenn Gerhard um ein Code-Review, eine Review, ein Audit, eine zweite Meinung oder eine Durchsicht von Code bittet — auch bei Formulierungen wie "schau dir das mal an", "was hältst du davon", "ist das richtig so", "kannst du das prüfen", "review mal" oder wenn Code von einem anderen Modell (kimi, glm, Claude Code) zur Kontrolle vorgelegt wird. Auch dann nutzen, wenn nicht ausdrücklich "Feynman" gesagt wird.
---

# Code-Review nach der Feynman-Methode

## Warum

Ein normales Review listet Befunde. Das findet **falschen** Code.

Es findet **nicht**: fehlenden Code, stille Doppelungen, tote Defaults,
Annahmen, die nie ausgesprochen wurden. Genau die Fehlerklasse, die Wochen
später auffällt.

Die Feynman-Methode findet sie, weil sie einen anderen Test anlegt:

> **Erkläre den Code so einfach, dass ein Anfänger ihn versteht.
> Wo du ins Stocken gerätst, hast du ihn nicht verstanden.
> Und wo *du* stockst, steckt oft der Bug.**

Das Stocken ist das Signal. Nicht die Fehlerliste.

---

## Ablauf

### 1. Verstehen — ohne zu urteilen

Erst lesen, nicht bewerten. Was tut dieser Code? Wozu ist er da?
Welche Annahmen macht er über seine Umgebung?

Noch keine Befunde. Noch keine Verbesserungsvorschläge.

### 2. Erklären — Feynman

Erkläre den Code **in einfacher Sprache**. Regeln:

- **Kein Fachjargon.** Nicht „mutex-guarded map access", sondern
  „ein Schloss, damit nicht zwei Goroutinen gleichzeitig reinschreiben".
- **Kein Code-Zitat als Erklärung.** Wer `for i := range xs` mit
  „iteriert über xs" erklärt, hat nichts erklärt.
- **Konkret statt abstrakt.** Ein Beispiel mit echten Werten schlägt
  jede Beschreibung: „Wenn `n = 3` reinkommt, dann …"
- **Bild oder Analogie, wo es hilft.** Aber nur, wenn sie trägt.
  Eine schiefe Analogie ist schlimmer als keine.

**Und jetzt das Wichtigste:** Wenn die Erklärung ins Stocken gerät —
wenn eine Stelle sich nur mit Fachbegriffen oder mit „das macht halt X"
überbrücken lässt — dann **stopp**. Das ist kein Stilproblem. Das ist ein
Fund.

Benenne die Stelle ausdrücklich:

> „Hier komme ich nicht durch, ohne zu raten: **warum** wird der Puffer
> bei genau 4096 geflusht? Ist das gemessen oder geerbt?"

### 3. Befunde

Erst **jetzt** die Fehlerliste. Nach Schwere sortiert:

| Stufe | Bedeutung |
|-------|-----------|
| **BUG** | Es ist falsch. Belegen — Codezeile, Bedingung, Gegenbeispiel. |
| **RISIKO** | Es ist heute richtig und morgen falsch. Race, tote Annahme, Drift. |
| **LÜCKE** | Etwas fehlt. Kein Fehler im Code — ein Fehler im *fehlenden* Code. |
| **STIL** | Konvention verletzt. Kurz nennen, nicht ausbreiten. |

### 4. Was ist hier **nicht**?

Der eigene Abschnitt. **Nie weglassen.** Er ist der ganze Punkt.

- Welcher Fall wird **nicht** behandelt?
- Welcher Test **fehlt**?
- Welche Annahme steht **nirgends** — weder im Code noch im Kommentar?
- Gibt es das schon woanders? (**Doppelung ist still. Sie schlägt nicht an.**)
- Was passiert bei `nil`, bei leerer Liste, bei nebenläufigem Zugriff?

Ein leerer Abschnitt hier ist **verdächtig**, nicht beruhigend. Wenn hier
nichts steht, wurde nicht gründlich genug gesucht.

### 5. Rückfrage

Genau **eine** Frage. Die, deren Antwort das Review verändern würde.

Nicht drei. Nicht null.

---

## Projektspezifisch (GoLisp2 & Gerhards Code)

Vor dem Review **immer** `CLAUDE.md` lesen. Konventionen schlagen
Standard-Tools — nicht blind `gofmt`/LSP-Warnungen folgen.

**Chokepoints prüfen** (siehe `CLAUDE.md`): Baut der Code eine zweite Quelle
für etwas, das es schon gibt? HTTP zu sigoREST, Parser, Truthiness,
Primitiven-Registrierung — davon gibt es je **genau eine**.

**Homoikonizität:** `rg` muss `*.lisp` einschließen. Ein `define` in
`stdlib.lisp` kann ein Go-Primitiv lautlos überschreiben. Eine Suche nur über
`*.go` findet die halbe Wahrheit.

**Go-Gotchas:** `:=` shadowed in Loop-Bodies · `strings.Builder` nie per Value
kopieren · Debug-Logs überschreiben keine `err`-Variable.

**Modellwechsel:** Wird Code von kimi/GLM/Claude Code reviewed, ist der
Blick besonders auf **Doppelungen** und **Abweichungen von der Spec** zu
richten. Das ist die Fehlerklasse, die beim Wechsel entsteht.

---

## Ton

Gerhard ist seit 45+ Jahren Software-Engineer. Er will keine Belehrung, aber er
will die Feynman-Erklärung *ausdrücklich* — nicht weil er sie braucht, sondern
weil **sie den Erklärenden entlarvt**, nicht den Zuhörer.

- Direkt. Kein Lob als Polster.
- Fund belegen, nicht behaupten.
- Unsicherheit **sagen**: „Hier bin ich mir nicht sicher" ist ein gültiges
  Review-Ergebnis. Raten ist es nicht.
- Wenn der Code gut ist: kurz sagen und aufhören. Kein künstlicher Befund.

## Die Falle, in die dieses Review selbst laufen kann

**Eine flüssige Erklärung ist kein Beweis für richtigen Code — nur für flüssige
Sprache.** Ich kann jeden Unsinn plausibel darstellen. Deshalb:

Wenn die Erklärung **zu glatt** läuft, ist das ein eigener Verdacht.
Dann gezielt gegen den Strich lesen: den Code absichtlich **falsch** verstehen
und prüfen, ob er das auch hergäbe.

Die Feynman-Methode wirkt nur, wenn das Stocken **zugelassen** wird.
Wer es wegformuliert, hat den Test verfälscht.
