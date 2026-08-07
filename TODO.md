# Spec: GoLisp2 Web-Bridge (Stufe 1)

**Autor:** Gerhard Quell – gquell@skequell.de
**CoAutor:** Claude Opus 5
**Copyright:** 2026 Gerhard Quell – SKEQuell
**Erstellt:** 20260806
**Status:** ERLEDIGT — 20260807

---

## 0. ERLEDIGT — Ergebnis

Alle fünf Schritte aus Abschnitt 11 umgesetzt und committed:

1. `9cbfaae` `lib/jsoncell.go` — JSON ↔ Cell
2. `419046e` `lib/httpserver.go` — HTTP-Primitiven
3. `18a1c31` `lib/wsbridge.go` — WebSocket-Hub + RPC
4. `9bba481` `lib/embed/boot.js` + `public/index.html` + `tests/webbridge.lisp`
5. Szenarien 8.4 A und B — manuell verifiziert (Protokoll unten)

`go build ./...` sauber, `go test ./... -count=1` → 292 Tests grün.

**Testaufbau Schritt 5:** `golisp2d` + `golisp2-client -repl` (eine
durchgehende SWANK-Verbindung, damit `s` über mehrere Kommandos hinweg
gültig bleibt — jede neue Verbindung bekommt sonst ein frisches `BaseEnv()`,
siehe `lib/swank/server.go:49`). Browser-Seite: echtes Chromium via
Playwright gegen `http://127.0.0.1:<port>/`.

**Szenario A — Live-Redefinition:**
`(ws-export s "ping" (lambda (c) "eins"))` → Browser `golisp.call('ping')`
→ `"eins"`. Danach im laufenden Server `(ws-export s "ping" (lambda (c)
"zwei"))`, `(ws-clients s)` → `(1)` (Client-ID unverändert, kein Reconnect).
Erneuter Aufruf im selben Tab → `"zwei"`. Bestätigt.

**Szenario B — Reentranz:**
`(ws-export s "breite" (lambda (c) (ws-call s c "return window.innerWidth")))`,
Browser `golisp.call('breite')` → `1895` (tatsächliche Fensterbreite), kein
Deadlock, kein Timeout. Bestätigt.

Offene Punkte aus Abschnitt 12: siehe dort, jetzt beantwortet.

---

## 0. Ziel und Nicht-Ziel

**Ziel:** GoLisp2 wird zum lebenden Image, an das sich Browser-Clients
anklemmen — Swank-Modell. Bidirektionales RPC über WebSocket, statische
Dateien über HTTP. Nach Stufe 1 muss folgendes im REPL funktionieren:

```lisp
(define srv (http-serve 0))
(http-static srv "/" "./public")
(ws-export srv "ask" (lambda (client frage) (string-append "Echo: " frage)))
(browser-open (string-append "http://127.0.0.1:" (number->string (http-port srv))))
(ws-emit srv 'tick 42)          ; Push, im Browser sofort sichtbar
(http-wait srv)
```

Und: `ws-export` darf **im laufenden Betrieb** neu aufgerufen werden. Der
Client bleibt verbunden, das Verhalten ändert sich. Das ist der eigentliche
Punkt.

**Nicht-Ziel in Stufe 1:** Renderer, Panel-Typen, KaTeX, SVG, EPUB, Sessions,
Persistenz, Authentifizierung, TLS. Alles Stufe 2+.

---

## 1. Neue Dateien

| Datei | Inhalt | Richtwert |
|-------|--------|-----------|
| `lib/httpserver.go` | `http-serve`, `http-static`, `http-port`, `http-wait`, `http-stop`, `browser-open` | ~220 Z. |
| `lib/wsbridge.go` | `ws-export`, `ws-emit`, `ws-emit-to`, `ws-eval`, `ws-call`, `ws-clients`, Hub | ~260 Z. |
| `lib/jsoncell.go` | `CellToJSON`, `JSONToCell` | ~140 Z. |
| `lib/embed/boot.js` | Client-Bootstrap, via `//go:embed` | ~120 Z. |
| `lib/jsoncell_test.go` | Round-Trip-Tests | ~80 Z. |

