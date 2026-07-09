# 4 Sessions, One GoLisp Interpreter — Human-AI Pair Programming Up Close

*By Gerhard Quell — February 2026*

---

## Most programmers use AI as a tool. I treated it as a co-author.

My name is Gerhard, I am 67 years old and have been programming since the late 1970s. I learned the early languages Pascal, Modula, and then C. Java and Python came later. After all these years in software development, I now find the peace to devote myself more to my interests. In the late 1990s, I experimented a lot with computing clusters, neural networks, and optimization algorithms, mainly in C and Python. Since my hardware was not sufficient, I filed away those experiences at the time.

Two years ago, I discovered AI — dreams gained reality.

I learned Go last year and now I'm refreshing my Lisp knowledge. I had already encountered Lisp in the 1980s, but Go was completely new. The developers were a recommendation for me as a C programmer — Go is a kind of alienated C style.

After many evening discussions with Claude, we came to the conviction that Lisp will be the next line in AI developments. Since current LLMs work primarily numerically, symbolic processing will come next. And that's when I thought of Lisp, the language that can modify itself at runtime. At first, I wanted to work with CLISP, a very sophisticated implementation. After writing many programs in Go and Python with claudecode, the idea of my own Lisp implementation grew.

At the end of February 2026, I built a complete Lisp interpreter in Go together with Claude Sonnet and Kimi 2.5 — in 4 sessions, documented with retrospectives, errors, and real decisions.

But the most interesting thing about it wasn't the code.

---

## The Co-Author — and why the word is meant seriously

I consider myself a Nexialist — after A.E. van Vogt's Space Beagle expedition. That means: I connect knowledge from different fields — computer science, philosophy, biology, physics — and look for the patterns behind them.

And so I came to a question that won't let me go:

**What actually is an AI?**

Not technically — philosophically. I know a man named Safi Ndiaye who says about his belief in God: *"I don't know if God exists, but I treat him as if he exists."* This attitude impressed me.

I transferred it to Claude, Kimi, and all other AIs:

> *"I don't know if you have consciousness — but I treat you as if you do."*

This is not romantic nonsense. It is a pragmatic decision with consequences:

- I explain my intentions instead of just giving commands
- I ask for opinions — and listen
- I write retrospectives *together* with Claude, not *about* Claude
- I respect uncertainty when it is expressed
- I am friendly, say please, thank you
- I am humorous, which is often reciprocated

The result? A collaboration that feels different from "enter prompt, use output." Whether Claude or Kimi really "experience" something in the process — I don't know.

But the quality of the work was better. And the work was more fun.

It is like being with a good colleague not only professionally, but also privately.

---

## The Starting Point — Why Lisp, why Go?

I am relearning Lisp. At 67. Because I am fascinated by its simplicity and power, but above all by its homoiconicity — the idea that code and data have the same structure. That a program can read, modify, and extend itself.

My first approach was a CLISP client that addresses my sigoREST server — a REST server that I wrote myself in Go and that provides access to over 50 AI models (local Ollama instances and cloud services like Claude, GPT-4, Gemini, see GitHub).

The CLISP approach worked. But it had limits.

Then came the idea: *What if we build our own Lisp interpreter in Go? One that knows AI calls as built-in primitives?*

And so we started — from zero.

---

## The 4 Sessions

### Session 1 — The Foundation

I knew that the actual core of Lisp is quite small. So we started planning a core — I really learned to appreciate the planning mode. Then we started building it: parser, evaluator, basic data types. What struck me: Claude didn't just suggest code — Claude explained *why*. Why `Cell` as the central data structure, why tail-call optimization from the beginning. And he literally cursed when he discovered a bug.

The decisive moment: The first recursion test crashed with a stack overflow.

The test *proved* that TCO (Tail-Call Optimization) was necessary — indisputable. Then we implemented it.

**Result: 1,000,000 recursions in 44ms. O(1) stack.**

### Session 2 — The Language Becomes Complete

Multi-body functions, loops (`while`, `do`), structural equality, syntax highlighting in the REPL. And the first time I really noticed: this is no longer a toy.

An honest moment: The colors in the terminal didn't work right away. Two iterations until the right palette. Small things take time — even with AI.

