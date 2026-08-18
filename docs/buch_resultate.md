# Retrospektive — Lispbuch-Überarbeitung

## Session 20260815 — Kapitel 0019 (Let over Lambda 2)

### Was hat funktioniert

- **Primitive-first-Ansatz:** Bevor ein Code-Block geschrieben wurde, ein
  Probe-Skript (`tmp/test/probe19*.lisp`) angelegt, das jede benötigte
  golisp2-Funktion durchprüft (`push`, `assoc`, `case`, `&optional`,
  `filter`, `printf`, …). So wusste ich *vor* dem Schreiben, was fehlt
  (`copy-list`, `make-list`, `remove-if-not`) und was geht. Verhinderte
  spekulativen Code.
- **Getestetes Skript als Quelle:** Erst das vollständige
  `ch19_examples.lisp` geschrieben und so lange korrigiert, bis exit 0 —
  *dann* den Code in `chapt-0019.md` eingebettet. Code im Buch ist keine
  Retting aus dem Gedächtnis, sondern Kopie eines lauffähigen Skripts.
- **Code-Extraktion als Verifikation:** Nach dem Schreiben der .md alle
  ```` ```lisp ````-Blöcke per Python-Regex extrahiert und als *ein*
  Skript mit golisp2 laufen lassen. Fing einen echten Bug: `make-list` war
  im .md *nach* seinem ersten Aufruf definiert (im Testskript stand es am
  Anfang). Ohne diesen Schritt wäre ein Reihenfolge-Fehler im Buch
  gelandet.
- **md2xhtml.py-Workflow:** xhtml via das existierende Skript statt
  pro-Kapitel-Agent (wie Memory [[md2xhtml-skript-build-agent-ersetzt]]
  schon nahelegt). Schnell, deterministisch, kein Kontext-Overhead.
- **CL-Vergleich als didaktisches Mittel:** Jede golisp2-Lösung neben das
  CL-Original zu stellen macht die Übersetzungen (`incf`→`setf`,
  `:keyword`→`(quote sym)`, `funcall`→direkter Aufruf) explizit und lehrreich.

### Was war unerwartet schwierig

- **Versteckte Fallstricke, die erst beim Testen auffielen:**
  1. `function` ist eine Special-Form — als Parametername scheitert sie
     *still* (gibt Argument statt Funktionsaufruf zurück, kein Fehler).
     memo-fib lieferte 10/11 statt 55/89 — sah auf den ersten Blick
     plausibel aus, war aber völlig falsch. Erst als die Curry-Version
     ungefiltert zurückkam, fiel es auf.
  2. `<=` ist nur 2-stellig. `(<= 0 12 10)` gibt `t` — der dritte Wert
     wird ignoriert. Bounded-Counter ließ `:set 12` durch, obwohl 12
     außerhalb der Grenzen. Wieder: kein Fehler, nur falsches Ergebnis.
  3. `setf` kann keine Places wie `(cadr entry)` setzen — erst in den
     QA-Übungen (Multi-Counter) aufgetaucht. Zwang zum funktionalen
     Ersetzen statt Mutieren.
- **`define` vs `defun`-Verwirrung:** `(define (name args) ...)` geht
  nicht, nur `(define name value)`. Funktionen brauchen `defun`. Ein
  `sed`-Versuch, das global zu ersetzen, war zu gierig und fraß Klammern —
  Datei sauber neu geschrieben. Lektion: bei Strukturänderungen Datei
  neu schreiben statt sed-Regex basteln.
- **REDEF beim stdlib-Überschreiben:** `alist-set`/`alist-get` sind
  bereits in der golisp2-stdlib. Eigene Definition → REDEF-Warnung.
  CLAUDE.md-Fallstrick 1 in Aktion — aber erst nach dem Schreiben
  bemerkt.

### Gelernte Lektionen

- **„Stiller falscher Wert" ist der gefährlichste Bug-Typ.** golisp2
  liefert bei fehlerhaften Namen/Spezialformen oft kein `ERR:`, sondern
  ein plausible aussehendes `nil` oder einen falschen Wert. Abhilfe:
  erwartete Ausgaben als Kommentare daneben schreiben (`; -> 55`) und
  beim Lauf *zeilenweise* vergleichen. Wer nur „läuft ohne Fehler" als
  Kriterium nimmt, veröffentlicht falschen Code.
- **Vor Eigenbau stdlib prüfen.** golisp2 hat mehr, als die Minimalität
  vermuten lässt (`filter`, `alist-set`, `assoc` mit `equal?`-Default).
  Ein kurzer Probe-Lauf spart Definitionen und REDEF-Kollisionen.
- **Code-Reihenfolge im Buch ≠ Code-Reihenfolge im Skript.** Das
  Testskript kann Helper oben sammeln; im Buch steht ein Helper oft erst
  beim ersten Einsatz. Extrahieren-und-laufenlassen fängt das. Merke:
  Helper-Definitionen vor erster Nutzung platzieren, auch im .md.
- **Memory als Zutaten-Datei-Substitut.** Das Projekt hat keine
  separaten „Zutaten"-Dateien, aber die persistenten Memory-Dateien
  erfüllen dieselbe Funktion. Neue Fallstricke dort abgelegt
  ([[golisp2-cl-nach-golisp2-fallstricke]]) → Kapitel 0020+ profitiert.

### Kennzahlen

- golisp2-Testläufe für Kapitel 0019: ~8 (Probe-Skripte + Beispiele +
  Bug-Isolation + QA + Extraktions-Verifikation).
- Code-Blöcke im Kapitel: 18 (golisp2) + CL-Vergleiche.
- Neue Memory-Einträge: 1 (golisp2 Fallstricke).
- Bug-Typen gefunden: Special-Form-Kollision, 2-stelliger Vergleich,
  setf-Place-Limit, fehlende Primitive, REDEF, Definitionsreihenfolge.

### Nächste Schritte

- Kapitel 0020 (`Section0020.xhtml`, 13.8K — kleiner, vermutlich
  thematisch verwandt). Gleicher Workflow: Original lesen → Primitive
  prüfen → Testskript exit 0 → .md mit CL-Vergleich → QA → xhtml.
- Bei CL-Code mit `defstruct`/`defclass`/`loop` aufpassen — das sind
  die nächsten potenziellen golisp2-Lücken.

---

## Session 20260815 — Serie-Abschluss (Kapitel 0018–0027)

Rückblick über die gesamte Let-over-Lambda-Serie (Kap18–25 aus epub),
Kapitel 26 (Metaprogrammierung, neu) und Anhang A / Kapitel 27 (golisp2
Besonderheiten, neu). Die Kapitel 18–25 wurden in früheren Sessions
überarbeitet; 26 und 27 entstanden in dieser Session.

### Was hat funktioniert

- **Probe-Workflow ist gereift.** Was in Kap19 als „Primitive-first-Ansatz"
  begann, ist nun Standard: pro Kapitel ein `probeNN*.lisp`, das jeden
  Code-Block isoliert testet (mit `trap` pro Block, kein Crash killt das
  Skript). Code-Blöcke im .md sind Kopien verifizierter Probe-Abschnitte.
  Ablauf: Original lesen → Primitive sondieren → Testskript exit 0 → .md
  schreiben → QA-Probe → xhtml → Extraktions-Verifikation → Memory →
  nachfragen. Deterministisch, wiederholbar.
- **Ehrlichkeit bei CL-Grenzen statt Fake-Port.** Kap24 (Performance) war
  CL-lastig (Disassembler, Type-Declarations, Cons-Pools, `nconc`). Statt
  unvollkommene Portierung vorzutäuschen: CL-Original zeigen, golisp2-Grenze
  *begründen* (Design-Prinzip: Interpreter, immutable cells, kein Paket-System),
  portierbare Substanz herausarbeiten. Didaktisch ehrlicher als eine
  Pseudo-Übersetzung, die beim Leser nicht läuft.
- **Neue Kapitel als logische Fortsetzung.** Kap26 (Metaprogrammierung) löste
  den Kap25-Ausblick ein; Kap27 (Anhang) krönte die Serie mit golisp2s
  Identität (KI/GA/PG/Nebenläufigkeit). Kein Cliffhanger, sondern runder
  Abschluss. Der Round-Trip in Kap26 (`(* (+ 2 3) 4)` → `(2 3 + 4 *)` → 20)
  verband Kap25 (Stack) mit Kap26 (Compiler) — die Serie hat *Kohärenz*.
- **Memory als wachsende Wissensbasis.** Die Fallstricke-Datei
  ([[golisp2-cl-nach-golisp2-fallstricke]]) wuchs auf 21 Einträge; für Kap27
  kamen zwei neue Dateien dazu ([[golisp2-anhang-scope]],
  [[golisp2-anhang-primitive]]). Jedes Kapitel profitierte von den Lektionen
  der vorigen — der „stille falsche Wert"-Bug aus Kap19 wurde durchgehend
  durch erwartete Ausgaben (`; -> 55`) abgefangen.
- **Scope-Disziplin (Gerhards Korrektur).** Anhang sollte ursprünglich
  sigoREST-Setup enthalten. Gerhard stellte klar: nur golisp2-Nutzung, Setup
  = Anwender via github. Das hielt den Anhang fokussiert und verhinderte
  ein ausuferndes Kapitel über externe Dienste. Lektion in
  [[golisp2-anhang-scope]] festgehalten.

### Was war unerwartet schwierig

- **Performance-Primitive systematisch fehlend (Kap24).** Nicht einzelne
  Lücken, sondern eine ganze *Klasse*: `time`, `get-internal-real-time`,
  `sort`, `max`/`min`, `nconc`/`nreverse`, `defstruct`, `disassemble`,
  `declare`/`optimize`. Performance-Vergleiche ohne Zeitprimitive → auf
  Aufrufzähler (`define-counted` via `intern`) als Proxy ausweichen.
  Echte TList unmöglich (immutable cells, kein `setf (cdr ...)`) →
  reversed-accumulator. Lehrt: golisp2 ist bewusst *kein* Performance-Lisp.
- **`ga-create`-Signatur in CLAUDE.md falsch (Kap27).** CLAUDE.md nannte
  `ga-create` ohne Arg-Zahl; Sondierung ergab 4 Argumente
  `(type gen-len gen-par fitness-fn)`. `ga-cross` braucht `points < gen-len`
  (sonst „codist out of range"). Hätte ich CLAUDE.md vertraut, wäre der
  Code falsch. CLAUDE.md irrt auch beim Fehlermodell (`catch` = dynamischer
  Sprung, *kein* error-handler; `trap` ist der handler — siehe
  [[golisp2-catch-vs-trap-fehlermodell]]).
- **Channel-Deadlock im Single-Thread (Kap27).** `(chan-make)` ungepuffert →
  `chan-send` blockiert bis Empfänger bereit. In einem linearen Skript
  Deadlock; erstes Probe-Skript timeoutete (EXIT 124). Lösung: gepufferter
  Channel `(chan-make 1)` oder `parfunc` mit Producer/Consumer. Go-Semantik,
  die man kennen muss.
- **`sigo`-Kürzeldrift (Kap27).** CLAUDE.md nennt `"claude-h"`; das Kürzel
  existiert nicht mehr (sigoREST lädt Modelle dynamisch). `(sigo-models)`
  liefert 204 Einträge mit Duplikaten (kanonisch + Alias). Wahrheit ist
  zur Laufzeit abzufragen, nie hart zu kodieren.
- **`pg-connect` schweigt in CLAUDE.md (Kap27).** golisp2 hat *native*
  PostgreSQL-Primitive — CLAUDE.md erwähnt sie nicht. Vollständig
  sondieren müssen: `pg-connect/query/exec/close`, Alist-Results,
  libpq-env-Defaults (`pg-connect ""` funktioniert über `PGHOST` etc.).
- **Triple-nested Quasiquote (Kap26).** CLs `once-only` braucht 3-Level-
  Backquote; golisp2s Backquote ist bis 2 Level sicher, triple fehleranfaellig.
  Ausweichen auf `gensym`+`let` manuell. Grenzt die Makro-Mächtigkeit ein,
  ist aber für die Praxis ausreichend.

### Gelernte Lektionen

- **CLAUDE.md ist Startpunkt, nicht Wahrheit.** Die golisp2-CLAUDE.md
  dokumentiert viel, irrt aber bei Details (GA-Signatur, Fehlermodell,
  fehlende Primitive). Probe-Skripte sind die *eigentliche* Wahrheitsquelle.
  Niemals Code aus CLAUDE.md-Erinnerung schreiben ohne Verifikation.
- **„Ehrlich begründen" schlägt „vollständig portieren".** Wo golisp2 eine
  Grenze hat (Disassembler, Type-Decls, Pakete), ist die didaktisch beste
  Antwort: Grenze zeigen, *warum* begründen (Design-Philosophie),
  Alternative geben. Leser lernt mehr aus einer ehrlichen Grenze als aus
  einer unvollkommenen Nachahmung. CLAUDE.md-Regel 1 („nicht annehmen,
  nicht vermuten") in Reinform.
- **Kapitel-Kohärenz durch Querverweise.** Kap26 nutzt `postfix-eval` aus
  Kap25; Kap27 GA nutzt `parfunc` aus Abschnitt E desselben Kapitels;
  Anhang PG nutzt Alist-Idiom aus Kap21. Explizite Querverweise
  („wie in Kapitel 25 gezeigt") machen die Serie zu einem *Gewebe*, nicht
  isolierten Einzelstücken.
- **Scope-Korrekturen ernst nehmen.** Gerhards Einwand „sigoREST-Setup ist
  nicht golisp2-Thema" kam spät, korrigierte aber den ganzen Abschnitt B.
  Sofort in Memory ([[golisp2-anhang-scope]]) fixiert, damit künftige
  Kapitel denselben Fehler nicht wiederholen. Rückmeldungen sind nicht
  Kritik, sondern Kurskorrektur —立即 umsetzen.

### Zutaten-Datei: Empfehlungen

- **CLAUDE.md golisp2-Sektion korrigieren** (Gerhard entscheidet):
  1. `ga-create`-Signatur auf 4 Argumente
     `(ga-create type gen-len gen-par fitness-fn)` präzisieren; `ga-cross
     points < gen-len`, `ga-mut rate`, `ga-print lines` ergänzen.
  2. Fehlermodell: `trap` als error-handler klarstellen (nicht `catch`);
     `catch`/`throw` als dynamischen Kontrollsprung bezeichnen (Memory
     [[golisp2-catch-vs-trap-fehlermodell]]).
  3. `pg-connect`/`pg-query`/`pg-exec`/`pg-close` als native PG-Primitive
     aufnehmen (derzeit fehlen sie völlig).
  4. `sigo`-Kürzel `"claude-h"` als veraltet markieren; auf
     `(sigo-models)` als Wahrheitsquelle verweisen.
  5. System-Primitive ergänzen: `system`/`getenv`/`environ`/`file-exists?`/
     `memstats`; Nebenläufigkeit `parfunc`/`chan-make`/`lock-make`.

### Kennzahlen Serie 18–27

- Kapitel überarbeitet/neu: 10 (18–27), davon 2 neu (26, 27).
- Code-Blöcke insgesamt: ~80 golisp2 + CL-Vergleiche.
- Memory-Einträge: 4 (Fallstricke 21 Einträge, catch-vs-trap,
  anhang-scope, anhang-primitive, md2xhtml-Workflow).
- Bug-Klassen dokumentiert: Special-Form-Kollision, 2-stelliger Vergleich,
  setf-Place-Limit, immutable cells, chan-Deadlock, Kürzeldrift,
  CLAUDE.md-Fehler (GA/Fehlermodell/PG).
- Alle 10 Kapitel: Extraktions-Verifikation EXIT 0.

### Nächste Schritte

- **Buch ist rund.** Serie 18–27 (Let over Lambda + Metaprogrammierung +
  Anhang) bildet einen geschlossenen Bogen. Kein offener Faden.
- **Optional:** Zutaten-Datei-Updates (oben) in CLAUDE.md einpflegen, falls
  Gerhard das wünscht — das würde künftige golisp2-Arbeit (nicht nur Buch)
  vor denselben Sondierungs-Schleifen bewahren.
- **Aufräumen:** `tmp/` enthält Probe-Skripte (probe18–27) und Roh-Extrakte
  (sect18–25_raw.txt). Können erhalten (Referenz) oder geräumt werden.
  `tmp/epub25` (falls noch vorhanden) kann weg.
