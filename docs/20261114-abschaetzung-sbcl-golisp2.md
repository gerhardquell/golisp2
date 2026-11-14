# Abschätzung: SBCL 2.6.10 und golisp2
Gerhard Quell / Claude Opus 5.5 – 20261114
Aufgabe: TODOS.md vom 20261114. Zielsystem: Debian 12, x86-64.

Das ist eine Abschätzung. Werte mit „gemessen“ wurden heute gemessen, alle anderen sind geschätzt.

---

## Kurzantwort

| Frage | Antwort |
|---|---|
| 1. Ganz SBCL optimieren? | Lohnt sich nicht. Der relevante Code hat rund 455.000 Zeilen, an denen seit 25 Jahren Fachleute feilen. Nur gezielte Punkte sind sinnvoll. |
| 2. golisp2-Eigenschaften in SBCL? | Ja, aber als **CL-Bibliothek**, nicht im SBCL-Quellcode. Fast jede Eigenschaft hat ein fertiges Gegenstück. |
| 3. golisp2-Programme portieren? | Ja. Möglich sind ein Übersetzer golisp2 → CL oder eine Kompatibilitätsschicht. Die größte Hürde ist Lisp-1 gegenüber Lisp-2. |
| 4. Compilierung nutzen? | Ja, über den Weg „golisp2-Code → CL → SBCL-Compiler“. Gemessen: rund **100-mal schneller**. Den SBCL-Compiler nach Go zu übernehmen ist nicht machbar. |

---

## 1. Messwerte (gemessen, 20261114)

Testprogramm: `fib 30`, naiv rekursiv. Die Dateien liegen in `tmp/fib.lisp` und `tmp/fib-cl.lisp`.

| System | Zeit | Faktor |
|---|---:|---:|
| golisp2 0.24.1 (Interpreter, Go) | 1,22 s | 1 |
| SBCL 2.6.5, Interpreter `sb-eval` | 1,44 s | 0,85 |
| SBCL 2.6.5, compiliert, ohne Deklarationen | 0,012 s | ≈ 100 |
| SBCL 2.6.5, compiliert, `fixnum` + `speed 3` | 0,008 s | ≈ 150 |

Erkenntnisse:
- golisp2 ist als Interpreter **gleich schnell wie der SBCL-Interpreter**. Dort ist golisp2 also nicht schlecht.
- Den Faktor 100 bringt allein der **native Compiler**.
- Startzeit: golisp2 < 0,01 s, SBCL ≈ 0,02 s. Beide sind schnell genug.

Hinweis: Das ist nur ein Benchmark, und er misst nur Funktionsaufrufe und Ganzzahlen. Bei Strings, Hashtabellen oder KI-Aufrufen ist der Faktor kleiner. Bei KI-Aufrufen bestimmt das Netz die Zeit, nicht die Sprache.

---

## 2. Größe von SBCL 2.6.10 (gemessen)

| Teil | Zeilen gesamt | davon für x86-64/Linux nötig |
|---|---:|---:|
| `src/compiler` Kern | 96.250 | 96.250 |
| `src/compiler/x86-64` | 31.574 | 31.574 |
| `src/compiler/generic` | 13.129 | 13.129 |
| `src/compiler` andere CPUs (arm, arm64, ppc, riscv, …) | 101.987 | 0 |
| `src/code` (Laufzeitbibliothek in Lisp) | 231.468 | ≈ 231.000 |
| `src/runtime` (C: GC, Signale, Start) | 59.804 | ≈ 45.000 |
| `src/pcl` (CLOS) | 23.863 | 23.863 |
| `src/assembly`, `src/cold`, `src/interpreter` | 21.693 | ≈ 13.600 |
| **Summe `src/`** | **≈ 580.000** | **≈ 455.000** |
| zusätzlich `contrib/` | 70.445 | nach Bedarf |
| zusätzlich `tests/` | 113.491 | – |

Den Code der anderen CPUs zu entfernen macht SBCL **nicht schneller**, denn er wird auf x86-64 ohnehin nicht übersetzt. Es spart nur Lesearbeit.

---

## 3. Frage 1: Aufwand, den gesamten Quellcode zu optimieren

| Schritt | Aufwand (geschätzt) | Bemerkung |
|---|---|---|
| 455.000 Zeilen lesen und verstehen | 6–12 Personenmonate | Compilerbau-Wissen ist Voraussetzung (IR1, IR2, VOPs, Registervergabe). |
| Bewerten, was besser geht | weitere Monate | Fachleute haben den Code über 25 Jahre getunt. |
| Ändern, bauen, testen | je Änderung Stunden bis Tage | Selbst-Bootstrap: SBCL baut sich selbst. Die Testsuite hat 113.000 Zeilen. |
| Erwarteter Gewinn | klein | Einzelne Prozent in Sonderfällen |

