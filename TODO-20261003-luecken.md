# TODO 20261003 - Lücken in golisp2, gefunden im Projekt ai-vergleiche

Anlass: Für den sigoREST-Modellvergleich (`/u/ki-projekte/ai-vergleiche`, `src/steuerung.lisp`, ca. 350 Zeilen)
habe ich golisp2 einen Tag lang als Steuerung benutzt. Dabei fehlte einiges oder war umständlich.
Geprüft im Interpreter (`./build/golisp2 -e …`, `(env-symbols)` liefert 310 Symbole), Stand 03.10.2026.

Legende: **belegt** = im Interpreter ausprobiert, **offen** = nicht geprüft oder unklar.

## 0. Die Doku ist an zwei Stellen veraltet

- [ ] `CLAUDE.md` / `sprachen/golisp2.md` (Projektvorgaben) behaupten „Keine Vektoren, keine Hash-Maps“.
  **Belegt:** Hash-Tabellen gibt es (`make-hash-table`, `puthash`, `gethash`, `remhash`, `clrhash`, `maphash`,
  `hash-table-count`, `hash-table-p`). Argumentreihenfolge wie CL: `(puthash KEY TABLE VALUE)`, `(gethash KEY TABLE)`.
  Falsche Reihenfolge gibt nur „Hashtabelle erwartet, got …“. Das steht schon in `TODO.md` als „Lispbuch-CLAUDE.md veraltet“.
- [ ] Dieselbe Doku sagt, `(sigo …)` liefere nur Text. **Belegt:** `(sigo* …)` liefert Text, Modell, finish-reason,
  Prompt-/Completion-/Cached-/Reasoning-Tokens und `cost-usd`, dazu `(sigo-usage)`. Ich habe deshalb im Projekt
  zu Unrecht einen Go-Runner gebaut, weil ich `sigo*` nicht kannte. Grenzen von `sigo*` siehe 6.

## 1. Zeit (Gerhards Idee, höchste Priorität)

**Belegt:** Nur `get-universal-time` (ganze Sekunden, Unix-Sekunden plus Offset?) und `sleep` existieren.
`steuerung.lisp` ruft deshalb `(system "date +%s.%N > datei")` auf und liest die Datei zurück
(Funktion `jetzt`, dazu `date '+%F %T'` für das Protokoll).

- [x] `(now)` → Sekunden mit Nachkommastellen (Float), oder `(now-ms)`.
- [x] `(format-time fmt [zeit])` im Stil von `strftime`, z. B. `"%F %T"`.
- [ ] `(parse-time text fmt)`, falls einfach machbar.
- **Geklärt 20261003:** `get-universal-time` ist exakt CL-Universal-Time (Epoche 1900, Abstand zu Unix 2208988800 s). `(now)` liefert Unix-Sekunden als Float, `(format-time fmt [zeit] [:utc])` strftime-artig. `parse-time` nicht gebaut (YAGNI).
- Alt: `(get-universal-time)` lieferte 4000022742. Das ist **nicht** die Unix-Zeit (ca. 1,79 Mrd. am 03.10.2026),
  sondern vermutlich CL-Universal-Time (Sekunden seit 1900) plus etwas? 1900→2026 wären ca. 3,98 Mrd.,
  passt grob. **Offen:** stimmt die Zahl genau, und ist das gewollt? Vor dem Bau von `now` festlegen, welche Epoche gilt.

## 2. Befehlsausgabe ohne Umweg über Dateien

**Belegt:** `system` liefert nur den Exit-Code. Jede Ausgabe (Zeit, Git-Version, Dateiliste, Python-Ausgabe)
muss nach Datei umgeleitet und mit `file-read` gelesen werden.

- Erledigt 20261003 als Fassade über die schon vorhandene Spezialform `exec` (fängt stdout/stderr/Exit-Code). `ki-referenz.md` beschrieb `exec` falsch als `(exec shell-cmd)` — korrigiert.
- [x] `(shell-output cmd)` → String, oder `(system* cmd)` → `(exit-code . ausgabe)`.
- Unklar: `shell-assoc` heißt nach Hilfe-Text „Shell-Befehl“, ist aber laut `src/lib/shellcmd.go:35` nur
  `assoc` über eine Liste (Schlüssel, Alist, **nicht** ein Shell-Aufruf). Der Name führt in die Irre.
  **Offen:** Umbenennen oder dokumentieren?

