# GoLisp2 – golisp2web aus Emacs/SLIME steuern

Ein laufender `golisp2 --swank`-Prozess ist derselbe Prozess, der auch
`webserv`/`http-serve` bedient — SWANK bringt kein eigenes, eingeschränktes
Environment mit (`lib.BaseEnv()` + `LoadStdlib`, identisch zum CLI, siehe
`lib/swank/server.go`). Alles, was im REPL geht, geht auch aus Emacs: die
Web-Bridge starten, golisp2web (der PySide6-GUI-Client, eigenes Repo
`golisp2web/`) als externen Prozess anstoßen, live gegen ihn entwickeln,
ihn wieder fernbeenden. Für die SWANK-Verbindung selbst siehe `docs/swank.md`
— hier geht es nur um das, was man **danach** im REPL damit anstellen kann.

## Voraussetzung

```bash
./build/golisp2 --swank 127.0.0.1:4242
```

In Emacs:

```
M-x slime-connect → Host: 127.0.0.1 → Port: 4242
```

Der `*slime-repl USER*`-Puffer öffnet sich — ab hier laufen alle Beispiele
unten als ganz normale Formen darin (`C-x C-e` bzw. Eingabe im REPL).

## Beispiel 1 — golisp2web interaktiv aus dem REPL starten

```lisp
(define s (webserv :port 8090
                    :html "<html><head></head><body><h1>Hallo aus Emacs</h1></body></html>"
                    :open nil))
(system "cd golisp2web && python3 golisp2web.py -t1 localhost:8090 &")
```

`system` läuft über `/bin/sh -c` und blockiert bis das Kommando fertig ist
— das `&` am Ende schickt den Python-Prozess in den Hintergrund, `system`
kehrt sofort zurück und der REPL bleibt frei. golisp2web öffnet sich auf
dem Desktop, verbunden mit `s`. `ws-export`/`ws-emit` auf `s` funktionieren
jetzt genau wie sonst auch — nur dass du sie interaktiv aus Emacs tippst,
nicht aus einer Skript-Datei.

## Beispiel 2 — Live-Redefinition während golisp2web offen ist

Der eigentliche Reiz: golisp2web offen lassen und Handler **während der
Sitzung** neu definieren — kein Neustart von golisp2web nötig, der Browser-
Tab bleibt verbunden (Live-Image-Idee der Web-Bridge, siehe Kommentar in
`lib/wsbridge.go` zu `ws-export`).

```lisp
(ws-export s "echo" (lambda (c text) (string-append "Echo: " text)))
;; -- im golisp2web-Fenster: golisp.call("echo", "hallo").then(...) testen --

;; Handler ohne Neustart still ueberschreiben:
(ws-export s "echo" (lambda (c text) (string-upcase text)))
```

Beide `ws-export`-Aufrufe landen unter demselben Namen `"echo"` — der
zweite ersetzt den ersten, der Browser-Tab merkt davon nichts außer dem
neuen Verhalten beim nächsten Aufruf.

## Beispiel 3 — golisp2web programmatisch starten, prüfen, beenden

Für alles, was mehr als "einmal hinschauen" ist — automatisierte Checks,
Demo-Skripte — das `parfunc`-Muster aus `tests/golisp2web-test.lisp`
(dort ausführlich kommentiert, hier die Kurzform zum Copy-Paste):

```lisp
(defun wait-for-client (s tries)
  (if (<= tries 0)
      ()
      (if (ws-clients s) t (begin (sleep 500) (wait-for-client s (- tries 1))))))

(let ((s (webserv :port 0 :host "127.0.0.1"
                   :html "<html><head></head><body>Test</body></html>"
                   :open nil))
      (verbunden ()))
  (let ((cmd (format nil "cd golisp2web && python3 golisp2web.py -t1 localhost:~a"
                      (http-port s))))
    (parfunc ergebnis
      (http-wait s)
      (begin (system cmd) (http-stop s))
      (begin
        (set! verbunden (wait-for-client s 20))
        (if verbunden (ws-emit s "golisp2web-quit" ())))))
  verbunden)
```

`parfunc` ist Fork-Join: alle drei Zweige teilen sich denselben Frame-Env
(`set!` darauf ist ausdrücklich unterstützt und mit `-race` verifiziert,
siehe `lib/env.go`) und `parfunc` kehrt erst zurück, wenn alle drei fertig
sind. Zweig 2 startet golisp2web *blockierend* über `system` — solange,
bis Zweig 3 per `ws-emit ... "golisp2web-quit"` das Fernbeenden auslöst
(die Konvention aus `golisp2web/lib/quitBridge.py`, siehe `docs/
superpowers/specs/2026-08-08-webserv-design.md`) — und ruft danach
`http-stop`, was wiederum Zweig 1 (`http-wait`) entblockt. **Wichtig:**
`http-stop` nicht vergessen — ohne den Aufruf bleibt Zweig 1 für immer
blockiert und `parfunc` kehrt nie zurück (kein Timeout, kein Fehler,
einfach ein stiller Hänger — selbst erlebt beim Schreiben dieses
Beispiels). Damit läuft der ganze Zyklus — öffnen, verbinden, prüfen,
schließen — ohne einen einzigen Klick.

## Warum das ohne Sonderfall funktioniert

- SWANK-Env == CLI-Env: kein separates, abgespecktes Environment für
  Emacs-Verbindungen.
- `system`/`exec` sind normale Primitiven (`lib/shellcmd.go`,
  `lib/eval_exec.go`) — golisp2web ist für golisp2 einfach ein externer
  Prozess, kein Sonderweg.
- Die Web-Bridge (`ws-export`/`ws-emit`/`ws-clients`) kennt ihre Aufrufer
  nicht — ob die aus einer geladenen Datei, einem `-e`-Aufruf oder einem
  SLIME-REPL kommen, ist ihr egal.

## Grenzen

- golisp2web bindet `QT_QPA_PLATFORM` fest auf `xcb` — braucht ein echtes
  X11-Display (`$DISPLAY` muss gesetzt sein). Kein Headless-Betrieb.
- `system "cd golisp2web && ..."` setzt voraus, dass der `golisp2`-Prozess
  im Repo-Root läuft (`golisp2web/` als Unterordner erreichbar). Läuft der
  Prozess anderswo, absoluten Pfad verwenden.
- golisp2web ist ein **eigenes Git-Repo** (`golisp2web/`), kein Submodul —
  Änderungen dort werden separat committet.
- **Nur eine golisp2web-Instanz gleichzeitig starten.** Läuft bereits eine
  (auch eine bereits geschlossene, aber noch nicht sauber beendete —
  `ps aux | grep golisp2web.py` prüfen), hängt eine zweite beim Start
  reproduzierbar fest: der `system`-Aufruf in Beispiel 3 kehrt dann nie
  zurück, `http-stop` wird nie erreicht, `parfunc` wartet ewig (kein
  Timeout, kein Fehler). Vermutliche Ursache: beide Instanzen teilen sich
  das gleiche Chromium/QtWebEngine-Profilverzeichnis auf der Platte
  (Web-Tabs nutzen bewusst das geteilte Default-Profil, siehe
  `golisp2web/lib/mainWindow.py` — anders als epub-Tabs, die je ein
  eigenes off-the-record-Profil bekommen) — noch nicht abschließend
  verifiziert, aber beim Schreiben dieses Dokuments reproduzierbar
  beobachtet und durch Beenden der hängenden Instanz behoben. Vor
  automatisierten Läufen (Beispiel 3, `tests/golisp2web-test.lisp`) immer
  prüfen, dass keine golisp2web-Instanz mehr läuft.
