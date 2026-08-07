//**********************************************************************
//  lib/wsbridge.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// WebSocket-Bridge der Web-Bridge (Spec TODO.md §3.2, §5).
// Schritt 3 der Umsetzung — aktuell nur der Client-Typ, damit
// httpserver.go eigenstaendig kompiliert.
//**********************************************************************

package lib

import "github.com/gorilla/websocket"

// wsClient ist eine verbundene WebSocket-Verbindung (Spec §5: eigene
// Writer-Goroutine mit gepuffertem Channel, Kapazitaet 64).
type wsClient struct {
  id   int
  conn *websocket.Conn
  send chan []byte
}