## 3. JSON

**Belegt:** kein `json-parse`, kein `json-encode`. Im Projekt waren `aufruf.json`, `bewertung.json`, die sigoREST-Antworten
und `/api/costs` alle JSON. Ich habe die Teile in Python/Go ausgelagert. In Lisp hätte ich sie nicht lesen können.

- [x] `(json-parse text)` → Alist/Liste (Objekt → Alist, Array → Liste, `true/false/null` → `t`/`nil`/?).
- [x] `(json-encode wert)`.
- **Entschieden 20261003:** Objekt ↔ Hash-Tabelle (equal), Array ↔ Liste, `null` ↔ `:null`, `false` → `()`, `()` → `[]`, `:false` → `false`. Alist-Heuristik gestrichen, gilt auch für die Web-Bridge (`jsoncell.go` bleibt die einzige Abbildung); parvmira `punkt->alist` → `punkt->hash`.
- Alt: Abbildung von `false` und `null` (beide `nil`?) und von leeren Objekten/Arrays. Vorher entscheiden.

## 4. Dateien und Strings

- Erledigt 20261003: `(string-split s [sep])`, `(string-join liste sep)`, `(directory-files dir [muster])` (Namen, sortiert, Verzeichnisse mit `/`). Kein `glob` über ganze Pfade.
- [x] **Verzeichnis auflisten / glob**: fehlt (**belegt**: kein Symbol dafür). `mkdir -p` lief über `system`.
- [x] **`string-split`, `string-join`** fehlen (**belegt**, kein Symbol). Ich habe `zeilen` selbst geschrieben.
- [ ] **Regulärausdrücke**: `(regexp …)` ist **kein** Symbol (**belegt**). Im Go-Quelltext kommt der String `"regexp"` vor.
  Unklar, wofür (vielleicht nur ein Import). Es gibt offenbar keine Regex-Funktion für Lisp-Code.
- [ ] **Vektoren/Arrays** mit O(1)-Zugriff (`make-array`, `aref`): nicht vorhanden (**belegt**, kein Symbol).
- [ ] `string-find` hat die Reihenfolge `(string-find NADEL HEUHAUFEN)` und liefert den Index oder `nil`.
  **Belegt:** `(string-find "b" "abc")` → 1, `(string-find "abc" "b")` → `()`. Das ist umgekehrt zu `string-contains "abc" "b"` → `t`
  (Heuhaufen zuerst). Zwei ähnliche Funktionen mit entgegengesetzter Argumentreihenfolge. Ich bin darüber gestolpert.
  **Offen:** Absicht? Dokumentieren oder angleichen?

## 5. Mögliche Fehler (einzeln zu prüfen)

- [x] `(coerce 1 'string)` liefert `""` statt `"1"` und keinen Fehler (**belegt**). `(coerce "12" 'number)` liefert einen
  klaren Fehler („nur 'list/'string“). Der stille leere String ist gefährlich. Fehler oder `"1"`.
- [ ] `(get-working-directory)` liefert `()` (**belegt**), wenn es nie gesetzt wurde. **Offen:** Sollte es nicht das echte
  Arbeitsverzeichnis liefern? Ich habe es nicht in einem Skript geprüft, nur mit `-e`.
- [ ] `slurp` erwartet laut Fehlermeldung „keine Argumente“ (**belegt**). Nach dem Namen hätte ich „Datei lesen“ erwartet
  (das ist `file-read`). **Offen:** `slurp` liest vermutlich stdin. Doku prüfen.