**Bessere Alternative:** gezielt vorgehen.
- Die Datei `OPTIMIZATIONS` im SBCL-Baum listet 25 offene, bekannte Optimierungsideen. Dort fängt man an, wenn man an SBCL arbeiten will.
- Eigene Programme mit `sb-sprof` (statistischer Profiler, contrib) messen und nur die heißen Stellen ansehen.

**Empfehlung:** SBCL nicht optimieren, sondern **benutzen**.

---

## 4. Frage 2: golisp2-Eigenschaften in SBCL

Nichts davon gehört in den SBCL-Quellcode. Alles lässt sich als Lisp-Bibliothek (ASDF-System) über SBCL legen.

| golisp2 | Gegenstück in SBCL / Common Lisp | Aufwand (geschätzt) |
|---|---|---|
| `sigo`, `sigo-models` (sigoREST) | HTTP-Client (dexador oder drakma) + JSON (jzon) | 1–2 Tage |
| `parfunc`, `pool-map` | `sb-thread` oder lparallel | 1–2 Tage |
| `chan-make/send/recv` | `sb-concurrency:mailbox` (in SBCL enthalten) | Stunden |
| `lock-make`, `lock` | `sb-thread:make-mutex`, `with-mutex` | Stunden |
| PostgreSQL-Primitive | postmodern | 1 Tag |
| Regex | cl-ppcre | Stunden |
| HTTP-Server, Webserv, WebSocket | hunchentoot, hunchensocket | 2–4 Tage |
| Genetische Algorithmen (`ga-*`) | eigene Portierung aus `genalg.go` | 2–3 Tage |
| SWANK-Server | in SBCL der **Original-SWANK** von SLIME | 0 |
| `trace`, `redef-log` | `trace` und Redefinition-Warnungen sind eingebaut | 0 bis Stunden |
| FORMAT-Engine | CL-`format` ist eingebaut | 0 |
| `shm` (Shared Memory) | `sb-posix` mmap | 1–2 Tage |
| **Summe Kompatibilitätsschicht** | | **≈ 2–4 Wochen** |

**Umgekehrt (golisp2 → SBCL beisteuern):** In SBCL selbst fehlt nichts, was golisp2 hat. Sinnvoll ist ein eigenes, veröffentlichtes System, zum Beispiel `sigo-cl` (KI-Anbindung für CL). Das nützt auch anderen.

---

## 5. Frage 3: golisp2-Programme nach SBCL portieren

Beide Sprachen benutzen S-Ausdrücke. Darum kann der CL-Reader golisp2-Code großteils direkt lesen. Die Unterschiede liegen in der Bedeutung:

| Punkt | golisp2 | Common Lisp | Lösung |
|---|---|---|---|
| Namensräume | **Lisp-1** (`(define g f)`, `(apply + …)`) | **Lisp-2** (`#'f`, `funcall`) | Übersetzer muss Funktionspositionen erkennen. **Größte Hürde.** |
| Zahlen | nur `float64` (`(/ 10 4)` → 2.5) | Ganzzahl, Bruch, Bignum (`5/2`) | Übersetzer oder Kompat-Funktionen; Ergebnisse können abweichen |
| Definitionen | `define`, `set!`, `begin`, `while` | `defparameter`/`let`, `setf`, `progn`, `loop` | einfache Makros |
| Gleichheit | `equal?` | `equal` | Makro / Alias |
| Ausgabe | `println`, `print` (ohne Umbruch) | `print`, `format` | Kompat-Funktionen |
| Gleiche Namen, andere Bedeutung | `last`, `round`, `catch`, `eq` bei Zahlen | CL-Bedeutung | Paket mit eigenen Symbolen (shadowing) |
| Laufzeit-Code von der KI | `(eval (read (sigo …)))` | geht genauso; `eval` compiliert sogar | – |

Zwei Wege:

| Weg | Aufwand (geschätzt) | Für |
|---|---|---|
| A: Paket `golisp2-kompat` in CL (Makros + Funktionen); golisp2-Datei mit kleinen Handänderungen laden | 2–4 Wochen (siehe Abschnitt 4) | wenige, wichtige Programme |
| B: automatischer Übersetzer golisp2 → CL (in CL geschrieben, arbeitet auf S-Ausdrücken) | weitere 2–6 Wochen für die häufig benutzte Teilmenge | viele Programme, laufende Weiterentwicklung in golisp2 |

Weg B braucht Weg A als Laufzeitbasis.

---

## 6. Frage 4: Die Compilierung von SBCL nutzen

