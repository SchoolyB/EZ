# I Built a Programming Language Using Only AI Prompts: Here's the Framework That Actually Worked

## The story of building EZ: from Python frustration to a working interpreter, powered entirely by collaborative AI development

---

Three years ago, I started learning to program with Python. Like most beginners, I was thrilled to see `"Hello, World!"` appear on my screen. But that honeymoon phase didn't last long.

The more I coded, the more frustrated I became. `if __name__ == "__main__"`, `__init__`, `self.something`, cryptic error messages, pip vs pip3, python vs python3 — Python was supposed to be "beginner-friendly," but it felt like death by a thousand paper cuts.

I tried C next. I liked it, but didn't get deep into memory management. Then JavaScript for job prospects. That was worse — NaN, truthy/falsy values, classes, no types, multiple specifications. I hated it.

**So I decided to build my own programming language.**

Not a toy. Not a learning exercise. A real, functional, interpreted language with a clear vision: **make programming actually simple for beginners** without the lies Python tells.

I called it **Otium** (Latin for "simple" or "easy"), designed the syntax I wished existed, and promptly gave up when I hit the hard parts.

Then I did it again with a new name: **EZ**. And gave up again.

But this year, after finishing my database project OstrichDB, I came back with a secret weapon: **Claude Code**.

**This time, I shipped.**

