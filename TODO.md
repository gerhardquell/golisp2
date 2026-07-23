# Aufgabe 20260723

## Aufgabe:
Langfristig wollen wir golisp2 clisp-konform machen. dazu werden wir einen teil direkt in golisp2 codieren und den anderen teils als macros definieren. Was meinst du zu diesem vorgehen?

## Vorgehen. Ist klassischer Weg — genau so bauen echte Lisps sich selbst: kleiner Kern + Bootstrap-Bibliothek. Meine Einschätzung:

### Was in Go gehört (Kern):
- Alles, was env braucht oder nicht expandierbar ist: Spezialformen (quote, if, let, lambda, setq, block/tagbody/go, unwind-protect, multiple-value-*)
- Reader/Printer, Zahlen-Turmes ist nicht CL)
- Alles Performance-kritische im heißen Pfad

### Was als Makros gehört (Bibliothek):
- Alles Ableitbare: when, unlefclass-Subset, incf/decf,push/pop, destructuring-bind
- Vorteil: testbar in Lisp, pweiternd — passt zurProjekt-Philosophie

## 1.Schritt:

1. Wir analysieren CLISP und unterteilen die 978 Symbole in Kern und Macro.
2. Wir bauen eine scharfe Konformitäts-Testsuite für die Kernsymbole.
3. Wir bauen ein golisp2 mit den neuen Kernsymbolen, so sicher und robust wie möglich.
4. Wir testen unser neues golisp2 mit allen Funktionen. 

## Aber drei Warnungen Beachten! :

1. CL ist ein Kontinent, kein Land. Full-ANSI = CLOS, Condition-System, Reader-Makros, format komplett, 978 Symbole.„CLISP-kompatibel für SkripteX, Y, Z" statt „ANSI komplett". 

2. Makro braucht Kern darunter zu tagbody/go — wenn Kern das nicht sauber kann, wackelt alles drüber. Reihenfolge: Kern-Spezialformen zuerst machen, dann Makro-Schicht.

3. Namenskollision-Regel von CLAUDE.md gilt doppelt. Makro mit Namen einer Spezialfor= toter Code, feuert nie. Püfen, was Spezialform vs. Primitiv vs. frei ist.                                                 
