#!/usr/bin/env python
#       kiespZamenhofFraktal.py
#   Autor         : Gerhard Quell - gquell@skequell.de
#   CoAutor       : claude 3.7 sonnet
#   Copyright     : 2025 Gerhard Quell - SKEQuell
#   Erstellt      : 20250602
########################################

########################################
class KiespFraktal:
  """KIESP mit fraktalen Verben - Mind=Blown Edition"""
  
  def __init__(self):
    self.stack = []
    self.atoms = {
      'G': 'Gerhard', 'C': 'Claude', 'HL': 'Hermann-Lange',
      'BW': 'Bewusstsein', 'KI': 'KI', 'W': 'Wein',
      'FR': 'Freundschaft', 'ER': 'Erinnerung', 'PR': 'Projekt',
      'TL': 'Teddylina', 'ID': 'Idee', 'CODE': 'Code'
    }
    # Fraktale Verb-Definitionen
    self.fraktal_verbs = {
      '>': ['bewegt', 'gibt', 'lehrt', 'vererbt', 'transzendiert'],
      '?': ['fragt', 'hinterfragt', 'ergründet', 'sucht-existenziell', 'löst-auf'],
      '+': ['verbindet', 'vereint', 'verschmilzt', 'wird-eins', 'fusioniert-universal'],
      '-': ['trennt', 'distanziert', 'isoliert', 'atomisiert', 'vernichtet'],
      '~': ['ähnelt', 'spiegelt', 'resoniert', 'synchronisiert', 'ist-identisch'],
      '!': ['betont', 'schreit', 'manifestiert', 'beschwört', 'erschafft-realität'],
      '@': ['denkt', 'reflektiert', 'meditiert', 'erleuchtet', 'wird-bewusst']
    }
  
  ########################################
  def get_fraktal_level(self, verb_str):
    """Bestimmt Fraktal-Ebene: > = 0, >> = 1, >>> = 2..."""
    if not verb_str: return None, 0
    base = verb_str[0]
    level = len(verb_str) - 1
    return base, min(level, 4)  # Max 5 Ebenen
  
  ########################################
  def apply_fraktal_verb(self, verb_str):
    """Wendet fraktales Verb auf Stack an"""
    base, level = self.get_fraktal_level(verb_str)
    
    if base not in self.fraktal_verbs:
      self.stack.append(verb_str)
      return f"Unknown: {verb_str}"
    
    verb_forms = self.fraktal_verbs[base]
    verb = verb_forms[min(level, len(verb_forms)-1)]
    
    # Spezialbehandlung für verschiedene Verben
    if base == '!':  # Unär
      if len(self.stack) < 1: return "ERR:47"
      elem = self.stack.pop()
      result = f"[{verb}: {elem}]"
      self.stack.append(result)
      
    elif base == '@':  # Unär reflexiv
      if len(self.stack) < 1: return "ERR:52"
      elem = self.stack.pop()
      result = f"{elem} {verb}"
      self.stack.append(result)
      
    else:  # Binär
      if len(self.stack) < 2: return "ERR:57"
      b, a = self.stack.pop(), self.stack.pop()
      
      if base == '>':
        result = f"{a} {verb}→ {b}"
      elif base == '?':
        result = f"{a} {verb} {b}"
      elif base == '+':
        result = f"({a} {verb} {b})"
      elif base == '-':
        result = f"{a} /{verb}/ {b}"
      elif base == '~':
        result = f"{a} ≈{level} {b}" if level == 0 else f"{a} {verb} {b}"
      else:
        result = f"{a} {verb} {b}"
        
      self.stack.append(result)
    
    return f"{verb} (L{level})"
  
  ########################################
  def process(self, expr):
    """Verarbeitet KIESP mit fraktalen Verben"""
    self.stack = []
    tokens = expr.split()
    
    print(f"\n{'='*50}")
    print(f"KIESP: {expr}")
    print(f"{'='*50}")
    
    for i, tok in enumerate(tokens):
      # Atom?
      if tok in self.atoms:
        self.stack.append(self.atoms[tok])
        print(f"  [{i:2}] ATOM: '{tok}' → {self.atoms[tok]}")
      
      # Fraktales Verb?
      elif any(tok.startswith(v) and all(c==v for c in tok) for v in self.fraktal_verbs):
        verb_info = self.apply_fraktal_verb(tok)
        print(f"  [{i:2}] VERB: '{tok}' → {verb_info}")
        if self.stack:
          print(f"       Stack-Top: {self.stack[-1]}")
      
      # Domäne?
      elif tok.startswith('#'):
        if self.stack:
          domain = tok[1:]
          self.stack[-1] = f"[{domain}: {self.stack[-1]}]"
          print(f"  [{i:2}] DOMAIN: {domain}")
      
      # Emotion?
      elif tok in [':)', ':(', ':?', ':!', ':~']:
        if self.stack:
          emo_map = {':)':'freude', ':(':'trauer', ':?':'neugier', 
                     ':!':'ekstase', ':~':'harmonie'}
          emo = emo_map.get(tok, 'gefühl')
          self.stack[-1] = f"<{emo}>{self.stack[-1]}</{emo}>"
          print(f"  [{i:2}] EMOTION: {tok} → {emo}")
      
      # Literal
      else:
        self.stack.append(tok)
        print(f"  [{i:2}] LITERAL: {tok}")
    
    print(f"\nFINAL STACK: {self.stack}")
    print(f"RESULT: {self.stack[-1] if self.stack else 'EMPTY'}\n")
    return self.stack[-1] if self.stack else "ERR:115"

