# ez-philosophy.md  
### *The Design Philosophy of the EZ Programming Language*

---

## 1. Purpose of EZ
EZ exists to solve one problem:

> **Give absolute beginners a simple, readable, no-bullshit language  
> that still rewards mastery and deep technical skill.**

EZ is not trying to compete with “big” languages.  
EZ is trying to be the **best possible teaching + practical scripting language** with strict rules, clarity, and real structure.

---

## 2. Core Identity

### 2.1 Low Barrier to Entry
Beginners should be able to pick up EZ and write working code in seconds.  
This requires:

- simple syntax  
- explicit semantics  
- predictable behavior  
- easy-to-read error messages  
- no hidden magic  
- no overload of features  

Every feature must be self-explanatory to a newcomer.

### 2.2 High Skill Ceiling
EZ is also designed so advanced programmers can push it far beyond beginner-level tasks.

Experienced users should appreciate:

- strict, explicit typing  
- predictable performance rules  
- first-class functions  
- clean modules  
- straightforward interop (FFI in the future)  
- a simple yet expansive, clear standard library(@io, @math, @arrays, @maps, @sets, more to come)

The language should scale *with the developer*, not against them.

### 2.3 The Two Questions (Non-Negotiable)
Every design decision must pass these two tests:

1. **Does this make it easier for a beginner to start?**  
2. **Does this give an expert more power without confusing beginners?**

If the answer to either is “no,” the feature does not belong in EZ.

---

## 3. Anti-OOP Stance (Permanent and Philosophical)
EZ is intentionally **anti-OOP**.

OOP often creates:
- unnecessary abstractions  
- hidden behavior  
- complex hierarchies  
- mental overhead for beginners  

EZ rejects entirely:
- classes  
- objects  
- inheritance  
- interfaces  
- methods  
- OOP polymorphism  

Instead, EZ embraces:
- procedural programming  
- functional patterns where appropriate  
- clarity over abstraction  
- composition via simple data + functions  
- explicitness everywhere  

This is non-negotiable and defines the character of the language.

---

## 4. Design Pillars

### 4.1 Explicit Over Implicit
EZ will never guess what the programmer means.

Examples:
- no implicit type coercion  
- no automatic conversions  
- no magic imports  
- no hidden memory operations  

When something happens, the programmer *knows* it is happening.

### 4.2 One Obvious Way, With Deliberate Alternatives

Multiple ways to do the same thing can confuse beginners, so EZ must always have
**one obvious, canonical way** to do core things.

However, in some areas, EZ will also offer **deliberate alternatives** that give
experienced users more control or convenience, as long as they do not:
- change the fundamental mental model,
- create ambiguity,
- or become required knowledge for beginners.

For example, imports support both:
- `import @arrays`              // canonical, beginner-friendly form  
- `import arr@arrays`           // explicit alias form for advanced users  

Both are clear, both are explicit, and the canonical one is what beginners see first.

In general, EZ should have:
- one *canonical* way to declare variables  
- one *canonical* way to declare functions  
- one *canonical* way to import modules  
- one primary looping pattern that covers most cases  
- one recommended formatting style  

Additional forms are allowed **only if**:
- they are strictly additive (do not change semantics based on context),
- they remain easy to read even for a beginner,
- they mainly benefit advanced users (ergonomics, control, clarity),
- they are not required to understand or write basic EZ programs.

If a new feature introduces ambiguity, hides behavior, or forces newcomers to learn
multiple styles early, it gets rejected.


### 4.3 Simplicity as a First-Class Feature
Simplicity is not a lack of power.  
It is a design strength.

Simple languages:
- are easy to learn  
- are easy to read  
- are easy to debug  
- enforce solid fundamentals  

EZ aims to be one of the cleanest languages to read and write.

### 4.4 Predictable Performance
Even as an interpreted language (for now), EZ must behave predictably.

Programmers should have a consistent mental model of:
- cost of allocations  
- cost of function calls  
- cost of array operations  
- value vs reference semantics  

Performance transparency teaches beginners real concepts  
and gives experts room to optimize.

### 4.5 Beginner-Friendly Error Messages
An error message should:
- state what happened  
- show where it happened  
- explain why it happened  
- hint at the likely fix  

Tone should be clear, friendly, and instructive — not condescending.

Errors must teach, not frustrate.

---

## 5. Target Users

### 5.1 Beginners
People writing their first lines of code should feel comfortable in EZ.  
The language must never punish newcomers for not knowing advanced concepts.

### 5.2 Intermediate Developers
They should enjoy the clarity, predictability, and explicit rules.

### 5.3 Advanced / Systems Developers
Not the primary audience — but they should still respect EZ’s philosophy  
and find value in its strictness and composability.

---

## 6. Long-Term Vision

### 6.1 Short Term Goals
- A clean, stable interpreter  
- Solid syntax and semantics  
- Strong error handling  
- Simple, clear modules  
- A small but consistent standard library  
- Reliable tests for every new feature  

### 6.2 Mid-Term Goals
- Performance improvements  
- FFI or host-language interop  
- A more complete standard library  
- A structured, readable spec  

### 6.3 Long-Term Goals
- A full compiled version of EZ  
- Cross-platform binaries  
- Educational materials  
- A stable ecosystem for scripts and small tools  
- Maintain the core identity without bloat  

---

## 7. What EZ Will Never Become
EZ will **never** become any of the following:

- an OOP language  
- a “kitchen sink” general-purpose giant  
- feature-bloated or overly clever  
- ambiguous or magic-heavy  
- an imitation of another language  

EZ remains focused, small, clear, and principled.

---

## 8. Philosophy Summary
EZ is:

> **Simple for beginners.  
> Powerful for experts.  
> Explicit, predictable, and honest.  
> A teaching language that never sacrifices clarity.**

Every feature must reinforce this identity.  
Every decision must pass the two questions.

This is the soul of EZ.