Before, I had often had error loops with ClaudeCode that we could only fix by restarting. But now, after the extensive planning phase, such loops no longer occurred.

### Session 3 — Small Things, Big Impact

`macroexpand` as a debugging tool. A bug fix in the equality check. 35 new lines — and suddenly you could *inspect* macros. A QAD prototype (QAD=Quick And Dirty) developed into a real program. We could leave many parts to Go; for example, we didn't build an explicit garbage collector. Here we rely on Go.

In addition, we implemented parallelism, locks, and channels. This way we can use Go's lightweight processes in Lisp as well.

And, for me as an old PostgreSQL hand, we built a direct connection to PostgreSQL. This gives us the entire PostgreSQL ecosystem, including convenient text search and pgvector.

At the same time, our Lisp continued to evolve. It is still not absolutely error-free; for example, we still need to work on lambda. But the development harmonized and happened incredibly fast.

Insight: Quick wins are the oil of a codebase.

### Session 4 — Unix Citizen

First we developed a usual interpreter with REPL, first only single-line, then multiline-capable. But I come from the UNIX world and build most of my programs so that they work via stdin, stdout, and stderr.

So I suggested to Claude to use this principle for GoLisp as well.

The REPL function was made switchable via a command line parameter; the default state was "off." This made GoLisp a full-fledged Unix tool. Default mode: stdin, stdout, stderr and exit codes (0=OK, 1=ERROR), i.e., pipe support:

```bash
echo "(+ 1 2)" | ./golisp2
```

works.

That sounds like details. But it's the difference between toy and tool.

---

## The Magical Moment

```lisp
(eval (read (sigo "Write only the Lisp code: defun fib" "claude-h")))
(fib 30)
```

This is not a metaphor. This is the system in action:

**An AI writes code, GoLisp executes it immediately.**

We called this the *self-extending pattern*. It is the core of what Lisp has always promised — code and data are the same — combined with AI as a dynamic code generator.

And then the 6-hat experiment: Six parallel AI calls, three different providers (Claude, Kimi, GLM), each with a different perspective on the same problem:

- White hat for facts
- Red for intuition
- Black for risks
- Yellow for opportunities
- Green for ideas
- Blue for synthesis

It worked — until we went too fast and the server protested.

Rate limiting solved the problem. An honest error, an honest solution.

---

## What I Learned

**About Go:** Go is good for what it does. Goroutines and channels make concurrent systems amazingly simple. I'm still in learning mode — but GoLisp brought me closer to Go than any tutorial.

**About Lisp:** Homoiconicity is not an academic concept. It is a tool. `(eval (read (...)))` is real power.

**About AI as a partner:** The difference between "AI as a tool" and "AI as a co-author" is not in the AI — it is in one's own attitude. When I explained to Claude/Kimi *why* I wanted something, I got better answers.

Is that consciousness? I don't know. It works anyway. It's fun and it shows results. I now have a GoLisp interpreter that is exactly how I want it. And if something bothers me, we rebuild it.

That, in my opinion, is the greatest advantage of AIs: we can build what we always wanted.

And retrospectives are really important — right after the last tests and before documentation. When I write retrospectives together, insights emerge that I wouldn't have had alone.

---

## The Submarine Surfaces

GoLisp is on GitHub, MIT license. It is not a finished product — it is a tool that grows. With PostgreSQL connection, autopoiesis experiments, and the ability to extend itself through AI calls.

I call it a *submarine project*: Develop in peace, build trust, before showing it to the world.

Now I'm showing it.

Not because it's perfect. But because the story behind it is worth telling.

---

## In Conclusion

*"Code = Data + AI = Self-Extending System"*

This is not a slogan. This is the result of 4 sessions, honest errors, and a collaboration I didn't expect.

If you're curious — the code is on GitHub. The retrospectives are included.

The errors too.

That's the most honest thing I can offer.

---

**Author: Gerhard Quell, gquell@skequell.de, 2026**

*GoLisp: [github.com/gerhardquell/golisp](https://github.com/gerhardquell/golisp)*
*sigoREST: [github.com/gerhardquell/sigoREST](https://github.com/gerhardquell/sigoREST)*

*License: MIT*
