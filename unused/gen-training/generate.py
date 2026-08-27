# **********************************************************************
#  tools/gen-training/generate.py
#  Autor    : Gerhard Quell - gquell@skequell.de
#  CoAutor  : claude sonnet 4.6
#  Copyright: 2026 Gerhard Quell - SKEQuell
#  Erstellt : 20260623
# **********************************************************************
# Erzeugt aus data.py die Dateien golisp-tutorial.md und golisp-anki.json.
# **********************************************************************

import json
import random
from datetime import date

from data import FUNCS

CATEGORY_ORDER = [
    "Spezialformen",
    "Arithmetik",
    "Vergleiche",
    "Typ-Prädikate",
    "Listen",
    "Strings",
    "Ein-/Ausgabe",
    "Fehler",
    "Makro-Hilfe",
    "Datei-I/O",
    "Nebenläufigkeit",
    "KI",
    "Zeit",
    "Memory",
    "Shell & System",
    "PostgreSQL",
    "Standardbibliothek",
]


def write_tutorial():
    by_cat = {}
    for f in FUNCS:
        by_cat.setdefault(f["cat"], []).append(f)

    lines = []
    lines.append("<!--")
    lines.append("  golisp-tutorial.md")
    lines.append("  Autor    : Gerhard Quell - gquell@skequell.de")
    lines.append("  CoAutor  : claude sonnet 4.6")
    lines.append("  Copyright: 2026 Gerhard Quell - SKEQuell")
    lines.append(f"  Erstellt : {date.today().strftime('%Y%m%d')}")
    lines.append("-->")
    lines.append("")
    lines.append("# GoLisp – Tutorial")
    lines.append("")
    lines.append(
        "Dieses Dokument beschreibt die öffentlichen Funktionen, Spezialformen und "
        "Makros von GoLisp mit kurzen, lauffähigen Beispielen. Dateioperationen "
        "nutzen immer das Projekt-temp-Verzeichnis `./tmp`. Funktionen, die externe "
        "Dienste benötigen (sigo, PostgreSQL), enthalten beschreibende Beispiele."
    )
    lines.append("")

    for cat in CATEGORY_ORDER:
        entries = by_cat.get(cat, [])
        if not entries:
            continue
        lines.append(f"## {cat}")
        lines.append("")
        for entry in entries:
            lines.append(f"### {entry['name']}")
            lines.append("")
            lines.append(f"**Syntax:** `{entry['syntax']}`")
            lines.append("")
            lines.append(entry["desc"])
            lines.append("")
            if entry.get("examples"):
                lines.append("```lisp")
                for ex in entry["examples"]:
                    lines.append(ex)
                lines.append("```")
                lines.append("")

    with open("golisp-tutorial.md", "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
        f.write("\n")


def mc_options(correct, all_cats):
    others = [c for c in all_cats if c != correct]
    random.shuffle(others)
    opts = [correct] + others[:4]
    random.shuffle(opts)
    return [{"mctext": o, "mctf": o == correct} for o in opts]


def write_anki():
    categories = sorted({f["cat"] for f in FUNCS})
    cards = []
    for entry in FUNCS:
        syntax = entry["syntax"] or f"({entry['name']} ...)"
        cards.append({
            "name": entry["name"],
            "cont": entry["desc"],
            "anki": [
                {"frage": f"Was macht `{entry['name']}` in GoLisp?", "antwort": entry["desc"]},
                {"frage": f"Wie lautet die Syntax von `{entry['name']}`?", "antwort": syntax},
            ],
            "mult": [
                {
                    "frage": f"Zu welcher Kategorie gehört `{entry['name']}`?",
                    "antworten": mc_options(entry["cat"], categories),
                }
            ],
        })

    with open("golisp-anki.json", "w", encoding="utf-8") as f:
        json.dump(cards, f, ensure_ascii=False, indent=2)
        f.write("\n")


if __name__ == "__main__":
    random.seed(42)  # reproduzierbare MC-Optionen
    write_tutorial()
    write_anki()
    print("golisp-tutorial.md und golisp-anki.json erzeugt.")
