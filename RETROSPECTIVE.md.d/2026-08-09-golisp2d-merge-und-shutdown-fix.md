# Retrospektive: Shutdown-Fix (http-wait) & golisp2/golisp2d-Merge

**Datum:** 9. August 2026
**Autor:** Gerhard Quell & claude-sonnet-5
**Feature:** Root-Cause-Fix für die 90s-Shutdown-Verzögerung des SWANK-Daemons,
anschließend Zusammenlegung von `golisp2` und `golisp2d` zu einem Binary

---

## Was wurde gebaut?

- **Diagnose:** `shutdown`/`systemctl stop` auf `golisp2d`/`golisp2 -swank`
  brauchte konstant 1:30 Min statt sofort zu terminieren.
- **Root Cause gefunden:** `fnHTTPWait` (`lib/httpserver.go`) installiert bei
  blockierendem `(http-wait srv)` per `signal.Notify` einen **prozessweiten**
  SIGTERM/SIGINT-Handler. Bei Shutdown fing der das Signal ab, gab aber nur
  den Lisp-Call frei — der Daemon lief weiter, bis systemd nach
  `TimeoutStopSec` (Default 90s) mit SIGKILL nachhalf.
- **Fix:** `fnHTTPWait` ruft bei Signal-Empfang jetzt `os.Exit(0)` statt nur
  zu entblocken. Live verifiziert per `kill -TERM` + Zeitmessung:
  90s → ~4ms. Commit `f0ae077`.
- **Folgefrage vom Nutzer:** „Können wir `golisp2` und `golisp2d`
  zusammenlegen?" — Befund: `cmd/golisp2d` war ein 43-Zeilen-Wrapper um
  `swank.RunServer`, den `golisp2 --swank host:port` in `main.go` bereits
  identisch bereitstellte. Beide liefen zusätzlich parallel als eigene
  systemd-Services auf unterschiedlichen Ports (4242 und 9123).
- **Merge:** `cmd/golisp2d/` gelöscht, `build.sh` baut nur noch `golisp2` +
  `golisp2-client`, betroffener Integrationstest
  (`lib/swank/gps_bug_test.go`, startete den Server bisher per
  `exec.Command("golisp2d", ...)`) umgestellt, Doku (`CLAUDE.md`,
  `doc/struktur.md`, `doc/swank.md`, `doc/sigo.md`, `README*.md`,
  `BESCHREIBUNG.md`, `chinese/ABOUT*.md`) korrigiert. Commit `ea93974`.
- **Deployment:** neues Binary in `/usr/local/sbin/golisp2` ausgerollt,
  `golisp2-swank.service` (Port 4242) neu gestartet, `golisp2d.service`
  (Port 9123) gestoppt/disabled/entfernt — Unit-File und Binary weg, Port
  frei. Verifiziert per SLIME-Connect + `(sigo "test")`.
- `TODO.md` mit beiden erledigten Punkten aktualisiert. Commit `4e82579`.

---

## Was lief schief? (⚫ Schwarz)

| Problem | Ursache | Auswirkung |
|---------|---------|------------|
| Ein `rg`-Aufruf über die RTK-Bash-Hook zeigte `http-wait` durchgängig als `ln` an (Registrierung, Fehlermeldungen, README-Treffer) | RTK-Token-Kompression der Bash-Hook verfälschte die Anzeige, nicht die echte Datei | Hätte zu einer falschen Fix-Grundlage führen können; nur durch Gegencheck via `Read` (ungefiltert) bemerkt |
| Weder passwortloses `sudo` noch TTY über den Bash-Tool-Kanal verfügbar | Sandbox-Umgebung, `sudo` verlangt interaktives Passwort | Alle Service-Restarts/-Retires mussten der Nutzer manuell im echten Terminal ausführen — mehrere Hin-und-her-Runden statt eines durchgehenden Flows |
| `golisp2-swank.service` ging nach dem `golisp2d`-Retire unerwartet down (SIGTERM, `code=killed`, kein Crash) | Ursache nicht abschließend geklärt — kein `journalctl`-Zugriff (nicht in `adm`/`systemd-journal`), keine zeitliche Korrelation zu den gegebenen Retire-Befehlen erkennbar | Zusätzliche Verifikations- und Neustart-Runde nötig, bevor der Merge als abgeschlossen gelten konnte |
| Erste `AskUserQuestion` zu den Merge-Optionen wurde abgelehnt | Nutzer wollte erst frei nachfragen (Rückfrage zu `golisp2-client`/Port 9123), nicht sofort aus vorgegebenen Optionen wählen | Strukturierte Fragen mussten durch freie Konversation ersetzt werden — am Ende aber schneller geklärt als gedacht |