Today, [EZ is a working interpreted programming language](https://github.com/SchoolyB/EZ) with a lexer, parser, interpreter, REPL, type system, structs, enums, error handling, and a growing standard library. You can clone it, install it, and write code in it right now.

And I built it using nothing but AI prompts and my vision.

Here's the framework that made it possible.

---

## Why Build a Language with AI?

I had two reasons for using Claude Code to build EZ:

1. **Stress test the tool.** I wanted to see if AI could handle something complex — not just CRUD apps or scripts, but a full interpreter with semantics, parsing, and type checking.

2. **Actually finish what I started.** I'd already abandoned two language projects when complexity overwhelmed me. Claude Code let me tackle hard problems without giving up.

This wasn't about generating code mindlessly. It was about **collaborative development** where I provided vision and Claude Code provided implementation expertise.

---

## The Vision: What Makes EZ Different

Before writing a single line of code (or prompting Claude to write one), I needed crystal clarity on what EZ should be.

**What EZ is NOT:**
- Python with better syntax
- A systems language
- Production-ready (yet)

**What EZ IS:**
- An **educational language** for absolute beginners
- A bridge to typed languages like Go, Rust, and C
- **Opinionated and simple** above all else

Here's what sold me on the vision:

Imagine a new programmer searches "best language to learn programming" and finds EZ. They see:

✅ Simple, readable syntax
✅ Strong type system (smooth transition to Go/Rust/C)
✅ **No package manager, no dependencies**
✅ No OOP — just procedural and functional programming
✅ **Rust-inspired error messages that actually help**
✅ No classes, objects, or interfaces — just primitives, enums, and structs
✅ Cross-platform (write on any OS, run on any OS)

Compare that to Python's reality:
- Learn classes and OOP immediately
- Fight with package managers and virtualenvs
- Pray your package works with your Python version
- Read stack traces designed to confuse
- Discover dynamically typed code creates more bugs than it prevents

EZ says: **"Programming doesn't have to be this complicated."**

---

## The Prompting Framework That Actually Works

After hours of iteration, I developed a framework for prompting Claude Code that consistently produces quality results. Here's what I learned:

### 1. Start with a Context Foundation

Your first prompt is the most important. Don't just say "build me a lexer." **Define the entire landscape.**

Here's what I sent Claude Code on day one:

```
You are an expert Go language programmer and programming language
designer. You are tasked with helping me design and build an
interpreted programming language called EZ. EZ is meant to be a
low-barrier to entry, high skill ceiling programming language akin
to Go or Odin. EZ will be used for general purpose programming.

EZ will NOT have Classes, Objects, or Interfaces. Ideally EZ will
only be used with procedural or functional programming paradigms.

I will provide a comprehensive list of syntax, keywords, flows, and
expected behaviors that I want my language to have.

Please give me brutally honest feedback on my decisions. If an idea
is dumb and has no real use case, let me know. If an idea is good
but not quite fleshed out or is confusing, ask me questions so that
I can provide you more context.
```

**Why this worked:**
- Set expertise expectations (Go programmer, language designer)
- Defined the scope (interpreted, general purpose)
- Clarified constraints (no OOP)
- **Invited criticism** — this is crucial for collaborative development

### 2. Establish Operating Principles

Every subsequent prompt should reinforce your principles. I created a checklist that I reference constantly:

**My Prompt Principles:**
- ✅ Always tell the AI their role (critic, implementer, tester)
- ✅ Ensure changes are not code-breaking
- ✅ Ensure syntax/semantics align with my vision (documented in README)
- ✅ Ensure Go source code is high quality and readable with comments
- ✅ Always write or update tests for new features/fixes
- ✅ Always run the testing suite after work is done

### 3. Use Context Files, Not Conversation Tokens

**Biggest mistake beginners make:** Having long conversations with the AI explaining your vision over and over.

**What I do instead:** I created comprehensive documentation files:
- `README.md` — full syntax, features, examples
- `BUGS.md` — known issues with expected behavior
- `FEATURES.md` — planned features with specs

When I need something implemented, I reference these files:

```
Review the README.md syntax for enums. Implement enum support
according to the documented behavior. Ensure your implementation
matches the examples in the README exactly.
```

This approach:
- Saves tokens
- Maintains consistency
- Creates a single source of truth
- Forces you to think clearly about your design

### 4. Show Examples, Don't Just Explain

When I needed to add struct members that could be arrays, I didn't explain the problem. I **showed the bug:**

```ez
const Person struct {
  name string
  age int
  friends [Person]  // THIS SHOULD WORK BUT DOESN'T
}
```

Then I said:

```
This EZ code should work but throws an error. The expected behavior
is that struct members can be array types. Create a GitHub issue for
this, then implement a fix.
```

Claude Code immediately understood the problem, created [issue #60](https://github.com/SchoolyB/EZ/issues/60), and fixed it.

### 5. The Power of "Review and Test Everything"

This prompt became my secret weapon:

```
Review the README.md. Write a comprehensive testing suite in the
EZ programming language using the information from the README so
that we can check what features are implemented correctly AND
working, as well as which ones are not or causing bugs.

Create BUGS.md and FEATURES.md files and store your findings there.

After all bugs are found and tested, begin writing more comprehensive
tests by finding edge cases — maybe a user misspells a keyword or
forgets to enter the `return` keyword within a function that expects
a return value.
```

**What made this work:**
- It gave Claude Code autonomy to explore
- It created structured documentation automatically
- It found bugs I didn't know existed
- It caught edge cases I never considered

Claude Code found things like:
- Integers being returned as floats
- Type mismatches in array operations
- Unreachable code after return statements
- Missing error handling for malformed input

### 6. When Prompts Fall Short: Add More Context

I noticed Claude Code would occasionally miss my vision or implement something that felt "off."

**My mistake:** I hadn't given enough context.

**My fix:** I created a living document of principles (similar to the checklist above) and now reference it in every session:

```
Before starting, review Context_For_Claude.md which contains:
- Claudes role
- My vision for EZ
- Design principles
- What I like/dislike about other languages
- Who EZ is for
- Testing requirements
```

Since adding this, misalignments dropped to nearly zero.

---

## What Surprised Me Most

**Lesson #1: AI is patient; you need to be patient too.**

When a prompt didn't work, it wasn't Claude Code's fault. It was mine. I hadn't provided enough context, clear examples, or stated my expectations explicitly.

**Lesson #2: Vibe coding requires structure.**

"Vibe coding" doesn't mean chaos. It means **rapid iteration with clear vision**. The README became my spec. The tests became my validation. Claude Code became my implementation partner.

**Lesson #3: You still need to understand the domain.**

I couldn't have built EZ without understanding:
- How lexers tokenize input
- How parsers build ASTs
- How interpreters evaluate code
- How type systems work

Claude Code wrote the Go code, but **I designed the language.** That required real knowledge.

---

## The Current State of EZ

Today, EZ is functional but early-stage. You can:

```bash
# Install it
git clone https://github.com/SchoolyB/EZ.git
cd EZ
make install

# Write EZ code
ez examples/hello.ez

# Use the REPL
ez repl
```

Here's what works right now:

**✅ Core Features:**
- Variables (`temp`) and constants (`const`)
- Type system: `int`, `float`, `string`, `char`, `bool`
- Arrays (dynamic and fixed-size)
- Structs and enums
- Functions with return types
- Control flow (`if`/`or`/`otherwise`, `for`, `for_each`)
- While loops (`as_long_as`)
- String interpolation
- Module system with imports
- Interactive REPL
- Rust-inspired error messages

**🚧 In Progress:**
- Multiple return values (Go-style error handling)
- Expanded standard library (`@strings`, `@io`, `@math`)
- Type conversion functions

Check the [full feature list](https://github.com/SchoolyB/EZ#current-features) and [planned features](https://github.com/SchoolyB/EZ#planned-features) in the README.

---

## If You're Starting This Journey Tomorrow

Here's what I'd tell you:

### 1. Write Your Vision FIRST

Before touching the AI, create a document with:
- Your vision (what problem does this solve?)
- Your goals (what does success look like?)
- What you like/dislike about existing solutions
- Who this is for
- The domain (systems programming, web dev, scripting, etc.)
- How much time you'll commit

### 2. Don't Waste Tokens on Conversations

Open an empty file. Write your thoughts there. Let the AI read it. This creates:
- Clear documentation
- A single source of truth
- Efficient use of context windows

### 3. Prompt Structure Matters

Every good prompt includes:
- **Role:** "You are an expert..."
- **Context:** "We're building X which does Y..."
- **Task:** "Implement feature Z according to..."
- **Constraints:** "Ensure code is readable, tests pass, etc."
- **Success criteria:** "When done, X should behave like Y..."

### 4. Embrace the Feedback Loop

- Prompt → Review output → Refine prompt
- If something feels wrong, it probably is so add more context
- If Claude Code misunderstands, **you** weren't clear enough

### 5. Test Everything

My prompt framework always ends with:
```
After implementation, run the full test suite and report any failures.
```

This catches regressions immediately.

---

## What's Next for EZ

I'm working toward a v1.0 release with:
- Full standard library
- Go-style error handling
- Comprehensive documentation
- Real-world examples
- Performance improvements

But more importantly, I want EZ to become what I wished existed when I started: **a truly beginner-friendly language that doesn't lie about complexity.**

No hidden behaviors. No magic. No "you'll understand this later." Just clear, explicit code that does what it says.

---

## The Bigger Picture

Building EZ taught me that **AI-assisted development isn't about replacing programmers.** It's about **unlocking projects that would otherwise die from complexity.**

I'm not a compiler or interpreter expert. I'm not a Go wizard. But I **had a vision**, and Claude Code helped me implement it.

That's the real power of AI in development: **it removes the gap between vision and execution.**

If you have an idea for a tool, language, framework, or system but the implementation feels overwhelming, try this framework. Start with clarity, provide context, show examples, and iterate.

You might be surprised what you can build.

---

**Want to try EZ?** → [GitHub: SchoolyB/EZ](https://github.com/SchoolyB/EZ)
**Found this helpful?** Let me know what you're building with AI — I'd love to hear your story.

---

*This is the first in a series about building and maintaining EZ. Next up: the prompts that actually work (with real examples), repository workflow with AI, and advanced debugging techniques.*