```
             Weg A (empfohlen)                       Weg C (Ideen übernehmen)
  golisp2-Code ──► Übersetzer ──► CL-Code        golisp2-Code ──► Vor-Analyse ──► Go-Closures
                                    │                                               │
                                    ▼                                               ▼
                            SBCL-Compiler                                  golisp2-Eval (Go)
                                    │
                                    ▼
                          x86-64-Maschinencode
                          (gemessen ≈ 100× schneller)       (geschätzt 2–5× schneller)
```

| Weg | Beschreibung | Machbarkeit | Gewinn |
|---|---|---|---|
| **A** | golisp2 → CL übersetzen, SBCL compiliert | gut (Abschnitt 5) | ≈ 100× (gemessen bei fib) |
| B | SBCL-Compiler nach Go übernehmen | **nicht machbar**: rund 140.000 Zeilen Compiler, er braucht die gesamte CL-Laufzeit (CLOS, Conditions, GC) | – |
| **C** | Ideen aus SBCL in golisp2 einbauen: **`sb-fasteval`** (`src/interpreter/`, 4.700 Zeilen) analysiert jede Form einmal vorab und cached Makro-Expansionen. Auf golisp2 übertragen heißt das: Form → Go-Closure statt Baum-Durchlauf bei jedem Aufruf. | gut, der Entwurf ist lesbar und überschaubar | geschätzt 2–5×, nicht gemessen |
| D | golisp2 → Go-Quelltext → `go build` | möglich, aber ohne SBCL-Nutzen | geschätzt 10–50× |
| E | Go-Bibliothek und SBCL in **einem** Prozess (FFI, `sb-alien`) | **nicht empfohlen**: beide Laufzeiten nutzen Signale und eigene GCs und stören sich | – |

Weitere Ideen aus SBCL, die golisp2 als Interpreter nutzen kann:
- **Ganzzahl-Typ** neben `float64` (SBCL: fixnum). Das bringt Korrektheit (`5/2`, große Zahlen) und Tempo.
- **Constant Folding**: `(+ 1 2)` schon beim Einlesen ausrechnen (`OPTIMIZATIONS`, `srctran.lisp`).
- **Makro-Expansions-Cache** (`sb-fasteval`).

---

## 7. Homoikonizität bei compiliertem SBCL-Code

Kurz: Die Homoikonizität bleibt in SBCL vollständig erhalten, und zwar **bis zum Compilieren**. Danach ist die Funktion Maschinencode. Ihre Listenform liegt dann nur noch als Kopie daneben, oder sie fehlt ganz.

```
Text ──read──► Liste ──macroexpand──► Liste ──compile──► Maschinencode
               └──────── hier gilt "Code = Daten" ────────┘   └ hier nicht mehr
```

| Phase | Code ist Daten? | Bedeutung |
|---|---|---|
| `read` | ja | Ergibt eine echte Liste, genau wie in golisp2. |
| Makros | ja | Makros arbeiten auf Listen, **vor** dem Compilieren. Also kostet die Homoikonizität zur Laufzeit nichts. |
| `eval` / `compile` zur Laufzeit | ja | Der Compiler ist Teil jedes SBCL-Programms. Eine neue Liste wird sofort Maschinencode. |
| fertige Funktion | **nein** | Ein Funktionsobjekt mit x86-64-Code. Den Code sieht man mit `disassemble`. |

### Gemessen (20261114, SBCL 2.6.5, Testdatei `tmp/homo.lisp`)

| Fall | Gibt `function-lambda-expression` die Quelle zurück? |
|---|---|
| `defun` in der REPL oder per `load` einer `.lisp`-Datei | ja: `(LAMBDA (X) (BLOCK F (* X 2)))` |
| interpretiert (`sb-eval`) | ja |
| aus `compile-file`, also einer FASL-Datei | **nein: `NIL`** |
| `(compile nil (read-from-string "(lambda …)"))` | geht. Ergebnis: echter Maschinencode, rund 4 ms für eine kleine Funktion. |
| `(macroexpand-1 '(when a b c))` | `(IF A (PROGN B C))` |

### Vergleich mit golisp2

| | golisp2 (Interpreter) | SBCL (Compiler) |
|---|---|---|
| Woraus wird ausgeführt? | Aus der Liste selbst. Der Rumpf eines Lambdas bleibt eine Liste. | Aus Maschinencode. Die Liste ist dann nur noch eine Kopie. |
| Die Liste nachträglich ändern | würde direkt wirken (ein Zugriff dafür ist in golisp2 aber nicht veröffentlicht, ich fand keinen) | wirkt **nicht**, die Funktion muss neu compiliert werden |
| Neu definieren mit `defun` | wirkt sofort | wirkt sofort für alle Aufrufe über den Namen. Ausnahmen: `inline`-Funktionen und Block-Compilierung. |
| Selbsterweiterndes Muster `(eval (read (sigo …)))` | geht | geht genauso, und das Ergebnis ist **nativer Code** |