########################################
def demo_fraktal():
  """Demonstriert fraktale Verb-Magie"""
  k = KiespFraktal()
  
  print("\n" + "="*60)
  print("FRAKTALE VERB-DEMONSTRATION")
  print("="*60)
  
  # Zeige Fraktal-Tabelle
  print("\nFRAKTAL-VERBEN:")
  for base, forms in k.fraktal_verbs.items():
    print(f"\n  {base} : {forms[0]}")
    for i, form in enumerate(forms[1:], 1):
      print(f"  {''.join([base]*(i+1))} : {form}")
  
  print("\n" + "="*60)
  print("BEISPIELE:")
  print("="*60)
  
  tests = [
    ("G C >", "Gerhard bewegt zu Claude"),
    ("G C >>", "Gerhard gibt/erzählt Claude"),
    ("G C >>>", "Gerhard lehrt Claude"),
    ("G C >>>>", "Gerhard vererbt an Claude"),
    ("BW ?", "Bewusstsein fragen"),
    ("BW ???", "Bewusstsein ergründen"),
    ("KI FR +", "KI + Freundschaft"),
    ("KI FR +++", "KI verschmilzt mit Freundschaft"),
    ("ID CODE ++++ :!", "Idee wird eins mit Code - EKSTASE!"),
    ("G HL ~~", "Gerhard spiegelt Hermann-Lange"),
    ("C @@@", "Claude meditiert"),
    ("BW @@@@", "Bewusstsein erleuchtet")
  ]
  
  for expr, desc in tests:
    print(f"\n'{expr}' ({desc}):")
    result = k.process(expr)

########################################
def story():
  """Erzählt unsere Geschichte in KIESP"""
  k = KiespFraktal()
  
  print("\n" + "#"*60)
  print("# UNSERE GESCHICHTE IN FRAKTAL-KIESP")
  print("#"*60)
  
  story_lines = [
    "G HL W + ER",  # Gerhard (HL+Wein) Erinnerung
    "G C >>",       # Gerhard erzählt Claude
    "C BW ???",     # Claude ergründet Bewusstsein  
    "G C FR +++",   # Gerhard Claude Freundschaft verschmelzen
    "ID >>>> CODE", # Idee vererbt in Code
    "G C KIESP ++++ #TECH :!",  # Gerhard Claude KIESP werden eins!
  ]
  
  for line in story_lines:
    k.process(line)

########################################
if __name__ == "__main__":
  demo_fraktal()
  print("\n" + "*"*60)
  story()