- [ ] `(environ)` lieferte in meinem Test die Umgebung **doppelt** ineinander geschachtelt (**belegt** an der Ausgabe, die
  erste Liste wiederholt sich innerhalb der zweiten). Das kann an meinem Aufruf `-e "(println (environ))"` liegen (die
  Ausgabe erscheint sowohl als `println` als auch als Ergebnis der Zeile). **Offen:** Dasselbe gilt für alle `-e`-Tests in
  dieser Session: Jede Ausgabe kam doppelt. Prüfen, ob `-e` den Wert zusätzlich druckt (vermutlich gewollt, REPL-Stil).

- [ ] Printer gibt große Floats in Exponentialschreibweise aus: `(now)` → `1.7910463319138556e+09`, `1791046331.5` → `1.7910463315e+09` (**belegt** 20261003). Umgehung: `(format nil "~,3f" x)` oder `(format-time "%s.%3N")`. CL druckt `1.7910463e9` (single) bzw. Dezimal — Regel festlegen.

## 6. KI-Anbindung `sigo*` reicht nicht für Messungen

**Belegt (Doku `docs/sigo.md`):** Timeout nur per Umgebungsvariable `GOLISP_SIGO_TIMEOUT` (Standard 120 s), nicht pro Aufruf.
**Offen (nicht geprüft):** Ob `sigo*` streamt. In `src/lib/sigorest.go` fand ich kein `stream`, gelesen habe ich die
Datei nicht vollständig. Ob es Zeit misst, ist ebenfalls unklar (Zeit ließe sich mit 1. selbst messen).

Warum das zählt: Im Vergleich brachen Aufrufe bei genau 300 s ab (`WriteTimeout` in sigoREST). Nicht-streamende Aufrufe
mit Reasoning-Modellen laufen leicht in solche Grenzen. Für solche Messungen bräuchte golisp2:
- Erledigt 20261004 (Branch sigo-request, gemergt): `(sigo-request h [host])` mit Feld `timeout` pro Call, `elapsed` im Ergebnis, `sigo*`/`sigo-usage` als Hash-Tabelle, dazu `sigo-costs`, `sigo-budget`, `sigo-model-info`. `stream` ist in `sigo-request` gesperrt.
- [x] Timeout als Parameter pro Aufruf.
- [ ] Streaming-Unterstützung. **Achtung:** sigoREST bucht bei Streaming die Kosten falsch
  (siehe `/u/go-projekte/sigoREST/TODO-20261003-kosten.md`). `cost-usd` aus `sigo*` wäre dort ebenfalls zu prüfen.
- [ ] `cost-usd` von `sigo*` gegen `/api/costs` abgleichen (die Doku nennt es „obere Schranke“). Werkzeug da: `(sigo-costs t0)` als Messfenster, Abgleich selbst offen.

## 7. Nebenläufigkeit

- **Belegt:** `parfunc`, `spawn`, `lock-make`, `chan-make` existieren. Im Projekt lief ein Pool mit N Arbeitern über M Zellen
  (`*parallel*` = 3 bis 4). Ich habe ihn in `steuerung.lisp` selbst gebaut.
- [ ] Prüfen, ob die Stdlib einen Arbeiterpool bietet oder ob `(pool-map n fn liste)` sinnvoll wäre.
  **Offen:** Ob mein Eigenbau Lücken hat, habe ich nicht untersucht. Er lief stabil in 45 + 9 + 9 Zellen.

## 8. Was gut funktionierte (zur Einordnung)

- Trampolin/TCO, `trap`, `printf`/`sprintf`, `file-*`, `system`, `argv`, `exit`, `lock`. Die Steuerung mit Fortsetzen über
  Markerdateien lief fehlerfrei durch.
- `(exit (hauptprogramm))` als Exit-Code-Muster und `golisp2 skript.lisp arg` mit `(argv)` waren bequem.

## Reihenfolge-Vorschlag

1. Doku korrigieren (0.) und `coerce`-Verhalten (5.), klein und sicher.
2. `now` / `format-time` (1.) und `shell-output` (2.).
3. `json-parse` / `json-encode` (3.).
4. `string-split` / `string-join` / Verzeichnis auflisten (4.).
5. Danach entscheiden, ob `sigo*` Streaming und Timeout pro Aufruf bekommt (6.).
