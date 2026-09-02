# GoLisp2 — Generierte Funktionsreferenz

> Automatisch erzeugt aus `(env-symbols)`. Vollständigkeit ist strukturell garantiert. Die Beschreibungstexte werden in `tools/gen-reference.lisp` gepflegt (*ref-docs*) — NICHT in dieser Datei, sie wird bei jedem Lauf komplett überschrieben. Neue Symbole ohne *ref-docs*-Eintrag erscheinen mit leerer Beschreibungsspalte.
> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`

| Symbol | Beschreibung |
|---|---|
| `%cond-coerce` | Intern: Argument in Condition umwandeln (condition.lisp) |
| `%cond-get` | Intern: Key aus Property-Liste lesen |
| `%cond-slot` | Intern: Slot-Wert einer Condition |
| `%cond-subtype?` | Intern: Subtyp-Pruefung in der Condition-Hierarchie |
| `%cond-type?` | Intern: Typ-Pruefung einer Condition |
| `%cond?` | Intern: Pruefung ob Objekt eine Condition ist |
| `%db-bindings` | Intern: Codegenerator fuer destructuring-bind |
| `%file-shared?` | Intern: Datei bereits durch anderes System geladen (defsystem) |
| `%generic-dispatch` | Intern: Methodendispatch der generischen Funktionen |
| `%generic-methods` | Intern: Methodenliste einer generischen Funktion |
| `%generic-registry` | Hash-Table der generischen Funktionen |
| `%loop-acc` | Intern: loop-Akkumulator-Aktion + Familiencheck (loop.lisp) |
| `%loop-acc-kw?` | Intern: loop-Akkumulations-Schluesselwort? (loop.lisp) |
| `%loop-drop-nonkw` | Intern: loop-Parser: Formen bis Schluesselwort ueberspringen |
| `%loop-for` | Intern: loop-Parser: for-Klausel (in/from) (loop.lisp) |
| `%loop-from` | Intern: loop-Parser: for-from/to/below/downto/by (loop.lisp) |
| `%loop-kw?` | Intern: loop-Klausel-Schluesselwort? (loop.lisp) |
| `%loop-parse` | Intern: loop-Parser: Klauseln -> let/while-Expansion (loop.lisp) |
| `%loop-take-nonkw` | Intern: loop-Parser: Formen bis Schluesselwort sammeln |
| `%loop-when` | Intern: loop-Parser: when/unless-Klausel (loop.lisp) |
| `%make-struct` | Intern: Konstruktor der defstruct-Instanzen |
| `%reduce-fold` | Intern: Faltkern von reduce |
| `%remove-first` | Intern: erstes Vorkommen entfernen (unload-system) |
| `%setf-one` | Intern: einzelne setf-Zuweisung ausfuehren |
| `%sys-entry` | Intern: Systemeintrag aus *systems* holen |
| `%sys-get` | Intern: Key aus System-Property-Liste lesen |
| `%sys-loaded?` | Intern: System bereits geladen? |
| `%topo` | Intern: topologische Sortierung mit Zyklenerkennung (defsystem) |
| `*` | Multiplikation aller Argumente |
| `*condition-types*` | Registry der definierten Condition-Typen |
| `*loaded-files*` | Liste der bereits geladenen Dateien (defsystem) |
| `*loaded-systems*` | Liste der geladenen Systeme (defsystem) |
| `*ref-docs*` | Beschreibungstabelle des Referenz-Generators (Symbol -> Text) |
| `*ref-out*` | Ausgabedatei des Referenz-Generators |
| `*setf-expanders*` | Registry der setf-Expander |
| `*systems*` | Registry der definierten Systeme (defsystem) |
| `+` | Addition aller Argumente |
| `-` | Subtraktion (erstes minus Rest); unär 0 |
| `/` | Division; Fehler bei Division durch 0 |
| `<` | numerisch kleiner |
| `<=` | numerisch kleiner oder gleich |
| `=` | numerische Gleichheit |
| `>` | numerisch größer |
| `>=` | numerisch größer oder gleich |
| `abs` | Absolutbetrag |
| `alist-get` | Wert zu Key aus Assoziationsliste oder nil |
| `alist-set` | Eintrag in Assoziationsliste setzen oder ersetzen |
| `any` | t, wenn mindestens ein Element das Praedikat erfuellt |
| `append` | Listen verketten; Atom als letztes Argument ergibt dotted pair |
| `apply` | Funktion aufrufen, letztes Argument wird als Liste gesplict |
| `argv` | Kommandozeilenargumente des Prozesses als Liste |
| `assert` | Makro: signalisiert Fehler, wenn Form nil ergibt |
| `assoc` | erstes (key . val)-Paar aus Assoziationsliste (equal?) |
| `atom` | t, wenn x kein Cons ist |
| `atom?` | Alias fuer atom |
| `browser-open` | URL im Browser oeffnen (detached) |
| `butlast` | Liste ohne letztes Element |
| `caar` | (car (car x)) |
| `cadddr` | (car (cdr (cdr (cdr x)))) |
| `caddr` | (car (cdr (cdr x))) |
| `cadr` | (car (cdr x)) |
| `car` | erstes Element einer Liste |
| `cdar` | (cdr (car x)) |
| `cddr` | (cdr (cdr x)) |
| `cdr` | Restliste ohne erstes Element |
| `chan-make` | Channel erzeugen (optional Puffergroesse n) |
| `chan-recv` | Wert aus Channel empfangen (blockierend) |
| `chan-send` | Wert in Channel senden |
| `clrhash` | Hash-Table leeren |
| `coerce` | Typumwandlung, unterstuetzt 'list und 'string |
| `complement` | Funktion, die das Praedikat negiert |
| `compose` | Funktion f(g(x)) |
| `cons` | neues Cons (x vor Liste) |
| `constantly` | Funktion, die stets x zurueckgibt |
| `copy-list` | flache Listenkopie |
| `copy-tree` | tiefe Kopie (rekursiv) |
| `decf` | Makro: place um d (Default 1) vermindern |
| `defgeneric` | generische Funktion definieren (Dispatch erstes Argument) |
| `define-condition` | Condition-Typ mit Slots definieren |
| `defined-in` | Liste der Symbole, die in der geladenen Datei definiert wurden |
| `defmethod` | Methode fuer generische Funktion definieren; Dispatch (var typ) oder (eql wert) |
| `defparameter` | globale Variable definieren, ueberschreibt immer (CL-Compat) |
| `defstruct` | Struktur definieren: erzeugt make-name, name-slot-Accessoren, name-p |
| `defstruct-resolve-name` | Intern: Namensaufloesung des defstruct-Makros |
| `defsystem` | System aus Dateien (:components) mit Abhaengigkeiten (:depends-on) definieren |
| `defvar` | Variable definieren (Dokumentationskonvention) |
| `destructuring-bind` | Listenelemente an Variablen binden (flache Muster) |
| `documentation` | Docstring einer defun/defmacro abfragen ('function) |
| `dolist` | Makro: ueber Liste iterieren, Ergebnisform () |
| `dotimes` | Makro: var von 0 bis n-1 wiederholen |
| `drop` | Liste ohne die ersten n Elemente |
| `env-symbols` | Liste aller Root-Environment-Symbole |
| `environ` | Umgebungsvariablen als Assoziationsliste |
| `eq` | Pointer-Identitaet; Symbole sind interniert, Zahlen/Strings nicht |
| `eq?` | Alias fuer eq |
| `eql` | wie eq, aber Zahlen mit gleichem Wert gelten als eql |
| `equal?` | strukturelle Gleichheit (rekursiv) |
| `err-write` | String auf stderr schreiben |
| `error` | Laufzeitfehler signalisieren (faengbar mit catch) |
| `every` | t, wenn alle Elemente das Praedikat erfuellen |
| `exit` | Prozess mit Exit-Code beenden (Default 0) |
| `expt` | basis hoch exp (exp ganzzahlig) |
| `file-append` | Inhalte an Datei anhaengen |
| `file-delete` | Datei loeschen |
| `file-exists?` | t, wenn Datei existiert |
| `file-read` | Dateiinhalt als String lesen |
| `file-stat` | Assoziationsliste mit size und mtime oder nil |
| `file-write` | Inhalte in Datei schreiben (ueberschreibt) |
| `filter` | Elemente, fuer die das Praedikat wahr ist |
| `find-all` | alle Vorkommen von item (Default-Test equal?) |
| `first` | Alias fuer car |
| `flatten` | rekursiv flache Liste |
| `floor` | zur naechsten ganzen Zahl abrunden |
| `for-each` | Funktion auf jedes Element anwenden, Ergebnis nil |
| `format` | CL-FORMAT-Engine (HyperSpec 22.3); Ziel nil=String, t=stdout |
| `fourth` | Alias fuer cadddr |
| `fprintf` | C-Format in Datei (append) oder Systemstream "stdout"/"stderr" schreiben |
| `funcall` | Funktion mit Argumenten aufrufen |
| `ga-calc` | Fitness parallel berechnen, Population absteigend sortieren |
| `ga-create` | genetischen Algorithmus erzeugen (typ gen-len gen-par fitness-fn) |
| `ga-cross` | Crossover mit Blockgroesse codist |
| `ga-init` | Population mit Zufallswerten initialisieren |
| `ga-mut` | jedes Gen mit Wahrscheinlichkeit mutf mutieren |
| `ga-print` | Population formatiert ausgeben (lines steuert Umfang) |
| `ga-result` | Fitness-Scores als sortierte Liste |
| `ga-select` | die besten keep Genome behalten |
| `ga?` | Pruefung ob GA-Handle |
| `gcd` | groesster gemeinsamer Teiler |
| `gensym` | eindeutiges Symbol fuer Makros |
| `get-file-path` | Pfad aufloesen (load-Suchpfad-Logik) |
| `get-universal-time` | Sekunden seit 1.1.1900 00:00 UTC |
| `get-working-directory` | aktuelles Arbeitsverzeichnis |
| `getenv` | Wert einer Umgebungsvariablen oder nil |
| `getf` | Wert zu Key aus Property-Liste oder default |
| `gethash` | Hash-Lookup: liefert Wert und t/nil als Gefunden-Indikator |
| `gets` | eine Zeile von stdin lesen |
| `handler-case` | Conditions und Laufzeitfehler typbasiert abfangen |
| `hash-table-count` | Anzahl der Eintraege |
| `hash-table-p` | Pruefung ob Hash-Table |
| `http-port` | tatsaechlich gebundenen Port liefern |
| `http-serve` | HTTP-Server starten (host Default 127.0.0.1, port 0=freier Port) |
| `http-static` | Verzeichnis unter URL-Pfad mounten (kein Directory-Listing) |
| `http-stop` | Graceful Shutdown, idempotent |
| `http-upload` | POST-Endpoint fuer Multipart-Uploads; handler(dateiname, inhalt) |
| `http-wait` | blockieren bis http-stop/SIGINT oder :idle-exit |
| `identity` | Argument unverändert zurueckgeben |
| `ignore-errors` | Makro: bei Fehler nil statt Fehler liefern |
| `incf` | Makro: place um d (Default 1) erhoehen |
| `intern` | interniertes Symbol zum String liefern |
| `iota` | Liste (0 1 ... n-1) |
| `last` | letztes Element einer Liste |
| `length` | Elementzahl einer Liste bzw. Zeichenzahl eines Strings |
| `lisp-error-msg` | :msg-Slot einer Condition liefern |
| `list` | Liste aus den Argumenten erzeugen |
| `list->string` | Stringliste zu einem String verketten |
| `list-tail` | n-ten Rest der Liste |
| `list?` | t, wenn Liste oder nil |
| `load-system` | System topologisch sortiert laden (Abhaengigkeiten zuerst) |
| `loaded-systems` | Liste der geladenen Systeme |
| `lock-make` | Mutex fuer lock erzeugen |
| `loop` | Iteration (CL-Praxis-Kern): for/repeat/when/collect/sum/... (loop.lisp) |
| `macroexpand-1` | einen Makro-Expansionsschritt ausfuehren |
| `make-hash-table` | thread-sichere Hash-Table erzeugen |
| `make-list` | Liste mit n Elementen (:initial-element) |
| `mapcar` | Funktion elementweise anwenden, Ergebnisliste (first-class, #' moeglich) |
| `maphash` | Funktion (key wert) auf jeden Hash-Eintrag anwenden |
| `max` | groesste Zahl |
| `member` | Restliste ab erstem Treffer |
| `memstats` | Go-Runtime-Memory-Statistiken als Assoziationsliste |
| `min` | kleinste Zahl |
| `mod` | Rest der Division (Gleitkomma) |
| `negative?` | t, wenn kleiner 0 |
| `nil` | NIL-Konstante (Singleton, MakeNil) |
| `nth` | n-tes Element (0-basiert) |
| `null` | t, wenn nil |
| `null?` | Alias fuer null |
| `number->string` | Zahl in String umwandeln |
| `number?` | t, wenn Zahl |
| `pair?` | t, wenn Cons |
| `pg-close` | PostgreSQL-Verbindung schliessen |
| `pg-connect` | PostgreSQL-Verbindung oeffnen (Connection-String) |
| `pg-exec` | schreibendes SQL, betroffene Zeilen zurueck |
| `pg-query` | SELECT, Liste von Zeilen als Assoziationslisten |
| `pop` | Makro: erstes Element aus Liste in var entfernen und liefern |
| `positive?` | t, wenn groesser 0 |
| `print` | Werte ohne Newline auf stdout, letztes Argument zurueck |
| `printf` | C-Format direkt auf stdout ausgeben |
| `println` | Werte mit Newline auf stdout, letztes Argument zurueck |
| `push` | Makro: wert vor Liste in var einfuegen |
| `puthash` | Hash-Eintrag setzen: (puthash key tabelle wert) |
| `random` | Zufallszahl; mit n aus [0, n) |
| `read` | String in Lisp-Datenstruktur parsen |
| `read-line` | eine Zeile von stdin lesen |
| `redef-log` | Ringpuffer der Redefinitionen |
| `redef-log-clear` | Redefinitions-Log leeren |
| `redefine-policy` | Redefinitions-Politik setzen/lesen: allow, warn (Default), error |
| `reduce` | Liste falten: (reduce f seq :initial-value ...)  |
| `ref-lookup` | Intern: Beschreibungstext des Referenz-Generators zum Symbolnamen |
| `register-setf-expander` | setf-Expander fuer eigenen Accessor registrieren |
| `remainder` | Alias fuer mod |
| `remhash` | Hash-Eintrag entfernen |
| `remove` | alle Vorkommen von item entfernen (equal?) |
| `remove-duplicates` | Duplikate entfernen (erstes Vorkommen bleibt) |
| `remove-if` | Elemente entfernen, fuer die Praedikat wahr ist |
| `remove-if-not` | Elemente behalten, fuer die Praedikat wahr ist |
| `rest` | Alias fuer cdr |
| `reverse` | Liste umdrehen |
| `second` | Alias fuer cadr |
| `set-difference` | Elemente von a ohne die aus b |
| `set-nth` | Kopie mit n-tem Element ersetzt |
| `set-working-directory` | Arbeitsverzeichnis wechseln |
| `setf` | generalisierte Zuweisung: Variablen, (car lst), (gethash key tbl), struct-Slots |
| `shell-assoc` | passendes Paar aus Assoziationsliste suchen (Shell-Output-Parsing) |
| `shm-alloc` | Block im Shared-Memory-Pool belegen |
| `shm-cleanup` | Shared-Memory-Pool aufraeumen |
| `shm-free` | Shared-Memory-Block freigeben |
| `shm-read` | Block (maximal n Zeichen) als String lesen |
| `shm-status` | Pool-Statistik als Assoziationsliste (total, used, free) |
| `shm-write` | String in Shared-Memory-Block schreiben |
| `signal` | Condition des definierten Typs signalisieren (:msg ...) |
| `sigo` | Prompt an sigoREST senden, Antwort als String |
| `sigo-host` | sigoREST-Host setzen oder lesen |
| `sigo-models` | Liste der verfuegbaren sigoREST-Modelle |
| `sleep` | Millisekunden pausieren |
| `slurp` | stdin komplett als String lesen |
| `sort` | Liste nach Vergleich sortieren (neue Liste) |
| `sprintf` | C-Format in String umwandeln |
| `sqrt` | Quadratwurzel |
| `square` | x quadrieren |
| `sscanf` | String nach C-scanf-Muster parsen, Liste der Werte |
| `string->list` | String in Liste von Einzelzeichen-Strings |
| `string->number` | String als Zahl parsen |
| `string-append` | Strings verketten |
| `string-contains` | t, wenn Teilstring enthalten |
| `string-downcase` | in Kleinbuchstaben umwandeln |
| `string-find` | Position des ersten Vorkommens oder nil |
| `string-length` | Zeichenanzahl (Runes) |
| `string-replace` | alle Vorkommen ersetzen |
| `string-trim` | fuehrende und abschliessende Leerzeichen entfernen |
| `string-upcase` | in Grossbuchstaben umwandeln |
| `string?` | t, wenn String |
| `substring` | Teilstring von start (inklusive) bis end (exklusive) |
| `symbol->string` | Symbolname als String |
| `symbol-name` | Symbolname als String |
| `symbol?` | t, wenn Symbol |
| `system` | Shell-Kommando ausfuehren, Exit-Code zurueck |
| `system-symbols` | Symbole, die ein System definiert hat |
| `t` | T-Konstante (Singleton) |
| `take` | erste n Elemente |
| `third` | Alias fuer caddr |
| `trace` | Live-Tracing einer Funktion aktivieren: Aufruf und Ergebnis auf stderr |
| `trace?` | t, wenn Funktion gerade getraced wird |
| `union` | Vereinigungsmenge zweier Listen |
| `unless` | Makro: body nur auswerten, wenn test falsch ist |
| `unload-system` | System aus Ladestatistik entfernen (Definitionen bleiben) |
| `untrace` | Tracing deaktivieren |
| `values` | mehrere Werte liefern; Verwender sehen den Hauptwert |
| `warn` | Warnung ausgeben, Auswertung laeuft weiter |
| `webserv` | Web-Bootstrap: HTTP-Server plus boot.js plus Browser (:html/:htmlpath Pflicht) |
| `when` | Makro: body nur auswerten, wenn test wahr ist |
| `write-reference` | docs/referenz-generiert.md neu generieren |
| `ws-call` | JS im Browser des Clients aufrufen, blockiert auf Ergebnis (:timeout) |
| `ws-clients` | Liste der verbundenen Client-IDs |
| `ws-emit` | Server-Push an alle Clients, Anzahl Empfaenger zurueck |
| `ws-emit-to` | Server-Push an einzelne Client-ID |
| `ws-eval` | JS-Code an alle Clients feuern, ohne auf Ergebnis zu warten |
| `ws-export` | Funktion als browser-aufrufbare Operation exportieren (golisp.call) |
| `ws-unexport` | exportierte Operation entfernen |
| `zero?` | t, wenn gleich 0 |
| `zerop` | CL-Name fuer zero? |
| `zip` | zwei Listen elementweise zu Paaren verbinden |
