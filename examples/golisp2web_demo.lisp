#!/bin/env golisp2
;; ***********************************************************************
;;  examples/golisp2web_demo.lisp
;;  Autor    : Gerhard Quell - gquell@skequell.de
;;  CoAutor  : Claude Sonnet 5
;;  Copyright: 2026 Gerhard Quell - SKEQuell
;;  Erstellt : 20260826
;; ***********************************************************************
;; Control-Panel-Demo fuer golisp2s Web-Bridge (webserv/ws-export/ws-emit,
;; siehe lib/webserv.go, lib/wsbridge.go) UND fuer golisp2web (PySide6-
;; GUI-Client, eigenes Repo golisp2web/):
;;
;;   1. Textinput + Button -> ws-export "greet"      (RPC mit Argument)
;;   2. Tabelle             -> ws-export "table-data" (RPC, strukturierte
;;                              Antwort, Client rendert als <table>)
;;   3. Neuer Tab            -> ws-export "open-tab" -> ws-emit
;;                              golisp2web-new-tab (siehe golisp2web/lib/
;;                              quitBridge.py) -- oeffnet einen zweiten Tab
;;                              auf demselben Server
;;   4. golisp2web beenden   -> ws-export "quit-app" -> ws-emit
;;                              golisp2web-quit + http-stop
;;
;; golisp2web-* Events wirken NUR in golisp2web (dessen mainWindow.py
;; registriert den JS-Listener per QWebChannel) -- ein normaler Browser
;; (webserv :open t / browser-open) reagiert nicht darauf. Deshalb startet
;; dieses Skript golisp2web selbst.
;;
;; exec statt system: exec braucht timeout: -1 (sonst killt der Default
;; von 60s, lib/eval_exec.go, das Fenster mitten in der Sitzung -- siehe
;; Commit c18e600). exec kann kein "cd", darum absoluter Pfad zur lokalen
;; golisp2web.py per argv-Selbstlokalisierung (Muster: examples/parvmira-
;; web/server.lisp) -- zeigt IMMER den aktuellen Stand dieses Checkouts,
;; nicht ein evtl. veraltetes /usr/local/lib/golisp2web. python3 setzt
;; sys.path[0] selbst auf das Skriptverzeichnis, kein cwd-Trick noetig.
;;
;; exec blockiert (anders als "system ... &" vorher) -- darum parfunc:
;; Branch 1 wartet auf golisp2web, Branch 2 auf den Server (http-wait).
;; ws-export-Registrierung bewusst VOR dem parfunc, nicht in Branch 2 --
;; sonst Race: golisp2web koennte RPC-Calls schicken, bevor die Handler
;; registriert sind (paralleler Start beider Branches, keine Reihenfolge
;; garantiert).
;;
;; Aufruf vom Repo-Root (braucht echtes X11-Display, golisp2web bindet
;; QT_QPA_PLATFORM hart auf xcb -- kein Headless-Betrieb):
;;   ./examples/golisp2web_demo.lisp
;; oder: ./build/golisp2 examples/golisp2web_demo.lisp
;;
;; Sauberes Ende: den "golisp2web beenden"-Button in der Seite klicken --
;; der stoppt golisp2web UND den Server (http-stop), beide parfunc-
;; Branches kehren zurueck, das Skript endet. Schliesst man golisp2web
;; stattdessen ueber das Fenster-X, bleibt Branch 2 (http-wait) haengen
;; (kein Timeout) -- dann Ctrl-C im Terminal (http-wait beendet den
;; Prozess sauber bei SIGINT/SIGTERM, siehe lib/httpserver.go).
;; ***********************************************************************

(defun letzter-index (s zeichen)
  (let ((n (string-length s)) (i -1) (found -1))
    (while (< (+ i 1) n)
      (set! i (+ i 1))
      (if (equal? (substring s i (+ i 1)) zeichen)
          (set! found i)))
    found))

