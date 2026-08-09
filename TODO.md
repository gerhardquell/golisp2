# Aufgabe: 20260808

**Status:** ERLEDIGT — 20260809

## 1. Programmende — ERLEDIGT

Wenn ich meinen Rechner mit shutdown herunterfahre, wird golisp2d/golisp2-swank nicht sofort beendet, sondern benötigt 1:30Min zum terminieren. Warum?

**Ursache:** `fnHTTPWait` (`lib/httpserver.go`) installiert bei blockierendem
`(http-wait srv)` per `signal.Notify` einen **prozessweiten** SIGTERM/SIGINT-
Handler. Der fing das Shutdown-Signal ab, gab aber nur den Lisp-Call frei,
ohne den Prozess zu beenden — der Daemon lief weiter, bis systemd nach
`TimeoutStopSec` (90s = "1:30") mit SIGKILL nachhalf.

**Fix:** `fnHTTPWait` ruft bei SIGINT/SIGTERM jetzt `os.Exit(0)` statt nur
den Call zu entblocken. Commit `f0ae077`. Live verifiziert: Prozess beendet
sich in ~4ms statt nach 90s.

## 2. Zusammenfassen von golisp2 und golisp2d — ERLEDIGT

Wir haben 3 Programme: golisp2, golisp2d, golisp2-client
Können wir golisp2 und golisp2d zusammenfassen? Wie ließe sich das
machen?

**Befund:** `cmd/golisp2d` war ein 1:1-Wrapper um `swank.RunServer` —
`golisp2 --swank host:port` (in `main.go`) tat exakt dasselbe.

**Umsetzung:** `cmd/golisp2d/` gelöscht, `build.sh` baut nur noch `golisp2`
+ `golisp2-client`, Doku/Kommentare korrigiert. Commit `ea93974`.
Deployment: `golisp2d.service` (Port 9123) gestoppt, disabled, Unit-File
und Binary entfernt. `golisp2-swank.service` (Port 4242, `golisp2 --swank`)
ist jetzt der einzige SWANK-Daemon — läuft, verifiziert per SLIME-Connect
und `(sigo "test")`.