Namensschema folgt `fileio.go` / `goroutine.go` / `sigorest.go`
(kleingeschrieben, ohne Trenner). Datei-Header wie in `CLAUDE.md`.

**Registrierung:** je eine `RegisterHTTPFuncs(env *Env)` und
`RegisterWSFuncs(env *Env)`, beide aus `BaseEnv()` aufgerufen.

**Abhängigkeit:** `github.com/gorilla/websocket v1.5.3`. Keine weitere.
RFC 6455 nicht selbst implementieren.

---

## 2. Datentyp: Server-Objekt

`http-serve` liefert eine `Cell{Type: LIST, Env: *WebServer}` — dasselbe
Muster wie `pg-connect`. `Val` wird auf `"#<webserver :PORT>"` gesetzt,
damit `println` etwas Brauchbares zeigt.

```go
type WebServer struct {
  mu       sync.RWMutex
  srv      *http.Server
  ln       net.Listener
  port     int
  handlers map[string]*Cell      // ws-export
  clients  map[int]*wsClient
  nextCid  int
  pending  map[int]chan *Cell    // ws-call, Key = call-id
  nextCall int
  env      *Env                  // Root-Env für Handler-Aufrufe
  done     chan struct{}
  idleAt   time.Time
}
```

Jede Primitive prüft zuerst, ob ihr erstes Argument ein gültiges
Server-Objekt ist → sonst `fmt.Errorf("http-serve: kein Server-Objekt")`.

---

## 3. Primitiven — exakte Signaturen

### 3.1 HTTP

| Form | Rückgabe | Verhalten |
|------|----------|-----------|
| `(http-serve port)` | Server-Cell | `port = 0` → freier Port vom OS. **Bindet ausschließlich an `127.0.0.1`.** Startet Goroutine, kehrt sofort zurück. Fehler beim Binden → Lisp-Error. |
| `(http-static srv urlpath dir)` | `t` | Mountet Verzeichnis unter `urlpath`. Mehrfach aufrufbar. `dir` relativ zum CWD. Kein Directory-Listing (leeres `index.html` → 404). |
| `(http-port srv)` | NUMBER | Tatsächlicher Port. |
| `(http-wait srv &key idle-exit)` | `nil` | Blockiert bis `http-stop`, SIGINT/SIGTERM, oder — wenn `idle-exit` (ms) gesetzt ist — bis so lange **kein** Client mehr verbunden war. |
| `(http-stop srv)` | `t` | Graceful Shutdown, 2 s Timeout, schließt alle WS-Verbindungen. Idempotent. |
| `(browser-open url)` | `t` / `nil` | Sucht der Reihe nach: `chromium`, `chromium-browser`, `google-chrome-stable`, `google-chrome`, `xdg-open`. Für die Chromium-Varianten: `--app=URL --user-data-dir=<mkdtemp>`. Prozess detached, kein Warten. `nil`, wenn nichts gefunden. |

**`idle-exit`-Semantik präzise:** Der Timer läuft nur, wenn `len(clients) == 0`.
Beim ersten Connect wird er zurückgesetzt. Startet der Server und verbindet sich
**nie** ein Client, läuft der Timer ab Serverstart — sonst hängt ein
Shebang-Skript ewig, wenn `browser-open` fehlschlägt.

### 3.2 WebSocket

| Form | Rückgabe | Verhalten |
|------|----------|-----------|
| `(ws-export srv name fn)` | `t` | `name` = STRING oder ATOM. `fn` = Lambda. Überschreibt still eine vorhandene Registrierung (Reload-Semantik wie `define-condition`). |
| `(ws-unexport srv name)` | `t` / `nil` | `nil`, wenn nicht registriert. |
| `(ws-emit srv event data)` | Anzahl Empfänger (NUMBER) | Broadcast an alle Clients. `event` = STRING oder ATOM. |
| `(ws-emit-to srv client event data)` | `t` / `nil` | `nil`, wenn Client-ID unbekannt. |
| `(ws-eval srv js)` | Anzahl Empfänger | Broadcast, feuern und vergessen, **keine** Rückgabe aus dem Browser. |
| `(ws-call srv client js &key timeout)` | Wert oder Lisp-Error | Blockiert. Default-Timeout **5000 ms**. Timeout → Lisp-Error `"ws-call: timeout nach 5000ms"`. |
| `(ws-clients srv)` | Liste von NUMBER | Aktuell verbundene Client-IDs. |

