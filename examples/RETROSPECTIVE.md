# examples/ – Retrospektive

**Datum:** 2026-08-26
**Autoren:** Gerhard Quell & Claude Sonnet 5

---

# Session 1 – Control-Panel-Demo, parvmira-Review, exec-Timeout

## Was haben wir gebaut?

| Feature | Dateien | Beschreibung |
|---------|---------|--------------|
| Control-Panel-Demo | `examples/golisp2web_demo.lisp` (neu) | Textinput+Button, Tabelle, neuer Tab, Fernbeenden — je ein `ws-export`/`ws-emit`-Baustein |
| `exec timeout:` | `lib/eval_exec.go`, `lib/eval_exec_test.go` | Neues Keyword, `-1` = unendlich, sonst überschreibbares Sekunden-Limit statt fest 60s |
| parvmira-Review-Fixes | `examples/parvmira-web/server.lisp` | `timeout: -1` ergänzt, fehlendes `begin` in parfunc-Branch 2 eingefügt |
| Shebang + exec-Start | `examples/golisp2web_demo.lisp` | Von `system "... &"` auf `exec` mit lokaler Pfad-Selbstlokalisierung umgestellt |

**Vorher:** `exec` killt jeden Kindprozess nach fest 60s, keine Möglichkeit das zu ändern. Demo-Skript ohne Shebang, startete golisp2web nicht-blockierend über die Shell.
**Nachher:** `exec` konfigurierbar (inkl. unendlich), beide Demo-Skripte laufen darüber, direkt ausführbar per Shebang.

---

## Was lief gut?

### Code-Review eines fremden Skripts fand einen echten Sprach-Bug
Beim Durchsehen von `parvmira-web/server.lisp` (von Gerhard kopiert, nicht von uns geschrieben)
fielen zwei Dinge auf, die nicht sofort als Fehler sichtbar sind: `exec`s hartes 60s-Timeout
und ein fehlendes `begin` im zweiten `parfunc`-Branch. Beides lief "zufällig" durch — bewiesen
erst durch Nachlesen im Reader/Eval-Code (`evalParfunc`, `evalExec`), nicht durch Bauchgefühl.

### Empirische Verifikation statt Vertrauen aufs Codelesen
Gerhard widersprach der Timeout-Analyse ("lief mehrere Minuten"). Statt die Aussage einfach zu
übernehmen oder stur bei der Code-Lesart zu bleiben, wurde ein Reproduktionsscript gebaut
(`exec "sleep" param: "75" ...`) — klar bestätigt: nach 60s tot, `exitcd: -1`. Die scheinbare
Diskrepanz klärte sich strukturell: `parfunc`-Branches sind unabhängig, der Server-Branch lief
weiter, nur das GUI-Fenster starb lautlos. Beide Seiten hatten recht, nur über unterschiedliche
Teile des Systems.

### TDD für `exec timeout:`
Fünf Tests zuerst (Override, Infinite, zwei Invalid-Fälle, Not-a-Number), rot bestätigt
(`unbekanntes Keyword timeout:`), dann implementiert, grün. Kein Rätselraten, ob die
`-1`-Semantik wirklich "kein Deadline" bedeutet — `context.Background()` statt
`context.WithTimeout` beweist das strukturell, kein 61s-Test nötig.

---

## Was nicht lief / Verbesserungspotenzial

### Eigener Pfad-Trick brach beim ersten echten Test
Die argv-basierte Selbstlokalisierung (`eltern-verzeichnis` zweifach geschachtelt, um von
`examples/skript.lisp` zur Repo-Root hochzugehen) war nur für *eine* Aufrufform getestet
(`./examples/golisp2web_demo.lisp` von der Repo-Root aus). Gerhards natürlicher erster Versuch —
`cd examples && ./golisp2web_demo.lisp` — hatte nur ein Pfadsegment in `argv`, die doppelte
Verschachtelung brach auf `"."` zusammen. Kein Fehler, keine Exception: `exec` startete
`python3` gegen einen nicht existierenden Pfad, der Traceback landete nirgendwo (kein
`stderr:`/`exitcd:` erfasst), das Skript hing einfach in `http-wait`.

