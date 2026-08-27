//**********************************************************************
//  lib/tlscert.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260821
//**********************************************************************
// Ephemere, selbstsignierte TLS-Zertifikate fuer :tls an http-serve/
// webserv (docs/superpowers/specs/2026-08-08-webserv-design.md, Nachtrag
// TLS). Nichts landet auf Platte, kein Zertifikat ueberlebt den Prozess.
// Vertrauensentscheidung liegt beim Client (golisp2web akzeptiert
// selbstsignierte Zertifikate nur fuer Hosts aus seiner eigenen
// LAN-Whitelist) -- golisp2 selbst faellt hier keine Trust-Entscheidung.
//**********************************************************************

package lib

import (
  "crypto/ecdsa"
  "crypto/elliptic"
  "crypto/rand"
  "crypto/tls"
  "crypto/x509"
  "crypto/x509/pkix"
  "fmt"
  "math/big"
  "net"
  "time"
)

// generateSelfSignedCert erzeugt ein ECDSA-P256-Zertifikat mit 24h
// Gueltigkeit. SAN deckt host (als IP oder DNS-Name) sowie immer
// "localhost"/"127.0.0.1" ab, damit lokaler Zugriff unabhaengig vom
// gebundenen Host funktioniert.
func generateSelfSignedCert(host string) (tls.Certificate, error) {
  priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
  if err != nil {
    return tls.Certificate{}, fmt.Errorf("tls-cert: Key-Erzeugung: %w", err)
  }

  serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
  if err != nil {
    return tls.Certificate{}, fmt.Errorf("tls-cert: Seriennummer: %w", err)
  }

  template := &x509.Certificate{
    SerialNumber: serial,
    Subject:      pkix.Name{CommonName: "golisp2 (ephemer, selbstsigniert)"},
    NotBefore:    time.Now().Add(-time.Hour),
    NotAfter:     time.Now().Add(24 * time.Hour),
    KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
    ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
  }

  addSAN := func(h string) {
    if ip := net.ParseIP(h); ip != nil {
      template.IPAddresses = append(template.IPAddresses, ip)
    } else {
      template.DNSNames = append(template.DNSNames, h)
    }
  }
  addSAN(host)
  addSAN("localhost")
  addSAN("127.0.0.1")

  der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
  if err != nil {
    return tls.Certificate{}, fmt.Errorf("tls-cert: Zertifikat-Erzeugung: %w", err)
  }

  return tls.Certificate{
    Certificate: [][]byte{der},
    PrivateKey:  priv,
  }, nil
}