**Handler-Signatur:** Die exportierte Funktion wird **immer mit der
Client-ID als erstem Argument** aufgerufen, danach die Argumente aus dem
Request:

```lisp
(ws-export srv "ask" (lambda (client frage modell) ...))
;; Browser: golisp.call("ask", "Was ist Bewusstsein?", "claude-h")
;; Lisp:    (fn 3 "Was ist Bewusstsein?" "claude-h")
```

*Begründung:* GoLisp2 hat keine brauchbare dynamische Bindung (siehe
`referenz.md` 10.4), ein `(ws-client)` wäre also nur mit Tricks machbar.
Explizit ist hier besser als magisch.

Arity-Fehler (Browser schickt zu wenige/zu viele Argumente) → Fehler wird
als `err` zurückgeschickt, **Server läuft weiter**.

---

## 4. Drahtprotokoll

JSON, eine Nachricht pro WebSocket-Frame, Text-Frames (`TextMessage`).
Zwei getrennte ID-Räume: `id` für Browser→Lisp, `call` für Lisp→Browser.

### Browser → Lisp

```json
{"id": 7, "op": "ask", "args": ["Was ist Bewusstsein?", "claude-h"]}
{"call": 3, "ok": 1280}
{"call": 3, "err": "ReferenceError: foo is not defined"}
```

### Lisp → Browser

```json
{"id": 7, "ok": "Antwort..."}
{"id": 7, "err": "ask: unbekanntes Modell"}
{"event": "hut-fertig", "data": {"hut": "weiss", "text": "..."}}
{"call": 3, "js": "return window.innerWidth"}
```

**Regeln:**

- Unbekanntes `op` → `{"id":N,"err":"unbekannte Operation: xyz"}`. Kein Abbruch.
- Kaputtes JSON → Frame verwerfen, `warn` auf stderr, Verbindung **offen lassen**.
- `id`/`call` sind Integer, monoton, pro Verbindung bzw. pro Server.
- Der `js`-String wird im Browser als Body einer Funktion ausgeführt
  (`new Function(js)`), damit `return` funktioniert.

---

## 5. Nebenläufigkeit — der kritische Teil

**Jede eingehende `op`-Nachricht läuft in einer eigenen Goroutine.**
Kein globaler Eval-Mutex.

Begründung: Ein Handler, der `sigo` aufruft, blockiert sonst alle anderen
Clients für Sekunden. Sechs Hüte parallel wären unmöglich.

**Env-Regel:** Der Handler ist ein Lambda mit eigenem Closure-Env. Er wird
über denselben Pfad aufgerufen wie `funcall` — kein neues Root-Env, keine
Sonderbehandlung. Geteilter Zustand ist Sache des Lisp-Programmierers
(`lock-make`), genau wie bei `parfunc`.

**Schreibzugriff auf die WebSocket:** `gorilla/websocket` erlaubt **keinen
nebenläufigen Write**. Jeder Client bekommt deshalb eine eigene
Writer-Goroutine mit gepuffertem Channel (Kapazität 64). Alles, was raus
soll, geht in den Channel. Läuft der Channel voll → Nachricht verwerfen und
`warn` loggen; **nicht** blockieren, sonst hängt ein langsamer Client den
ganzen Server auf.

**`ws-call`-Deadlock-Falle:** Ruft ein Handler `ws-call` auf **denselben**
Client auf, von dem der Request kam, ist das erlaubt — weil Lesen und
Schreiben getrennte Goroutinen sind und der Handler ohnehin nebenläufig
läuft. Diese Zusage muss in einem Test abgesichert werden (siehe 8.4).

