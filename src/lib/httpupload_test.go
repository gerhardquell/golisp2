//**********************************************************************
//  lib/httpupload_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright : 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260821
//**********************************************************************
// Tests fuer http-upload: Handler-Aufruf pro Datei, Mehrfach-Upload,
// Fehlerfaelle (falsche Methode, kaputtes Formular, Handler-Fehler).
//**********************************************************************

package lib

import (
  "bytes"
  "mime/multipart"
  "net/http"
  "strconv"
  "testing"
)

// multipartBody baut einen multipart/form-data-Body mit einer Datei.
func multipartBody(t *testing.T, fieldname, filename, content string) (*bytes.Buffer, string) {
  t.Helper()
  buf := &bytes.Buffer{}
  w := multipart.NewWriter(buf)
  fw, err := w.CreateFormFile(fieldname, filename)
  if err != nil {
    t.Fatal(err)
  }
  if _, err := fw.Write([]byte(content)); err != nil {
    t.Fatal(err)
  }
  if err := w.Close(); err != nil {
    t.Fatal(err)
  }
  return buf, w.FormDataContentType()
}

func TestHTTPUploadSingleFile(t *testing.T) {
  ws, env := wsTestServer(t)
  srv := &Cell{Type: LIST, Env: ws}

  evalCode(t, env, `(define *uploads* '())`)
  handler := evalCode(t, env, `(lambda (filename content)
    (setq *uploads* (cons (cons filename content) *uploads*)))`)

  if _, err := fnHTTPUpload([]*Cell{srv, MakeStr("/upload"), handler}); err != nil {
    t.Fatalf("http-upload: %v", err)
  }

  body, contentType := multipartBody(t, "file", "hallo.txt", "inhalt-123")
  url := "http://127.0.0.1:" + strconv.Itoa(ws.port) + "/upload"
  resp, err := http.Post(url, contentType, body)
  if err != nil {
    t.Fatalf("POST: %v", err)
  }
  defer resp.Body.Close()
  if resp.StatusCode != http.StatusOK {
    t.Fatalf("status = %d, erwartet 200", resp.StatusCode)
  }

  uploads, err := env.Get("*uploads*")
  if err != nil {
    t.Fatal(err)
  }
  if uploads.Type != LIST {
    t.Fatalf("*uploads* ist keine Liste: %v", uploads)
  }
  entry := uploads.Car
  if entry.Car.Val != "hallo.txt" || entry.Cdr.Val != "inhalt-123" {
    t.Fatalf("Handler bekam falsche Werte: filename=%q content=%q",
      entry.Car.Val, entry.Cdr.Val)
  }
}

func TestHTTPUploadMultipleFiles(t *testing.T) {
  ws, env := wsTestServer(t)
  srv := &Cell{Type: LIST, Env: ws}

  evalCode(t, env, `(define *count* 0)`)
  handler := evalCode(t, env, `(lambda (filename content) (setq *count* (+ *count* 1)))`)
  if _, err := fnHTTPUpload([]*Cell{srv, MakeStr("/upload"), handler}); err != nil {
    t.Fatalf("http-upload: %v", err)
  }

  buf := &bytes.Buffer{}
  w := multipart.NewWriter(buf)
  for i, name := range []string{"a.txt", "b.txt"} {
    fw, err := w.CreateFormFile("file", name)
    if err != nil {
      t.Fatal(err)
    }
    if _, err := fw.Write([]byte("x")); err != nil {
      t.Fatal(err)
    }
    _ = i
  }
  if err := w.Close(); err != nil {
    t.Fatal(err)
  }

  url := "http://127.0.0.1:" + strconv.Itoa(ws.port) + "/upload"
  resp, err := http.Post(url, w.FormDataContentType(), buf)
  if err != nil {
    t.Fatalf("POST: %v", err)
  }
  defer resp.Body.Close()
  if resp.StatusCode != http.StatusOK {
    t.Fatalf("status = %d, erwartet 200", resp.StatusCode)
  }

  count, err := env.Get("*count*")
  if err != nil {
    t.Fatal(err)
  }
  if count.Num != 2 {
    t.Fatalf("*count* = %v, erwartet 2 (zwei Dateien)", count.Num)
  }
}

func TestHTTPUploadWrongMethod(t *testing.T) {
  ws, env := wsTestServer(t)
  srv := &Cell{Type: LIST, Env: ws}
  handler := evalCode(t, env, `(lambda (filename content) content)`)
  if _, err := fnHTTPUpload([]*Cell{srv, MakeStr("/upload"), handler}); err != nil {
    t.Fatalf("http-upload: %v", err)
  }

  url := "http://127.0.0.1:" + strconv.Itoa(ws.port) + "/upload"
  resp, err := http.Get(url)
  if err != nil {
    t.Fatalf("GET: %v", err)
  }
  defer resp.Body.Close()
  if resp.StatusCode != http.StatusMethodNotAllowed {
    t.Fatalf("status = %d, erwartet 405", resp.StatusCode)
  }
}

func TestHTTPUploadHandlerError(t *testing.T) {
  ws, env := wsTestServer(t)
  srv := &Cell{Type: LIST, Env: ws}
  handler := evalCode(t, env, `(lambda (filename content) (error "kaputt"))`)
  if _, err := fnHTTPUpload([]*Cell{srv, MakeStr("/upload"), handler}); err != nil {
    t.Fatalf("http-upload: %v", err)
  }

  body, contentType := multipartBody(t, "file", "x.txt", "y")
  url := "http://127.0.0.1:" + strconv.Itoa(ws.port) + "/upload"
  resp, err := http.Post(url, contentType, body)
  if err != nil {
    t.Fatalf("POST: %v", err)
  }
  defer resp.Body.Close()
  if resp.StatusCode != http.StatusInternalServerError {
    t.Fatalf("status = %d, erwartet 500", resp.StatusCode)
  }
}

func TestHTTPUploadErrors(t *testing.T) {
  ws, env := wsTestServer(t)
  srv := &Cell{Type: LIST, Env: ws}
  handler := evalCode(t, env, `(lambda (filename content) content)`)

  cases := []struct {
    name string
    args []*Cell
  }{
    {"zu wenig Argumente", []*Cell{srv, MakeStr("/upload")}},
    {"urlpath falscher Typ", []*Cell{srv, MakeNum(1), handler}},
    {"urlpath ohne fuehrenden Slash", []*Cell{srv, MakeStr("upload"), handler}},
    {"handler kein Lambda", []*Cell{srv, MakeStr("/upload"), MakeStr("x")}},
  }
  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      if _, err := fnHTTPUpload(tc.args); err == nil {
        t.Fatalf("%s: erwartet Fehler, bekam keinen", tc.name)
      }
    })
  }
}
