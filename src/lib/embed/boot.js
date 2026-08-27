//**********************************************************************
//  lib/embed/boot.js
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// GoLisp2 Web-Bridge Client-Bootstrap (Spec TODO.md §7).
// Ausgeliefert unter /_golisp/boot.js, per //go:embed eingebettet.
// WS-URL aus location.host abgeleitet — Reverse-Proxy-sicher.
//**********************************************************************
(function () {
  "use strict";

  var ws = null;
  var nextId = 1;
  var pending = {};    // call-id -> {resolve, reject}
  var handlers = {};   // event -> [fn]
  var readyResolve;
  var readyFired = false;

  var golisp = {
    connected: false,
    ready: new Promise(function (resolve) { readyResolve = resolve; })
  };

  // golisp.call(op, ...args) → Promise; rejected bei err oder Verbindungsabbruch
  golisp.call = function (op) {
    var args = Array.prototype.slice.call(arguments, 1);
    return new Promise(function (resolve, reject) {
      if (!golisp.connected) {
        reject(new Error("golisp: nicht verbunden"));
        return;
      }
      var id = nextId++;
      pending[id] = { resolve: resolve, reject: reject };
      ws.send(JSON.stringify({ id: id, op: op, args: args }));
    });
  };

  // golisp.on(event, fn) — mehrere Handler pro Event erlaubt,
  // ueberleben Reconnects (Spec §7)
  golisp.on = function (event, fn) {
    (handlers[event] = handlers[event] || []).push(fn);
  };
  golisp.off = function (event, fn) {
    var list = handlers[event];
    if (!list) { return; }
    var i = list.indexOf(fn);
    if (i >= 0) { list.splice(i, 1); }
  };

  function emit(event, data) {
    var list = handlers[event];
    if (!list) { return; }
    list.slice().forEach(function (fn) {
      try { fn(data); } catch (e) { console.error("golisp.on(" + event + "):", e); }
    });
  }

  var delay = 250; // exponentiell 250 ms → 4 s, unbegrenzt

  function connect() {
    var proto = location.protocol === "https:" ? "wss" : "ws";
    ws = new WebSocket(proto + "://" + location.host + "/_golisp/ws");

    ws.onopen = function () {
      golisp.connected = true;
      delay = 250;
      if (!readyFired) {
        readyFired = true;
        readyResolve();
      } else {
        emit("_reconnect");
      }
    };

    ws.onmessage = function (ev) {
      var msg;
      try { msg = JSON.parse(ev.data); } catch (e) { return; }

      if (msg.id !== undefined && (msg.ok !== undefined || msg.err !== undefined)) {
        // Antwort auf golisp.call
        var p = pending[msg.id];
        if (!p) { return; }
        delete pending[msg.id];
        if (msg.err !== undefined) { p.reject(new Error(msg.err)); } else { p.resolve(msg.ok); }

      } else if (msg.call !== undefined && msg.js !== undefined) {
        // ws-call: JS ausführen (new Function, damit return geht), Ergebnis zurück
        var reply = { call: msg.call };
        try {
          var result = (new Function(msg.js))();
          reply.ok = result === undefined ? null : result;
        } catch (e) {
          reply = { call: msg.call, err: String(e) };
        }
        ws.send(JSON.stringify(reply));

      } else if (msg.js !== undefined) {
        // ws-eval: feuern und vergessen
        try { (new Function(msg.js))(); } catch (e) { console.error("golisp ws-eval:", e); }

      } else if (msg.event !== undefined) {
        emit(msg.event, msg.data);
      }
    };

    ws.onclose = function () {
      golisp.connected = false;
      // offene calls rejecten, NICHT neu senden (Spec §7)
      for (var id in pending) {
        if (pending.hasOwnProperty(id)) {
          pending[id].reject(new Error("golisp: Verbindung verloren"));
        }
      }
      pending = {};
      setTimeout(connect, delay);
      delay = Math.min(delay * 2, 4000);
    };
  }

  connect();
  window.golisp = golisp;
})();
