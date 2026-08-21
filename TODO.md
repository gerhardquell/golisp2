# TODO - Aufgabenplanung 20260821

Vorige Runde (webserv/http-serve-Klärung, golisp2web-Grundgerüst,
Netzzugang, Darstellung, Buttonleiste, Beenden/Neustart) komplett
erledigt — archiviert unter `todos/TODO.md-20260821-done`.

epub3-Support (golisp2web-Commit `713ed8a`: lokaler epub-Tab, eigenes
QWebEngineProfile pro Tab, Scheme-Handler aus epubReader3 portiert)
ebenfalls erledigt.

## offene Ideen

- Editor-Integration (ursprünglich "pyscintella") — Rechercheergebnis
  20260821: natives QScintilla gibt es hier nur als PyQt6-Binding
  (`PyQt6-QScintilla`), keine PySide6-Bindung vorhanden. PyQt6+PySide6 im
  selben Prozess ist keine gute Idee (zwei inkompatible Qt-Bindings).
  Technisch sinnvolle Alternative: Editor-Tab wie der epub-Tab — ein
  Web-Editor (z. B. CodeMirror) als lokale HTML/JS-Datei in einem
  `QWebEngineView`, kein neues natives Binding nötig. Noch offen: Zweck
  (golisp2-REPL-Editor über die WS-Bridge vs. generischer Datei-Editor)
  — Gerhards Entscheidung bei Bedarf, noch nicht gebaut.