**Lehre:** Bei Pfad-Arithmetik aus `argv` nicht String-Segmente zählen und selbst wegschneiden —
ein wörtliches `".."` anhängen und dem Betriebssystem die Auflösung überlassen ist robust gegen
jede Aufrufform (Repo-Root, Unterverzeichnis, absoluter Pfad), weil der Kernel `..` unabhängig
vom eigenen String-Parsing aufhält.

### Silent Failures sind teuer, weil sie nichts sagen
Zwei von drei Bugs dieser Session (fehlendes `begin`, kaputter Pfad) waren *lautlos*: keine
Fehlermeldung, kein Absturz — nur falsches oder ausbleibendes Verhalten. `evalParfunc` schluckt
Branch-Fehler zu `nil`; `exec` ohne `exitcd:`/`stderr:` gibt bei einem gecrashten Kindprozess
trotzdem `t` zurück. Beide Chokepoints sind by design fehlertolerant (ein Branch soll den
anderen nicht reißen) — aber genau das macht Debugging ohne explizites Error-Capturing schwer.

**Lehre:** Bei `exec`-Aufrufen in Demo-/Betriebsskripten großzügig `stderr:`/`exitcd:` mitgeben,
zumindest während der Entwicklung — sonst verschwindet der eigentliche Fehler.

---

## Technische Erkenntnisse

### `exec`s Timeout kennt keine Prozess-Semantik
`context.WithTimeout` in `evalExec` ist ein reiner Wanduhr-Deckel — er weiß nichts über
Verbindungsaufbau, Sichtbarkeit eines Fensters oder sonstige Zustände des Kindprozesses. Jede
Annahme "der Timeout gilt nur für die Startphase" ist falsch; er gilt für die gesamte
Prozesslaufzeit, ausnahmslos.

### `parfunc`-Branches sind wirklich unabhängig
Ein sterbender Branch (GUI-Prozess nach 60s gekillt) hat keinerlei Einfluss auf die anderen
Branches (Server-Loop lief unbeeindruckt weiter). Das ist bei paralleler Fehlersuche wichtig:
"das Skript läuft noch" beweist nicht, dass *alle* Branches noch tun, was sie sollen.

### Reader-Check statt Vollausführung als Verifikationsstufe
Ohne X11-Display in dieser Umgebung ließ sich keines der beiden Demo-Skripte komplett
ausführen. `lib.ReadAll` direkt aus Go aufzurufen (kein `golisp2 -e`, kein `load`) prüft
Syntax ohne Seiteneffekte auszulösen — brauchbare Zwischenstufe zwischen "sieht richtig aus"
und "läuft wirklich", wenn echtes End-to-End-Testen nicht möglich ist.

---

## Offene Punkte

- [ ] `examples/parvmira-web/server.lisp` ist weiterhin WIP (Gerhards Angabe), nicht committet —
  hängt an externen Dateien (`/u/golisp2-projekte/nexora/...`), die außerhalb dieses Repos liegen.
- [ ] `/usr/local/lib/golisp2web` (installierte Version) ist veraltet gegenüber diesem Checkout —
  kein Sync-Mechanismus vorhanden. Betrifft jeden zukünftigen `exec "golisp2web" ...`-Aufruf,
  der nicht explizit den lokalen Pfad nutzt.

---

## Fazit Session 1

Eine Review-Session, kein Feature-Auftrag — und trotzdem ist ein echtes Sprach-Feature
(`exec timeout:`) dabei herausgekommen, weil ein fremdes Beispielskript genau die Lücke
sichtbar machte. Zwei der drei gefundenen Bugs liefen lautlos durch, bis sie durch gezieltes
Nachfragen und Reproduzieren (nicht durch bloßes Codelesen) aufgedeckt wurden.

> "Ein Beispielskript, das nicht ganz funktioniert, ist oft die beste Spezifikation für das,
> was der Sprache noch fehlt."
> — Gerhard & Claude, August 2026