**Client-Abbruch während `ws-call`:** Verbindung weg → alle offenen
`pending`-Channels dieses Clients bekommen einen Fehlerwert, nicht Timeout.

---

## 6. JSON ↔ Cell

Das ist die Stelle mit den meisten stillen Fallen. Deshalb explizit.

### JSON → Cell

| JSON | Cell |
|------|------|
| `null` | Nil (Singleton) |
| `true` | `t` |
| `false` | Nil |
| Zahl | NUMBER (float64) |
| String | STRING |
| Array | LIST |
| Objekt | Alist: `(("k" . v) ("k2" . v2))`, Keys als **STRING** |

### Cell → JSON

| Cell | JSON |
|------|------|
| Nil | `null` |
| `t` | `true` |
| NUMBER | Zahl (Ganzzahlen ohne `.0` ausgeben) |
| STRING | String |
| ATOM (sonstiges) | String des Symbolnamens |
| LIST, alle Elemente dotted pairs mit STRING/ATOM-Car | Objekt |
| LIST sonst | Array |

**Die Objekt-Regel exakt:** Eine LIST wird genau dann zum JSON-Objekt, wenn
sie nicht leer ist **und** für *jedes* Element gilt: es ist ein Cons, dessen
`Car` STRING oder ATOM ist und dessen `Cdr` **kein** LIST ist. Sonst Array.

Konsequenz, die dokumentiert werden muss: `(("a" . 1) ("b" . 2))` → Objekt,
aber `(("a" 1) ("b" 2))` → Array von Arrays. Der Unterschied ist der Punkt.

**Bekannte Asymmetrien — bewusst, nicht reparieren:**

- `false` und `null` werden beide zu Nil. Round-Trip verliert die Unterscheidung.
- Ein leeres Alist ist Nil ist `null`, nicht `{}`.
- Verschachtelte Alists funktionieren rekursiv.
- Zyklische Strukturen → Endlosschleife. **Tiefenbegrenzung 64**, danach
  Fehler `"CellToJSON: Tiefe 64 überschritten"`.

---

## 7. `boot.js` — Client-API

Ausgeliefert unter `/_golisp/boot.js`, per `//go:embed` eingebettet, **nicht**
über `http-static`. Muss auch ohne konfiguriertes Static-Verzeichnis erreichbar
sein.

```js
golisp.call(op, ...args)      // → Promise, resolved mit ok, rejected mit err
golisp.on(event, fn)          // Push-Handler, mehrere pro Event erlaubt
golisp.off(event, fn)
golisp.ready                  // Promise, resolved beim ersten Connect
golisp.connected              // Boolean
```

**Reconnect:** Exponentiell 250 ms → 4 s, unbegrenzt. Beim Wiederverbinden
bleiben alle `on`-Handler registriert. Offene `call`-Promises werden beim
Verbindungsabbruch **rejected**, nicht neu gesendet.

Die WS-URL leitet sich aus `location.host` ab, Pfad `/_golisp/ws` — nicht
hartkodiert, damit ein späterer Reverse-Proxy nicht bricht.

Zusätzlich: `golisp.on('_reconnect', fn)`, damit eine App ihren Zustand nach
Verbindungsverlust neu holen kann.

---

## 8. Tests

### 8.1 Go-Unit (`jsoncell_test.go`)

Round-Trip für: `nil`, `t`, `42`, `3.14`, `-7`, `"hallo"`, `"mit \"quote\""`,
`"Umlaut äöü"`, `(1 2 3)`, `()`, `(("a" . 1))`, `(("a" . (("b" . 2))))`,
`((1 2) (3 4))`, tief verschachtelt 65 Ebenen → Fehler.

### 8.2 Manuell mit `curl`

```bash
./golisp -e '(begin (define s (http-serve 8099)) (http-wait s))' &
curl -s http://127.0.0.1:8099/_golisp/boot.js | head -3
```

### 8.3 Lisp-Integrationstest (`tests/webbridge.lisp`)

