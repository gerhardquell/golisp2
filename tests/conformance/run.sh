#!/usr/bin/env bash
# run.sh — Konformitäts-Suite: golisp2 gegen clisp-Gold
# Autor: Gerhard Quell – gquell@skequell.de · CoAutor: kimi-k3
# Erstellt: 20260723
#
# Aufruf:
#   ./run.sh          Suite laufen lassen (Exit 0 = alles PASS/XFAIL)
#   ./run.sh --gold   Gold-Dateien via clisp neu erzeugen (bewusste Aktion!)
#
# Status je Fall:
#   PASS   golisp2 == Gold
#   FAIL   golisp2 != Gold
#   XFAIL  != Gold, aber in known-failures.txt gelistet (erwartetes Rot)
#   XPASS  == Gold, aber in known-failures.txt gelistet (Suite darf reifen!)
#
# Design: ein golisp2-Prozess pro Case-Datei (Zustand bleibt, wie im
# clisp-Treiber). Fehlerzeilen "ERR: …" im 2>&1-Strom markieren ERROR-Fälle
# positionsgenau.

set -u
cd "$(dirname "$0")/../.."
DIR=tests/conformance
GOLISP=./build/golisp2
CLISP="clisp -q -norc"

if [[ "${1:-}" == "--gold" ]]; then
  for f in "$DIR"/cases/*.lisp; do
    name=$(basename "$f" .lisp)
    $CLISP "$DIR/driver-clisp.lisp" "$f" 2>/dev/null | tr -d '\r' > "$DIR/gold/$name.gold"
    echo "gold: $name.gold ($(wc -l < "$DIR/gold/$name.gold") Fälle)"
  done
  exit 0
fi

[[ -x $GOLISP ]] || { echo "FEHLER: $GOLISP fehlt — erst ./build.sh"; exit 2; }

# Normalisierung beider Seiten:
#  ()    -> nil     Drucker-Konvention, eine Ursache (TODO eigener Fall)
#  #<…>  -> #<>     unreadable Objects (Funktionen), Identität irrelevant
norm() { sed 's/()/nil/g; s/#<[^>]*>/#<>/g; s/#s(hash-table[^)]*)/#<>/g' | tr 'A-Z' 'a-z' | sed 's/^error$/ERROR/'; }

pass=0; fail=0; xfail=0; xpass=0
fail_details=()

for f in "$DIR"/cases/*.lisp; do
  name=$(basename "$f" .lisp)
  gold="$DIR/gold/$name.gold"
  [[ -f $gold ]] || { echo "FEHLER: $gold fehlt — erst ./run.sh --gold"; exit 2; }

  # Formen (Kommentare/Leerzeilen weg) einmal durch einen Prozess jagen
  mapfile -t forms < <(sed 's/\r$//' "$f" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | grep -v '^;;' | grep -v '^$')
  mapfile -t out < <(printf '%s\n' "${forms[@]}" | $GOLISP 2>&1)
  mapfile -t goldlines < "$gold"

  if [[ ${#out[@]} -ne ${#forms[@]} ]]; then
    echo "FEHLER: $name — Zeilen-Alignment verloren (${#forms[@]} Formen, ${#out[@]} Ausgaben)"
    exit 2
  fi

  for i in "${!forms[@]}"; do
    form="${forms[$i]}"
    want="$(echo "${goldlines[$i]:-}" | norm)"
    raw="${out[$i]}"
    if [[ $raw == ERR:* ]]; then
      got="ERROR"
    else
      got="$(echo "$raw" | norm)"
    fi

    if [[ $got == "$want" ]]; then
      if grep -qFx "$form" "$DIR/known-failures.txt" 2>/dev/null; then
        xpass=$((xpass+1)); fail_details+=("XPASS  $name: $form")
      else
        pass=$((pass+1))
      fi
    else
      if grep -qFx "$form" "$DIR/known-failures.txt" 2>/dev/null; then
        xfail=$((xfail+1))
      else
        fail=$((fail+1)); fail_details+=("FAIL   $name: $form"$'\n'"       gold=[$want] got=[$got]")
      fi
    fi
  done
done

for d in ${fail_details[@]+"${fail_details[@]}"}; do echo "$d"; done
echo "----------------------------------------"
echo "PASS=$pass  FAIL=$fail  XFAIL=$xfail  XPASS=$xpass"
(( fail == 0 && xpass == 0 )) && exit 0 || exit 1