---

## Was haben wir gelernt? (🔵 Blau)

1. **Bash-Tool-Output ist nicht automatisch vertrauenswürdig, wenn Hooks
   dazwischenhängen.** Bei exakten Bezeichnernamen (Funktionsnamen,
   registrierte Primitiven) lieber `Read` als `rg`/`grep` über eine
   Bash-Hook-Pipeline — Read liefert unverändert das, was auf der Platte
   steht.
2. **`signal.Notify` in Go ist prozessweit, nicht scope-lokal** — auch wenn
   `defer signal.Stop(sigCh)` sauber aussieht, überschreibt es für die
   Dauer des Aufrufs die Default-Disposition für den **gesamten** Prozess.
   In einem geteilten Daemon (SWANK-Server mit mehreren Connections) kann
   ein einzelner blockierender Call so das Shutdown-Verhalten des ganzen
   Prozesses kapern.
3. **systemds `TimeoutStopSec` (Default 90s) ist ein guter Fingerabdruck.**
   Eine Shutdown-Verzögerung von exakt 1:30 Min ist fast immer "SIGTERM
   wurde nicht sauber verarbeitet, SIGKILL musste ran" — kein Zufall, kein
   Netzwerk-Timeout.
4. **Root-owned systemd-Services lassen sich aus einer Sandbox ohne TTY
   nicht restarten.** Frühzeitig erkennen und den Nutzer einbinden, statt
   es mehrfach zu versuchen.
5. **Ein dünner Wrapper-Binary, der nur eine bereits vorhandene Funktion
   erneut aufruft, ist die Deployment-Ebene derselben "stillen additiven
   Duplizierung", vor der `CLAUDE.md` im Code warnt.** `golisp2d` war
   codeseitig 100 % redundant zu `golisp2 --swank` — sichtbar wurde das
   aber erst, weil zwei separate systemd-Services parallel liefen.
6. **Der Blast-Radius-Check vor dem Löschen (`rg -ln "golisp2d"`) deckte
   eine echte funktionale Abhängigkeit auf** (`lib/swank/gps_bug_test.go`
   startete `golisp2d` direkt per `exec.Command`) — ohne den Check wäre
   dieser Test beim Merge kaputtgegangen.

---

## Action Items

| # | Aufgabe | Priorität |
|---|---------|-----------|
| 1 | `doc/CLAUDE.md` (stales 189-Zeilen-Duplikat der echten `CLAUDE.md`, fehlt u. a. Attribution/Chokepoints) aufräumen oder entfernen | Niedrig |
| 2 | Prüfen, ob weitere Primitiven in `lib/` prozessweite `os/signal`-Handler installieren, die ähnliche Shutdown-Fallen bergen könnten | Mittel |
| 3 | Journal-Lesezugriff (`adm`-Gruppe) klären, falls künftig Live-Diagnose an root-Services aus der Session nötig wird | Mittel |
| 4 | Falls sich der unerwartete Stop von `golisp2-swank.service` nach einem `daemon-reload` wiederholt: gezielt reproduzieren und Ursache klären | Niedrig |
