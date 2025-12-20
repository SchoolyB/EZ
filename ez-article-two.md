# Building EZ: Part 2

**Subtitle:** So What's Next?

---

## Introduction

You shipped something. Congratulations. The GitHub stars trickle in, maybe someone files an issue, and reality sets in: you now have a codebase to maintain.

Most "vibe coding" content focuses on the build. The initial rush of watching an AI spin up features faster than you can review them. Nobody talks about what happens after the honeymoon - when you're staring at code you half-understand, hunting bugs across a system that grew faster than your comprehension of it.

This is that article.

After shipping EZ and writing about the initial development process, I've spent months maintaining, debugging, and evolving the language. Here's what I've learned about the long game of AI-assisted development.

---

## Bug Hunting with AI

Here's my actual bug hunting process: I write programs.

Not toy examples. Real programs. A banking software simulator. A text adventure game. I use EZ like a user would, and when something breaks, I document it. This sounds obvious, but it's easy to fall into the trap of only testing what you *think* might break.

The interesting bugs come from places you weren't looking.

### The NaN Problem

One bug that kept resurfacing involved `NaN` and `Inf`/`-Inf` comparisons. The issue wasn't that it was hard to fix - it's that it kept coming back in different forms. Why? Because different Claude Code sessions had different mental models for how these edge cases should be handled.

Session one decides `NaN == NaN` should return `false` (mathematically correct). Session two, with no memory of that decision, implements comparison logic that assumes `NaN` values are equal. Now you have inconsistent behavior buried in your codebase, and it only surfaces when an integration test runs a specific comparison.

This is the hidden cost of AI-assisted development: **consistency requires explicit documentation**. The AI doesn't remember last week's architectural decisions unless you tell it.

---

## Feature Planning - Vision vs. Scope Creep

Claude is helpful. Sometimes too helpful.

At one point during JSON implementation, Claude suggested adding generics to the language. Generics would make unmarshaling cleaner, more flexible, more "correct." It was a reasonable suggestion from a language design perspective.

I said no.

Here's my rule: **if I don't understand it, it doesn't belong in my language.** I don't fully understand generics. I can use them in other languages, but I couldn't explain the implementation to someone else. So they're not going in EZ.

This came up repeatedly. Claude suggested `let`, `var`, and `mut` keywords at various points - patterns borrowed from Rust, Swift, JavaScript. Each suggestion made sense in isolation. Each one would have added complexity that contradicted EZ's core philosophy: clarity over cleverness.

The AI doesn't know your vision unless you enforce it. Every session, every conversation, the AI is optimizing for "good code" by general standards. Your job is to optimize for *your* standards.

---

## Writing Tests - Your Safety Net

Let me tell you about the tests that weren't actually testing anything.

Up until around version 0.17.x, the integration tests that Claude had written weren't asserting anything meaningful. Tests were "passing" because they ran without crashing - not because they verified correct behavior. A function could return completely wrong output, and the test would pass.

I didn't catch this for months.

This is the uncomfortable truth about vibecoded projects: **you can't review every line**. You're trusting the AI to write correct code, and sometimes that trust is misplaced. Tests are supposed to be your safety net, but if the AI writes the tests too, who's testing the tests?

My solution: I understand all the integration tests because they're written in EZ syntax - the language I designed. I know what those tests are doing because I can read them like English.

The unit tests? Written in Go. I haven't looked at most of them. They pass, and I trust that, but I couldn't explain what half of them are verifying.

This is a gap I live with. You'll have gaps too. The question is whether you're honest about where they are.

---

## Keeping AI On Track

### The Runaway Problem

I learned to include this in my prompts: "Report back to me the moment you find a bug."

It doesn't work.

Claude finds a bug, mentions it in passing - "I found an issue with string parsing" - and then keeps going. Writing more tests. Investigating related code. Refactoring nearby functions. By the time it stops, the original bug is buried under fifteen new findings, and I'm scrolling up through terminal history trying to find what actually mattered.

The same thing happens with test writing. Ask Claude to write tests for a simple function, and you'll get fifteen test cases covering edge cases you didn't ask about. At some point you have to interrupt: "What are you doing? The first test passed. We're done."

**The AI doesn't know when to stop.** That's your job.

### Context Is Everything

At the start of every session, I give Claude a master prompt that points to my context directory. Inside that directory is a `Language-Spec` folder containing files about every aspect of EZ's design: philosophy, syntax, error handling, type system, everything.

This is the documentation-as-memory approach from my first article, in practice. Without it, every session starts from zero. With it, Claude at least has a chance of making decisions consistent with past decisions.

It's not perfect. The AI still loses the plot mid-conversation. But I've never had to restart a session entirely - I just course-correct. "No, we're not adding that. Read the design philosophy doc again."

---

## Honesty Hour

Let's be real.

**What part of the EZ codebase do I understand the least?** The AST. Nodes. The abstract syntax tree that powers the entire parser. I know what it does conceptually. I could not write it from scratch, and I could not debug a complex issue in it without AI assistance.

**Have I shipped code thinking "I hope this doesn't break"?** The whole thing. But specifically? The basics. Simple variable declarations. Functions. Return statements. Parameters.

That's the stuff everyone touches first. If a new user writes `temp x int = 5` and it breaks, EZ is dead on arrival. Those foundational features were written early, when I understood the codebase least, and they've been stable enough that I haven't had to revisit them.

Stable enough. Probably fine. I hope.

This is the reality of vibecoded projects. You're shipping software built on foundations you didn't fully verify. The AI wrote it, the tests pass, users haven't complained - so you move forward and try not to think about it too hard.

---

## Conclusion

Maintaining an AI-assisted project isn't harder than maintaining a traditional one. It's differently hard.

You trade "I don't know how to implement this" for "I don't fully understand what was implemented." You trade slow development for fast development with gaps in comprehension. You trade complete control for velocity.

The goal isn't a perfect codebase. It's a sustainable one. That means:

- **Document decisions** so future sessions stay consistent
- **Say no to features** that don't match your vision, even good ones
- **Understand your tests** - at least enough to know what they're actually testing
- **Know your gaps** and be honest about what you don't understand
- **Write real programs** with your own tools - that's where the bugs hide

The honeymoon ends. The real work begins. And if you're honest with yourself about what you've built, you might just keep it running.

---

*This is Part 2 of the Building EZ series. [Part 1 covers the initial development process and prompting framework.](https://marshallburns.me/blog/ez-article-one)*
