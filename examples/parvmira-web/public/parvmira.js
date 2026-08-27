// parvmira.js
// Autor    : Gerhard Quell - gquell@skequell.de
// CoAutor  : Claude Sonnet 5
// Copyright: 2026 Gerhard Quell - SKEQuell
// Erstellt : 20260826
// PARVMIRA-Entscheidungsmodell (NEXORA) - Browser-Oberflaeche.
// Anders als beim ABC-Generator lebt der State HIER NICHT im Browser:
// jede Aenderung geht sofort per ws-Call an den Server und wird danach
// aus dem Server-Zustand neu geladen/gerendert.

const KATEGORIEN = ["plus", "minus", "interessant"];

function zeigeStatus(text) {
  document.getElementById("statuszeile").textContent = text;
}

function punktZeile(punkt) {
  const li = document.createElement("li");
  li.textContent = punkt.kurzbezeichnung + ": " + punkt.beschreibung +
    " [" + punkt.erstellerTyp + "/" + punkt.erstellerName +
    (punkt.persoenlichkeit ? "/" + punkt.persoenlichkeit : "") + "]";
  return li;
}

async function ladePunkte(kategorie) {
  const liste = document.getElementById("liste-" + kategorie);
  try {
    const punkte = await window.golisp.call("punkte-abrufen", kategorie);
    liste.innerHTML = "";
    punkte.forEach((punkt) => liste.appendChild(punktZeile(punkt)));
  } catch (e) {
    zeigeStatus("Fehler beim Laden (" + kategorie + "): " + e.message);
  }
}

async function ladeAllePunkte() {
  for (const kategorie of KATEGORIEN) {
    await ladePunkte(kategorie);
  }
}

async function ladeVorgabe() {
  try {
    const text = await window.golisp.call("vorgabe-abrufen");
    document.getElementById("vorgabe").value = text;
  } catch (e) {
    zeigeStatus("Fehler beim Laden der Vorgabe: " + e.message);
  }
}

async function vorgabeSetzen() {
  const text = document.getElementById("vorgabe").value.trim();
  if (text.length < 1) {
    alert("Bitte eine Vorgabe eingeben.");
    return;
  }
  try {
    await window.golisp.call("vorgabe-setzen", text);
    zeigeStatus("Vorgabe gesetzt - neue Entscheidung begonnen.");
    await ladeAllePunkte();
  } catch (e) {
    zeigeStatus("Fehler beim Setzen der Vorgabe: " + e.message);
  }
}

async function ladeModelle() {
  const ziel = document.getElementById("modelleListe");
  try {
    const modelle = await window.golisp.call("modelle-abrufen");
    ziel.innerHTML = "";
    if (modelle.length === 0) {
      ziel.textContent = "(keine Modelle - sigoREST nicht erreichbar?)";
      return;
    }
    modelle.forEach((modell) => {
      const label = document.createElement("label");
      const checkbox = document.createElement("input");
      checkbox.type = "checkbox";
      checkbox.value = modell;
      checkbox.checked = true;
      label.appendChild(checkbox);
      label.appendChild(document.createTextNode(" " + modell));
      ziel.appendChild(label);
    });
  } catch (e) {
    ziel.textContent = "Fehler beim Laden der Modelle: " + e.message;
  }
}

function gewaehlteWerte(containerId) {
  const container = document.getElementById(containerId);
  return Array.from(container.querySelectorAll("input[type=checkbox]:checked"))
    .map((cb) => cb.value);
}

function baueEnsemble() {
  const modelle = gewaehlteWerte("modelleListe");
  const persoenlichkeiten = gewaehlteWerte("persoenlichkeitenListe");
  const ensemble = [];
  modelle.forEach((modell) => {
    persoenlichkeiten.forEach((persoenlichkeit) => {
      ensemble.push([modell, persoenlichkeit]);
    });
  });
  return ensemble;
}

async function punktHinzufuegen(kategorie, block) {
  const feldKurz = block.querySelector(".feld-kurz");
  const feldBeschr = block.querySelector(".feld-beschr");
  const feldName = block.querySelector(".feld-name");
  const kurzbezeichnung = feldKurz.value.trim();
  if (kurzbezeichnung.length < 1) {
    alert("Bitte eine Kurzbezeichnung eingeben.");
    return;
  }
  try {
    await window.golisp.call("punkt-hinzufuegen", kategorie,
      kurzbezeichnung, feldBeschr.value.trim(), feldName.value.trim());
    feldKurz.value = "";
    feldBeschr.value = "";
    feldName.value = "";
    await ladePunkte(kategorie);
    zeigeStatus("Punkt hinzugefügt (" + kategorie + ").");
  } catch (e) {
    zeigeStatus("Fehler beim Hinzufügen (" + kategorie + "): " + e.message);
  }
}

async function kiAbrufen(kategorie) {
  const ensemble = baueEnsemble();
  if (ensemble.length === 0) {
    alert("Bitte mindestens ein Modell und eine Persönlichkeit auswählen.");
    return;
  }
  zeigeStatus("KI-Ensemble wird abgefragt (" + kategorie + ")...");
  try {
    const neue = await window.golisp.call("ki-abrufen", kategorie, ensemble);
    await ladePunkte(kategorie);
    zeigeStatus("KI-Ensemble geliefert (" + kategorie + "): " + neue.length + " Punkt(e).");
  } catch (e) {
    zeigeStatus("Fehler beim KI-Ensemble-Abruf (" + kategorie + "): " + e.message);
  }
}

function zweistellig(n) {
  return n < 10 ? "0" + n : "" + n;
}

function zeitstempelDateiname(jetzt) {
  return "" + jetzt.getFullYear() + zweistellig(jetzt.getMonth() + 1) +
    zweistellig(jetzt.getDate()) + zweistellig(jetzt.getHours()) +
    zweistellig(jetzt.getMinutes());
}

function zeitstempelAnzeige(jetzt) {
  return zweistellig(jetzt.getDate()) + "." + zweistellig(jetzt.getMonth() + 1) +
    "." + jetzt.getFullYear() + " " + zweistellig(jetzt.getHours()) + ":" +
    zweistellig(jetzt.getMinutes());
}

async function speichern() {
  const jetzt = new Date();
  try {
    const dateiname = await window.golisp.call("speichern",
      zeitstempelDateiname(jetzt), zeitstempelAnzeige(jetzt));
    zeigeStatus("Gespeichert: " + dateiname);
  } catch (e) {
    zeigeStatus("Fehler beim Speichern: " + e.message);
  }
}

function initialisiereKategorieBloecke() {
  KATEGORIEN.forEach((kategorie) => {
    const block = document.getElementById("block-" + kategorie);
    block.querySelector(".btnHinzufuegen")
      .addEventListener("click", () => punktHinzufuegen(kategorie, block));
    block.querySelector(".btnKiAbrufen")
      .addEventListener("click", () => kiAbrufen(kategorie));
  });
}

async function initialisieren() {
  initialisiereKategorieBloecke();
  document.getElementById("btnVorgabeSetzen").addEventListener("click", vorgabeSetzen);
  document.getElementById("btnSpeichern").addEventListener("click", speichern);

  await window.golisp.ready;
  await ladeVorgabe();
  await ladeAllePunkte();
  await ladeModelle();
}

document.addEventListener("DOMContentLoaded", initialisieren);
