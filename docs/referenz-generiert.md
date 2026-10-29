# GoLisp2 — Generierte Funktionsreferenz

> Automatisch erzeugt aus `(env-symbols)`. Vollständigkeit ist strukturell garantiert. Die Beschreibungstexte werden in `tools/gen-reference.lisp` gepflegt (*ref-docs*) — NICHT in dieser Datei, sie wird bei jedem Lauf komplett überschrieben. Neue Symbole ohne *ref-docs*-Eintrag erscheinen mit leerer Beschreibungsspalte.
> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`

| Symbol | Beschreibung |
|---|---|
| `%bad-spec` | Intern: Fehler bei ungueltiger typep-Spec (types.lisp) |
| `%builtin-subtype?` | Intern: Subtyp-Pruefung in *type-parents* (types.lisp) |
| `%builtin-type?` | Intern: Pruefung ob Name ein eingebauter Typ ist (types.lisp) |
| `%cell-type` | Intern: Kern-Typ einer Cell als Symbol (celltype.go) |
| `%cond-coerce` | Intern: Argument in Condition umwandeln (condition.lisp) |
| `%cond-get` | Intern: Key aus Property-Liste lesen |
| `%cond-slot` | Intern: Slot-Wert einer Condition |
| `%cond-subtype?` | Intern: Subtyp-Pruefung in der Condition-Hierarchie |
| `%cond-type?` | Intern: Typ-Pruefung einer Condition |
| `%cond?` | Intern: Pruefung ob Objekt eine Condition ist |
| `%cons-type` | Intern: type-of fuer Cons: Condition-, Struct- oder cons-Typ (types.lisp) |
| `%db-bindings` | Intern: Codegenerator fuer destructuring-bind |
| `%deftype-check` | Intern: deftype-Name gegen eingebaute Typen, Structs, Conditions pruefen (types.lisp) |
| `%deftype-expand` | Intern: deftype-Expander mit Argumenten aufrufen (types.lisp) |
| `%file-shared?` | Intern: Datei bereits durch anderes System geladen (defsystem) |
| `%ganzzahl-pruefen` | Intern: Fehler, wenn n keine ganze Zahl ist (evenp/oddp) |
| `%generic-dispatch` | Intern: Methodendispatch der generischen Funktionen |
| `%generic-methods` | Intern: Methodenliste einer generischen Funktion |
| `%generic-registry` | Hash-Table der generischen Funktionen |
| `%infinite?` | Intern: Pruefung auf +/-Inf (types.lisp) |
| `%list->vector` | Intern: Liste als neuer Vektor (fuer coerce) |
| `%loop-acc` | Intern: loop-Akkumulator-Aktion + Familiencheck (loop.lisp) |
| `%loop-acc-kw?` | Intern: loop-Akkumulations-Schluesselwort? (loop.lisp) |
| `%loop-drop-nonkw` | Intern: loop-Parser: Formen bis Schluesselwort ueberspringen |
| `%loop-for` | Intern: loop-Parser: for-Klausel (in/from) (loop.lisp) |
| `%loop-from` | Intern: loop-Parser: for-from/to/below/downto/by (loop.lisp) |
| `%loop-kw?` | Intern: loop-Klausel-Schluesselwort? (loop.lisp) |
| `%loop-parse` | Intern: loop-Parser: Klauseln -> let/while-Expansion (loop.lisp) |
| `%loop-take-nonkw` | Intern: loop-Parser: Formen bis Schluesselwort sammeln |
| `%loop-when` | Intern: loop-Parser: when/unless-Klausel (loop.lisp) |
| `%lower-ok?` | Intern: Untergrenze einer typep-Range pruefen (types.lisp) |
| `%make-struct` | Intern: Konstruktor der defstruct-Instanzen |
| `%number-type` | Intern: type-of fuer Zahlen: integer oder float (types.lisp) |
| `%one-arg?` | Intern: Pruefung auf genau ein Argument (types.lisp) |
| `%pool-arbeiter` | Intern: Arbeiter-Schleife von pool-map (stdlib.lisp) |
| `%proper-length` | Intern: Laenge einer echten Liste, -1 bei Dotted Pair (types.lisp) |
| `%proper-list?` | Intern: Pruefung auf echte (nicht gepunktete) Liste (types.lisp) |
| `%reduce-fold` | Intern: Faltkern von reduce |
| `%remove-first` | Intern: erstes Vorkommen entfernen (unload-system) |
| `%setf-one` | Intern: einzelne setf-Zuweisung ausfuehren |
| `%st-all` | Intern: subtypep-Teilergebnisse, alle muessen gelten (types.lisp) |
| `%st-bound` | Intern: Bereichsgrenze normalisieren (types.lisp) |
| `%st-bound-in?` | Intern: Grenze innerhalb einer anderen (types.lisp) |
| `%st-cons-free?` | Intern: eingebauter Typ ohne Schnitt mit cons (Untertyp von atom) (types.lisp) |
| `%st-deftype?` | Intern: ist die Typangabe ein deftype (types.lisp) |
| `%st-expand` | Intern: deftype-Angabe fuer subtypep expandieren (types.lisp) |
| `%st-ja/nein` | Intern: Wahrheitswert als sicheres subtypep-Ergebnis (types.lisp) |
| `%st-left` | Intern: subtypep mit zusammengesetzter linker Seite (types.lisp) |
| `%st-name-kind` | Intern: Typname einordnen: eingebaut, Struct, Condition (types.lisp) |
| `%st-names` | Intern: subtypep fuer zwei Typnamen (types.lisp) |
| `%st-range-bounds` | Intern: beide Grenzen eines Zahlbereichs (types.lisp) |
| `%st-range-in?` | Intern: Zahlbereich in Zahlbereich (types.lisp) |
| `%st-range?` | Intern: ist die Angabe ein Zahlbereich (types.lisp) |
| `%st-right` | Intern: subtypep mit zusammengesetzter rechter Seite (types.lisp) |
| `%st-some` | Intern: subtypep-Teilergebnisse, eines muss gelten (types.lisp) |
| `%st-unknown-compound` | Intern: not/satisfies/member/eql unentscheidbar, sonst Fehler (types.lisp) |
| `%strip-trailing-newlines` | Intern: abschliessende Newlines entfernen (shell-output) |
| `%struct-instance?` | Intern: Pruefung ob x eine Instanz des Structs name ist (types.lisp) |
| `%subtypep` | Intern: subtypep als Liste (sub? sicher?) (types.lisp) |
| `%symbol-type` | Intern: type-of fuer Symbole: boolean, keyword oder symbol (types.lisp) |
| `%sys-entry` | Intern: Systemeintrag aus *systems* holen |
| `%sys-get` | Intern: Key aus System-Property-Liste lesen |
| `%sys-loaded?` | Intern: System bereits geladen? |
| `%topo` | Intern: topologische Sortierung mit Zyklenerkennung (defsystem) |
| `%type-name-warnings` | Intern: WARN-Formen bei Typnamen-Kollisionen (defstruct/define-condition) (types.lisp) |
| `%type-start` | Intern: Einstiegstyp in *type-parents* fuer x (types.lisp) |
| `%typecase-clause` | Intern: typecase-Klausel in cond-Klausel uebersetzen (types.lisp) |
| `%typep-name` | Intern: typep-Aufloesung eines Typnamens: eingebaut, Struct, Condition (types.lisp) |
| `%typep-range` | Intern: typep fuer (integer lo hi) u. ae. (types.lisp) |
| `%typep-satisfies` | Intern: typep fuer (satisfies f) (types.lisp) |
| `%typep-spec` | Intern: typep-Dispatch ueber die Typangabe (or/and/not/member/eql/satisfies/Range/Name) (types.lisp) |
| `%upper-ok?` | Intern: Obergrenze einer typep-Range pruefen (types.lisp) |
| `%valid-bound?` | Intern: Pruefung ob eine typep-Range-Grenze gueltig ist (types.lisp) |
| `%vector->list` | Intern: Vektor als frische Liste (fuer coerce) |
| `%vector-length` | Intern: Laenge eines Vektors (fuer length) |
| `*` | Multiplikation aller Argumente |
| `*condition-types*` | Registry der definierten Condition-Typen |
| `*deftypes*` | Registry (name expander) der deftype-Definitionen (types.lisp) |
| `*loaded-files*` | Liste der bereits geladenen Dateien (defsystem) |
| `*loaded-systems*` | Liste der geladenen Systeme (defsystem) |
| `*setf-expanders*` | Registry der setf-Expander |
| `*setf-mutators*` | |
| `*struct-types*` | Registry (name slot-anzahl) der definierten Structs, fuer type-of/typep (stdlib.lisp) |
| `*systems*` | Registry der definierten Systeme (defsystem) |
| `*type-parents*` | Typhierarchie (typ (direkte-obertypen …)); einzige Quelle fuer type-of/typep (types.lisp) |
| `+` | Addition aller Argumente |
| `-` | Subtraktion (erstes minus Rest); (- x) negiert, (-) → 0 |
| `/` | Division; Fehler bei Division durch 0 |
| `1+` | n + 1 (CL) |
| `1-` | n - 1 (CL) |
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
| `aref` | Vektor-Element lesen (O(1)), Index 0..n-1 |
| `argv` | Kommandozeilenargumente des Prozesses als Liste |
| `aset` | Vektor-Element setzen, liefert den Wert; auch (setf (aref v i) x) |
| `ash` | arithmetischer Shift: (ash x n) = x·2^n, negatives n schiebt nach rechts (floor) |
| `ash-left` | Hilfsfunktion von ash: n-mal verdoppeln |
| `ash-right` | Hilfsfunktion von ash: n-mal floor-halbieren |
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
| `ceiling` | auf die naechste ganze Zahl aufrunden; 2 Werte (CL): Quotient, Rest |
| `chan-close` | Channel schliessen -> t; zweites Mal Fehler |
| `chan-closed?` | t nur wenn Channel geschlossen und leer |
| `chan-make` | Channel erzeugen (optional Puffergroesse n) |
| `chan-recv` | Wert aus Channel empfangen (blockierend, optional :timeout sek): (values x t); geschlossen+leer oder Timeout -> (values nil nil) |
| `chan-select` | erster liefernder Channel einer Liste: (values x t i); (values nil nil nil) bei Timeout oder wenn alle geschlossen; optional :timeout sek |
| `chan-send` | Wert in Channel senden (blockierend; geschlossener Channel -> Fehler) |
| `chan-try-recv` | nicht blockierend empfangen: (values x t) oder sofort (values nil nil) |
| `chan-try-send` | nicht blockierend senden: t gesendet, nil wenn Puffer voll |
| `check-type` | Typ zusichern: (check-type x integer) → () oder Fehler; optional Beschreibung |
| `clrhash` | Hash-Table leeren |
| `coerce` | Typumwandlung, unterstuetzt 'list und 'string |
| `complement` | Funktion, die das Praedikat negiert |
| `compose` | Funktion f(g(x)) |
| `cons` | neues Cons (x vor Liste) |
| `constantly` | Funktion, die stets x zurueckgibt |
| `copy-list` | flache Listenkopie |
| `copy-tree` | tiefe Kopie (rekursiv) |
| `decf` | Makro: place um d (Default 1) vermindern |
| `defclass` | Klasse fuer Objekte mit Identitaet (Hashtabelle, setf in place) |
| `defgeneric` | generische Funktion definieren (Dispatch erstes Argument) |
| `define-condition` | Condition-Typ mit Slots definieren |
| `defined-in` | Liste der Symbole, die in der geladenen Datei definiert wurden |
| `defmethod` | Methode fuer generische Funktion definieren; Dispatch (var typ), typ = Struct, Klasse oder t |
| `defparameter` | globale Variable definieren, ueberschreibt immer (CL-Compat) |
| `defstruct` | Struktur definieren: erzeugt make-name, name-slot-Accessoren, name-p |
| `defstruct-resolve-name` | Intern: Namensaufloesung des defstruct-Makros |
| `defsystem` | System aus Dateien (:components) mit Abhaengigkeiten (:depends-on) definieren |
| `deftype` | Eigener Typname: (deftype ziffer () '(integer 0 9)), mit Parametern (deftype bis (&optional (n '*)) …) |
| `defvar` | Variable definieren (Dokumentationskonvention) |
| `destructuring-bind` | Listenelemente an Variablen binden (flache Muster) |
| `directory-files` | sortierte Namen in dir, optional Glob-Muster; Verzeichnisse mit / |
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
| `etypecase` | wie typecase, ohne Treffer ein Fehler |
| `evenp` | t wenn ganze Zahl n gerade ist (CL), sonst Fehler bei Nicht-Ganzzahl |
| `every` | t, wenn alle Elemente das Praedikat erfuellen |
| `exit` | Prozess mit Exit-Code beenden (Default 0) |
| `exp` | e hoch x |
| `expt` | basis hoch exp, auch negativ und gebrochen |
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
| `format-time` | Zeit als String, strftime-Direktiven (%F %T …); Ortszeit oder :utc |
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
| `json-encode` | Wert als JSON-String: Hash-Tabelle->Objekt, Liste->Array, ()->[], :null/:false |
| `json-parse` | JSON-Text lesen: Objekt->Hash-Tabelle (equal), Array->Liste, null->:null, false->() |
| `last` | letztes Element einer Liste |
| `length` | Elementzahl einer Liste bzw. Zeichenzahl eines Strings |
| `lisp-error-msg` | :msg-Slot einer Condition liefern |
| `lisp-implementation-version` | Version von golisp2 als String (CL) |
| `list` | Liste aus den Argumenten erzeugen |
| `list->string` | Stringliste zu einem String verketten |
| `list-tail` | n-ten Rest der Liste |
| `list?` | t, wenn Liste oder nil |
| `load-system` | System topologisch sortiert laden (Abhaengigkeiten zuerst) |
| `loaded-systems` | Liste der geladenen Systeme |
| `lock-make` | Mutex fuer lock erzeugen |
| `log` | natuerlicher Logarithmus, x > 0 |
| `loop` | Iteration (CL-Praxis-Kern): for/repeat/when/collect/sum/... (loop.lisp) |
| `macroexpand-1` | einen Makro-Expansionsschritt ausfuehren |
| `make-array` | Vektor der Laenge n anlegen, :initial-element (Default ()) |
| `make-hash-table` | thread-sichere Hash-Table erzeugen |
| `make-list` | Liste mit n Elementen (:initial-element) |
| `mapcar` | Funktion elementweise anwenden, Ergebnisliste (first-class, #' moeglich) |
| `maphash` | Funktion (key wert) auf jeden Hash-Eintrag anwenden |
| `max` | groesste Zahl |
| `maxima-close` | Maxima-Session beenden (maxima-close session) |
| `maxima-eval` | Ausdruck an Maxima-Session senden, Ergebnis als String (maxima-eval session expr [timeout]) |
| `maxima-open` | Maxima-Session starten (externer Prozess, Zustand bleibt erhalten) |
| `member` | Restliste ab erstem Treffer |
| `memstats` | Go-Runtime-Memory-Statistiken als Assoziationsliste |
| `min` | kleinste Zahl |
| `minusp` | CL-Name fuer negative? |
| `mod` | Rest wie CL: abgerundet, Vorzeichen des Divisors ((mod -3 2) → 1) |
| `negative?` | t, wenn kleiner 0 |
| `nil` | NIL-Konstante (Singleton, MakeNil) |
| `now` | Unix-Sekunden (seit 1970) als Float |
| `nth` | n-tes Element (0-basiert) |
| `null` | t, wenn nil |
| `null?` | Alias fuer null |
| `number->string` | Zahl in String umwandeln |
| `number?` | t, wenn Zahl |
| `oddp` | t wenn ganze Zahl n ungerade ist (CL), sonst Fehler bei Nicht-Ganzzahl |
| `pair?` | t, wenn Cons |
| `parse-float` | String strikt als Fliesskommazahl parsen, Fehler bei ungueltigem Format |
| `parse-int` | String strikt als Ganzzahl parsen, Fehler bei ungueltigem Format |
| `pg-close` | PostgreSQL-Verbindung schliessen |
| `pg-connect` | PostgreSQL-Verbindung oeffnen (Connection-String) |
| `pg-exec` | schreibendes SQL, betroffene Zeilen zurueck |
| `pg-query` | SELECT, Liste von Zeilen als Assoziationslisten |
| `plusp` | CL-Name fuer positive? |
| `pool-map` | wie mapcar, hoechstens n Aufrufe gleichzeitig; Reihenfolge bleibt, Fehler → (:error text) an der Stelle |
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
| `regex-find` | erster Regex-Treffer oder (); 2. Wert: Liste der Gruppen |
| `regex-find-all` | alle Regex-Treffer als Liste |
| `regex-match` | Regex passt irgendwo im String? (regex-match re s) → t/() |
| `regex-replace` | alle Regex-Treffer ersetzen, $1/${name} fuer Gruppen |
| `regex-split` | String an Regex-Treffern trennen |
| `register-setf-expander` | setf-Expander fuer eigenen Accessor registrieren |
| `register-setf-mutator` | setf-Mutator fuer Accessor auf Objekt mit Identitaet registrieren |
| `rem` | Rest Richtung Null wie CL, Vorzeichen des Dividenden ((rem -3 2) → -1) |
| `remainder` | Scheme-Name fuer rem |
| `remhash` | Hash-Eintrag entfernen |
| `remove` | alle Vorkommen von item entfernen (equal?) |
| `remove-duplicates` | Duplikate entfernen (erstes Vorkommen bleibt) |
| `remove-if` | Elemente entfernen, fuer die Praedikat wahr ist |
| `remove-if-not` | Elemente behalten, fuer die Praedikat wahr ist |
| `rest` | Alias fuer cdr |
| `reverse` | Liste umdrehen |
| `round` | auf naechste ganze Zahl runden, Rundung-zum-Geraden bei .5 (CL); 2 Werte: Quotient, Rest |
| `second` | Alias fuer cadr |
| `set-difference` | Elemente von a ohne die aus b |
| `set-nth` | Kopie mit n-tem Element ersetzt |
| `set-working-directory` | Arbeitsverzeichnis wechseln |
| `setf` | generalisierte Zuweisung: Variablen, (car lst), (gethash key tbl), struct-Slots |
| `shell-assoc` | passendes Paar aus Assoziationsliste suchen (Shell-Output-Parsing) |
| `shell-output` | stdout von /bin/sh -c cmd als String; Exit != 0 -> Fehler |
| `shm-alloc` | Block im Shared-Memory-Pool belegen |
| `shm-cleanup` | Shared-Memory-Pool aufraeumen |
| `shm-free` | Shared-Memory-Block freigeben |
| `shm-read` | Block (maximal n Zeichen) als String lesen |
| `shm-status` | Pool-Statistik als Assoziationsliste (total, used, free) |
| `shm-write` | String in Shared-Memory-Block schreiben |
| `signal` | Condition des definierten Typs signalisieren (:msg ...) |
| `sigo` | Prompt an sigoREST senden, Antwort als String |
| `sigo*` | Wie sigo, Ergebnis als Hash-Tabelle (text, model, finish-reason, Tokens, cost-usd, elapsed) |
| `sigo-budget` | Tageslimit und -verbrauch von sigoREST (/api/budget) als Hash-Tabelle |
| `sigo-costs` | Kosten seit/bis (Unix-Sekunden) aus /api/costs als Hash-Tabelle |
| `sigo-host` | sigoREST-Host setzen oder lesen |
| `sigo-model-info` | Modell-Infos aus /api/models (Preise USD/1M Tokens, 0 = unbekannt), alle oder per id/shortcode |
| `sigo-models` | Liste der verfuegbaren sigoREST-Modelle |
| `sigo-reference` | Eingebettete KI-Kurzreferenz (Default-Vorspann von sigo) |
| `sigo-request` | Chat-Request als Hash-Tabelle (Whitelist), Antwort als Hash-Tabelle plus elapsed |
| `sigo-system-prompt` | Vorspann fuer sigo lesen, setzen oder mit leerem String leeren |
| `sigo-usage` | Summe der sigo-Tokens, -Kosten und -Antwortzeit seit Start/Reset, plus calls |
| `sigo-usage-reset` | sigo-usage-Summen auf 0 setzen |
| `sleep` | Millisekunden pausieren |
| `slurp` | stdin komplett als String lesen |
| `sort` | Liste nach Vergleich sortieren (neue Liste) |
| `spawn` | fn ohne Argumente in neuer Goroutine starten, sofort Ergebnis-Channel mit (:ok wert) oder (:error text) |
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
| `string-join` | Liste von Strings mit Trenner (Pflicht) verbinden |
| `string-length` | Zeichenanzahl (Runes) |
| `string-replace` | alle Vorkommen ersetzen |
| `string-split` | an Trenner (woertlich, leere Felder bleiben) oder an Whitespace trennen |
| `string-trim` | fuehrende und abschliessende Leerzeichen entfernen |
| `string-upcase` | in Grossbuchstaben umwandeln |
| `string?` | t, wenn String |
| `substring` | Teilstring von start (inklusive) bis end (exklusive) |
| `subtypep` | Untertyp-Prüfung, zwei Werte: (subtypep 'integer 'number) → t t; unentscheidbar → () () |
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
| `truncate` | Richtung Null runden; 2 Werte (CL): Quotient, Rest |
| `type-of` | Typ eines Werts als Symbol (integer, string, cons, Struct-/Condition-Name …) |
| `typecase` | Verzweigung nach Typ: (typecase x (integer …) ((or string symbol) …) (otherwise …)) |
| `typep` | Typprüfung: (typep x 'number), (or …), (and …), (not …), (member …), (satisfies f), (integer lo hi) |
| `union` | Vereinigungsmenge zweier Listen |
| `unless` | Makro: body nur auswerten, wenn test falsch ist |
| `unload-system` | System aus Ladestatistik entfernen (Definitionen bleiben) |
| `untrace` | Tracing deaktivieren |
| `values` | mehrere Werte liefern; Verwender sehen den Hauptwert |
| `vector` | Vektor aus den Argumenten |
| `vector-push-extend` | Element anhaengen, liefert dessen Index |
| `warn` | Warnung ausgeben, Auswertung laeuft weiter |
| `webserv` | Web-Bootstrap: HTTP-Server plus boot.js plus Browser (:html/:htmlpath Pflicht) |
| `when` | Makro: body nur auswerten, wenn test wahr ist |
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