(defun eltern-verzeichnis (pfad)
  (let ((idx (letzter-index pfad "/")))
    (if (< idx 0) "." (substring pfad 0 idx))))

;; NICHT zweifach eltern-verzeichnis schachteln, um zur Repo-Root
;; hochzugehen -- bricht auf "." zusammen, wenn argv nur ein Pfadsegment
;; hat (z. B. Aufruf per "cd examples && ./golisp2web_demo.lisp"). Ein
;; woertliches ".." anhaengen und dem Betriebssystem die Aufloesung
;; ueberlassen ist robust gegen jede Aufrufform (Repo-Root, examples/,
;; absoluter Pfad) -- exec braucht dafuer kein filepath.Clean, der
;; Kernel loest ".." unabhaengig vom cwd korrekt auf.
(define golisp2web-py
  (string-append (eltern-verzeichnis (cadr (argv)))
                  "/../golisp2web/golisp2web.py"))

(define demo-html "<!doctype html>
<html>
<head><meta charset='utf-8'><title>golisp2 Control-Panel-Demo</title></head>
<body>
  <h1>golisp2 Control-Panel-Demo</h1>

  <section>
    <h2>1. Textinput + Button</h2>
    <input id='nameInput' type='text' placeholder='Dein Name' value='Welt'>
    <button id='greetBtn'>Gruessen</button>
    <p id='greeting'></p>
  </section>

  <section>
    <h2>2. Tabelle</h2>
    <button id='tableBtn'>Tabelle laden</button>
    <table id='dataTable' border='1'></table>
  </section>

  <section>
    <h2>3. Neuer Tab</h2>
    <button id='newTabBtn'>Neuen Tab oeffnen</button>
  </section>

  <section>
    <h2>4. golisp2web beenden</h2>
    <button id='quitBtn'>golisp2web beenden</button>
  </section>

  <script>
    window.golisp.ready.then(function () {

      document.getElementById('greetBtn').onclick = function () {
        var name = document.getElementById('nameInput').value;
        golisp.call('greet', name).then(function (msg) {
          document.getElementById('greeting').textContent = msg;
        });
      };

      document.getElementById('tableBtn').onclick = function () {
        golisp.call('table-data').then(function (rows) {
          var tbl = document.getElementById('dataTable');
          tbl.innerHTML = '';
          rows.forEach(function (row) {
            var tr = document.createElement('tr');
            row.forEach(function (cell) {
              var td = document.createElement('td');
              td.textContent = cell;
              tr.appendChild(td);
            });
            tbl.appendChild(tr);
          });
        });
      };

      document.getElementById('newTabBtn').onclick = function () {
        golisp.call('open-tab');
      };

      document.getElementById('quitBtn').onclick = function () {
        golisp.call('quit-app');
      };
    });
  </script>
</body>
</html>")

(define s (webserv :port 8090 :host "127.0.0.1" :html demo-html :open nil))

(println (format nil "golisp2web_demo: Server auf Port ~a, starte golisp2web..." (http-port s)))

(ws-export s "greet" (lambda (c name) (string-append "Hallo, " name "!")))

(ws-export s "table-data" (lambda (c)
  (list (list "Sprache" "Erschienen")
        (list "Lisp" "1958")
        (list "Go" "2009")
        (list "golisp2" "2026"))))

(ws-export s "open-tab" (lambda (c)
  (ws-emit s "golisp2web-new-tab" (format nil "127.0.0.1:~a" (http-port s)))
  t))

(ws-export s "quit-app" (lambda (c)
  (ws-emit s "golisp2web-quit" ())
  (http-stop s)
  t))

(parfunc ret
  (exec "python3" param: golisp2web-py
                   param: "-t1" param: (format nil "localhost:~a" (http-port s))
                   timeout: -1)
  (http-wait s))

(println "golisp2web_demo: golisp2web beendet, Server gestoppt.")