### Was das für uns heißt
- Das selbsterweiternde Muster ist mit SBCL **kein Problem**, im Gegenteil: KI-erzeugter Code läuft dort compiliert, also rund 100-mal schneller.
- Wer einer Funktion zur Laufzeit ihre Quelle ablesen will (Inspektor, KI soll eine Funktion verbessern), kann sich bei SBCL **nicht** auf `function-lambda-expression` verlassen. Code aus FASL-Dateien liefert `NIL`. Die Lösung ist einfach: Die Quelle beim Definieren selbst ablegen, zum Beispiel mit einem eigenen `defun*`, das die Form in einer Hashtabelle speichert. Das sollte in die geplante Schicht `golisp2-kompat`.
- Einen Übersetzer golisp2 → CL macht die Homoikonizität leicht: Er wandelt Listen in Listen um und ist im Kern ein großes Makro.

---

## 8. Empfehlung

1. **SBCL als Werkzeug einsetzen, nicht umbauen.** Wo Rechenleistung zählt, ist SBCL rund 100-mal schneller als jeder Interpreter.
2. **golisp2 bleibt das Labor**: KI, Nebenläufigkeit, schnelles Ausprobieren.
3. **Brücke bauen, in kleinen Schritten:**
   1. `sigo-cl`: KI-Anbindung für SBCL (1–2 Tage). So zeigt sich früh, ob CL im Alltag passt.
   2. `golisp2-kompat`: die Kompatibilitätsschicht (2–4 Wochen).
   3. Erst wenn sich 1 und 2 bewähren: den Übersetzer golisp2 → CL bauen.
4. **Unabhängig davon für golisp2:** die Closure-Vorübersetzung nach dem Muster von `sb-fasteval` prüfen (Weg C).

---

## 9. Fazit (mit Gerhard abgestimmt, 20261114)

| Aussage | Bewertung |
|---|---|
| Für unsere Experimente bleiben wir bei golisp2. | **Ja.** KI, Nebenläufigkeit und schnelles Ausprobieren sind in golisp2 eingebaut. Dafür braucht man keinen Compiler. |
| golisp2 braucht ganze Zahlen. | **Ja.** Das ist vor allem eine Frage der Korrektheit. `float64` rechnet Ganzzahlen nur bis 2^53 (rund 9 Billiarden) exakt, darüber rundet es still. |
| Im Interpretermodus sind golisp2 und SBCL etwa gleich schnell. | **Ja, mit Vorbehalt.** Gemessen ist nur fib (1,22 s zu 1,44 s). Andere Lasten sind nicht gemessen. |
| Homoikonizität funktioniert in compiliertem SBCL-Code schlechter. | **Nur zum Teil.** Siehe unten. |

**Homoikonizität in SBCL:** Sie funktioniert voll. Nur die Quelle steckt nicht mehr in der fertigen Funktion, man muss sie selbst aufheben.
- Diese Teile gehen in SBCL **genauso gut** wie in golisp2: Code als Liste lesen, erzeugen und verändern; Makros; `(eval (read (sigo …)))`. Das Neu-Compilieren passiert dabei automatisch, in ein paar Millisekunden.
- Schwächer ist nur dies:
  1. Die Quelle einer fertigen Funktion zurückholen. Aus FASL-Dateien kommt `NIL`. Abhilfe: die Quelle beim Definieren selbst speichern.
  2. Die Rumpf-Liste einer laufenden Funktion ändern und die Änderung sofort wirken lassen. Stattdessen ändert man die Liste und definiert die Funktion neu.

**Wann SBCL interessant wird:** Sobald ein Experiment viel rechnen muss (zum Beispiel genetische Algorithmen mit großen Populationen oder Numerik). Dort ist compilierter SBCL-Code rund 100-mal schneller, und die Brücke aus Abschnitt 8 lohnt sich.

---

## Offen / nicht geprüft
- Gemessen wurde nur `fib`. Es fehlen Benchmarks mit Strings, Listen und Hashtabellen.
- SBCL 2.6.10 wurde nicht aus dem Quellcode gebaut. Gemessen wurde mit dem installierten SBCL 2.6.5.
- Den Gewinn von Weg C (2–5×) und Weg D (10–50×) habe ich nur geschätzt, nicht gemessen.
- Ich habe nicht geprüft, wie golisp2 sich verhält, wenn man den Rumpf einer Lambda-Liste wirklich verändert.
- Die 4 ms für `compile` gelten nur für eine kleine Funktion. Große Funktionen brauchen länger.
- Ich habe nicht geprüft, wie viel des golisp2-Sprachumfangs (`src/embed/ki-referenz.md`) sich automatisch übersetzen lässt.