Nutzt `tests/test-framework.lisp`. Prüft: Port 0 liefert Port > 0,
`http-port` konsistent, `ws-export`/`ws-unexport` Rückgaben,
`ws-emit` ohne Clients → `0`, `http-stop` idempotent,
`(ws-call srv 999 "1")` mit unbekanntem Client → Fehler, nicht Hänger.

### 8.4 Zwei Szenarien, die per Hand mit einer Testseite zu prüfen sind

**A — Live-Redefinition:** Server starten, Browser verbinden, im REPL
`(ws-export srv "ping" (lambda (c) "eins"))`, im Browser aufrufen, dann im
REPL auf `"zwei"` ändern, erneut aufrufen. Muss `"zwei"` liefern, **ohne**
Reconnect.

**B — Reentranz:** Ein Handler, der seinerseits `ws-call` auf den aufrufenden
Client macht:

```lisp
(ws-export srv "breite" (lambda (c)
  (ws-call srv c "return window.innerWidth")))
```

Muss die Fensterbreite liefern und darf nicht blockieren.

---

## 9. Sicherheit

- Listener **nur** auf `127.0.0.1`. Kein Flag für `0.0.0.0` in Stufe 1.
- WS-Handshake prüft `Origin`: erlaubt sind `http://127.0.0.1:<port>` und
  `http://localhost:<port>`. Fehlender Origin-Header (Nicht-Browser-Clients)
  → erlaubt. Anderer Origin → 403.
- `http-static` muss Path-Traversal abwehren: `http.Dir` plus
  `filepath.Clean`, Symlinks nicht folgen.
- `ws-eval`/`ws-call` führen beliebiges JS aus — das ist Absicht und der
  Grund, warum nur lokal gebunden wird.

---

## 10. Was diese Spec **nicht** abdeckt

Ausdrücklich offen, damit ClaudeCode nicht improvisiert:

- **Kein Backpressure für `ws-emit`** bei vielen Clients. Bei 3–5 lokalen
  Clients irrelevant; bei 500 wäre es ein Thema.
- **Keine Nachrichtenreihenfolge-Garantie** zwischen `ws-emit` und
  `ws-eval` an denselben Client — beide gehen durch denselben Channel, also
  ist die Reihenfolge faktisch stabil, aber nicht zugesichert.
- **Kein Zustand über Reconnect.** Client-ID ändert sich beim Wiederverbinden.
  Wer Sessions braucht, baut sie in Stufe 2 darüber.
- **Kein `http-serve` auf Unix-Socket.**
- **Kein Test für den Fall, dass zwei `http-serve` denselben Port wollen** —
  soll sauber als Lisp-Error scheitern, ist aber nicht abgesichert.
- **`browser-open` erkennt nicht, ob das Fenster geschlossen wurde.** Dafür
  ist `idle-exit` da, und das ist eine Heuristik, keine Lösung.

---

## 11. Reihenfolge der Umsetzung

1. `jsoncell.go` + Tests — isoliert, ohne Netz, sofort verifizierbar
2. `httpserver.go` ohne WS — mit `curl` prüfbar
3. `wsbridge.go` Hub + Reader/Writer-Goroutinen
4. `boot.js` + eine Testseite `public/index.html` mit Button und Ausgabe
5. Szenarien 8.4 A und B

Nach Schritt 2 muss `curl` eine statische Datei ausliefern. Nach Schritt 4
muss ein Klick im Browser eine Lisp-Funktion aufrufen. Zwischenstände sind
committable.

---

## 12. Offene Punkte für Gerhard — beantwortet

- **Dateinamen:** entschieden für `httpserver.go`/`wsbridge.go`/`jsoncell.go`
  (kleingeschrieben, ohne Trenner) — konsistent mit `fileio.go`/`sigorest.go`.
- **Registrierung:** getrennt umgesetzt — `RegisterHTTPFuncs` (`httpserver.go`)
  und `RegisterWSFuncs` (`wsbridge.go`), beide aus `BaseEnv()`
  (`primitives.go:127-128`) aufgerufen.
