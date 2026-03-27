#!/usr/bin/env python3
"""
fuzz.py - EZ language compiler fuzzer

Generates random EZ programs and feeds them to the ezc compiler,
looking for crashes, segfaults, hangs, and unexpected behavior.
The compiler should NEVER crash on any input — it should always
produce a clean error message or compile successfully.

Usage:
    python3 scripts/fuzz.py                    # Run 500 iterations
    python3 scripts/fuzz.py -n 2000            # Run 2000 iterations
    python3 scripts/fuzz.py --seed 42          # Reproducible run
    python3 scripts/fuzz.py --save-all         # Save all generated programs
    python3 scripts/fuzz.py --mode garbage     # Only run garbage input tests

Copyright (c) 2025-Present Marshall A Burns
Licensed under the MIT License. See LICENSE for details.
"""

import argparse
import os
import random
import signal
import string
import subprocess
import sys
import tempfile
import time

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

TIMEOUT_SECONDS = 10
EZC_PATH = None  # resolved at startup

TYPES = ["int", "float", "string", "bool", "char", "byte", "uint",
         "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64"]
KEYWORDS = ["mut", "const", "do", "if", "or", "otherwise", "for",
            "for_each", "as_long_as", "while", "loop", "when", "is",
            "default", "return", "break", "continue", "import", "using",
            "struct", "enum", "in", "not_in", "range", "nil", "true",
            "false", "ensure", "or_return", "new", "cast", "private",
            "temp", "module"]
OPERATORS = ["+", "-", "*", "/", "%", "**", "==", "!=", "<", ">",
             "<=", ">=", "&&", "||", "!", "=", "+=", "-=", "*=",
             "/=", "%=", "++", "--", "->"]
MODULES = ["@std", "@strings", "@arrays", "@maps", "@math", "@json",
           "@io", "@os", "@fmt", "@time", "@random", "@crypto",
           "@encoding", "@uuid", "@regex", "@csv", "@bytes", "@binary"]

# ---------------------------------------------------------------------------
# Generators — each produces a random EZ program string
# ---------------------------------------------------------------------------

def gen_valid_simple(rng):
    """Generate a simple valid program with random features."""
    lines = ["import @std", "using std", ""]
    num_funcs = rng.randint(0, 3)
    for i in range(num_funcs):
        ret_type = rng.choice(TYPES[:4])  # int, float, string, bool
        param_count = rng.randint(0, 4)
        params = ", ".join(f"p{j} {rng.choice(TYPES[:4])}" for j in range(param_count))
        lines.append(f"do helper_{i}({params}) -> {ret_type} {{")
        if ret_type == "int":
            lines.append(f"    return {rng.randint(-1000, 1000)}")
        elif ret_type == "float":
            lines.append(f"    return {rng.uniform(-1000, 1000):.2f}")
        elif ret_type == "string":
            lines.append(f'    return "hello"')
        elif ret_type == "bool":
            lines.append(f"    return {rng.choice(['true', 'false'])}")
        lines.append("}")
        lines.append("")

    lines.append("do main() {")
    num_stmts = rng.randint(1, 10)
    for _ in range(num_stmts):
        lines.append("    " + _gen_statement(rng))
    lines.append("}")
    return "\n".join(lines)


def gen_valid_structs(rng):
    """Generate a program with random struct definitions."""
    lines = ["import @std", "using std", ""]
    num_structs = rng.randint(1, 4)
    struct_names = []
    for i in range(num_structs):
        name = f"Type{i}"
        struct_names.append(name)
        num_fields = rng.randint(1, 6)
        lines.append(f"const {name} struct {{")
        for j in range(num_fields):
            ftype = rng.choice(TYPES[:5])
            lines.append(f"    field_{j} {ftype}")
        lines.append("}")
        lines.append("")

    lines.append("do main() {")
    for name in struct_names:
        lines.append(f'    println("{name} defined")')
    lines.append("}")
    return "\n".join(lines)


def gen_valid_enums(rng):
    """Generate a program with random enum definitions."""
    lines = ["import @std", "using std", ""]
    num_enums = rng.randint(1, 3)
    for i in range(num_enums):
        name = f"MyEnum{i}"
        attr = rng.choice(["", "#flags\n"])
        lines.append(f"{attr}const {name} enum {{")
        num_members = rng.randint(1, 8)
        for j in range(num_members):
            lines.append(f"    MEMBER_{j}")
        lines.append("}")
        lines.append("")

    lines.append("do main() {")
    lines.append('    println("enums defined")')
    lines.append("}")
    return "\n".join(lines)


def gen_valid_control_flow(rng):
    """Generate a program with nested control flow."""
    lines = ["import @std", "using std", ""]
    lines.append("do main() {")
    lines.append(f"    mut x int = {rng.randint(0, 100)}")
    lines.append(f"    mut y int = {rng.randint(0, 100)}")

    depth = rng.randint(1, 5)
    indent = 1
    for _ in range(depth):
        kind = rng.choice(["if", "for", "while", "when", "loop"])
        pad = "    " * indent
        if kind == "if":
            lines.append(f"{pad}if x > {rng.randint(0, 100)} {{")
            lines.append(f"{pad}    x = x + 1")
            if rng.random() < 0.5:
                lines.append(f"{pad}}} or x > {rng.randint(0, 50)} {{")
                lines.append(f"{pad}    x = x - 1")
            if rng.random() < 0.5:
                lines.append(f"{pad}}} otherwise {{")
                lines.append(f"{pad}    x = 0")
            lines.append(f"{pad}}}")
        elif kind == "for":
            end = rng.randint(1, 10)
            lines.append(f"{pad}for i in range(0, {end}) {{")
            lines.append(f"{pad}    x = x + 1")
            lines.append(f"{pad}}}")
        elif kind == "while":
            lines.append(f"{pad}mut counter_{indent} int = 0")
            lines.append(f"{pad}as_long_as counter_{indent} < {rng.randint(1, 5)} {{")
            lines.append(f"{pad}    counter_{indent}++")
            lines.append(f"{pad}}}")
        elif kind == "when":
            lines.append(f"{pad}when x {{")
            for v in range(rng.randint(1, 4)):
                lines.append(f"{pad}    is {v} {{")
                lines.append(f'{pad}        println("matched {v}")')
                lines.append(f"{pad}    }}")
            lines.append(f"{pad}    default {{")
            lines.append(f'{pad}        println("default")')
            lines.append(f"{pad}    }}")
            lines.append(f"{pad}}}")
        elif kind == "loop":
            lines.append(f"{pad}mut lc_{indent} int = 0")
            lines.append(f"{pad}loop {{")
            lines.append(f"{pad}    lc_{indent}++")
            lines.append(f"{pad}    if lc_{indent} >= {rng.randint(1, 3)} {{")
            lines.append(f"{pad}        break")
            lines.append(f"{pad}    }}")
            lines.append(f"{pad}}}")

    lines.append('    println("done")')
    lines.append("}")
    return "\n".join(lines)


def gen_valid_arrays_maps(rng):
    """Generate a program with array and map operations."""
    lines = ["import @std", "using std", ""]
    lines.append("do main() {")

    # Arrays
    size = rng.randint(0, 10)
    elems = ", ".join(str(rng.randint(-100, 100)) for _ in range(size))
    lines.append(f"    mut nums [int] = {{{elems}}}")
    lines.append(f"    println(len(nums))")

    if size > 0:
        lines.append(f"    for_each n in nums {{")
        lines.append(f"        println(n)")
        lines.append(f"    }}")

    # String array
    str_size = rng.randint(0, 5)
    str_elems = ", ".join(f'"s{i}"' for i in range(str_size))
    lines.append(f"    mut words [string] = {{{str_elems}}}")
    lines.append(f"    println(len(words))")

    # Map
    lines.append(f'    mut m map[string:int] = {{')
    for i in range(rng.randint(0, 5)):
        lines.append(f'        "key{i}": {rng.randint(0, 100)},')
    lines.append(f'    }}')
    lines.append(f"    println(len(m))")

    lines.append("}")
    return "\n".join(lines)


def gen_valid_string_interpolation(rng):
    """Generate programs with various string interpolation patterns."""
    lines = ["import @std", "using std", ""]
    lines.append("do main() {")

    lines.append(f"    mut x int = {rng.randint(0, 100)}")
    lines.append(f"    mut name string = \"test\"")
    lines.append(f"    mut flag bool = {rng.choice(['true', 'false'])}")

    num = rng.randint(1, 8)
    for _ in range(num):
        kind = rng.choice(["simple", "expr", "multi", "nested_field"])
        if kind == "simple":
            lines.append(f'    println("x = ${{x}}")')
        elif kind == "expr":
            lines.append(f'    println("x + 1 = ${{x + 1}}")')
        elif kind == "multi":
            lines.append(f'    println("${{name}} is ${{x}} and ${{flag}}")')
        elif kind == "nested_field":
            lines.append(f'    println("val=${{x}}, name=${{name}}")')

    lines.append("}")
    return "\n".join(lines)


def gen_valid_func_refs(rng):
    """Generate programs with function references."""
    lines = ["import @std", "using std", ""]

    num_funcs = rng.randint(1, 4)
    func_names = []
    for i in range(num_funcs):
        name = f"fn_{i}"
        func_names.append(name)
        lines.append(f"do {name}(n int) -> int {{")
        lines.append(f"    return n + {rng.randint(1, 10)}")
        lines.append("}")
        lines.append("")

    lines.append("do main() {")
    for name in func_names:
        lines.append(f"    mut ref_{name} = (){name}")
        lines.append(f"    println(ref_{name}({rng.randint(0, 50)}))")
    lines.append("}")
    return "\n".join(lines)


def gen_edge_empty(rng):
    """Generate empty or near-empty files."""
    return rng.choice(["", " ", "\n", "\n\n\n", "\t\t", "   \n   \n   "])


def gen_edge_comments_only(rng):
    """Generate files with only comments."""
    kinds = [
        "// just a comment",
        "// line 1\n// line 2\n// line 3",
        "/* block comment */",
        "/* multi\n   line\n   comment */",
        "// mixed\n/* block */\n// more",
        "/* nested /* not really */ end */",
    ]
    return rng.choice(kinds)


def gen_edge_long_lines(rng):
    """Generate a program with extremely long lines."""
    long_str = "a" * rng.randint(5000, 20000)
    return f'import @std\nusing std\ndo main() {{\n    println("{long_str}")\n}}'


def gen_edge_deep_nesting(rng):
    """Generate deeply nested expressions or blocks."""
    lines = ["import @std", "using std", "do main() {"]
    depth = rng.randint(20, 80)
    for i in range(depth):
        lines.append("    " * (i + 1) + "if true {")
    lines.append("    " * (depth + 1) + 'println("deep")')
    for i in range(depth, 0, -1):
        lines.append("    " * i + "}")
    lines.append("}")
    return "\n".join(lines)


def gen_edge_deep_expr(rng):
    """Generate deeply nested parenthesized expressions."""
    depth = rng.randint(20, 100)
    expr = "1"
    for _ in range(depth):
        op = rng.choice(["+", "-", "*"])
        expr = f"({expr} {op} {rng.randint(0, 10)})"
    return f"import @std\nusing std\ndo main() {{\n    mut x int = {expr}\n    println(x)\n}}"


def gen_edge_many_params(rng):
    """Generate a function with many parameters."""
    count = rng.randint(20, 100)
    params = ", ".join(f"p{i} int" for i in range(count))
    args = ", ".join(str(rng.randint(0, 10)) for _ in range(count))
    return (f"import @std\nusing std\n"
            f"do bigfunc({params}) -> int {{\n    return p0\n}}\n"
            f"do main() {{\n    println(bigfunc({args}))\n}}")


def gen_edge_many_vars(rng):
    """Generate a program declaring many variables."""
    lines = ["import @std", "using std", "do main() {"]
    count = rng.randint(50, 200)
    for i in range(count):
        t = rng.choice(["int", "string", "bool"])
        if t == "int":
            lines.append(f"    mut v{i} int = {rng.randint(0, 1000)}")
        elif t == "string":
            lines.append(f'    mut v{i} string = "s{i}"')
        else:
            lines.append(f"    mut v{i} bool = {rng.choice(['true', 'false'])}")
    lines.append(f"    println(v0)")
    lines.append("}")
    return "\n".join(lines)


def gen_garbage_random_bytes(rng):
    """Generate completely random bytes."""
    length = rng.randint(1, 500)
    return "".join(chr(rng.randint(1, 255)) for _ in range(length))


def gen_garbage_random_tokens(rng):
    """Generate random sequences of valid tokens."""
    all_tokens = KEYWORDS + TYPES + OPERATORS + [
        "(", ")", "{", "}", "[", "]", ",", ".", ":", ";",
        "@", "#", "^", '"hello"', "42", "3.14", "'a'",
        "true", "false", "nil", "myVar", "MyType", "_",
    ]
    count = rng.randint(5, 100)
    return " ".join(rng.choice(all_tokens) for _ in range(count))


def gen_garbage_keyword_soup(rng):
    """Generate random arrangements of keywords."""
    count = rng.randint(5, 50)
    return " ".join(rng.choice(KEYWORDS) for _ in range(count))


def gen_garbage_broken_strings(rng):
    """Generate programs with malformed strings."""
    cases = [
        'do main() { println("unterminated }',
        'do main() { println("nested "quotes" bad") }',
        'do main() { println("\\z invalid escape") }',
        "do main() { println(\"${\") }",
        'do main() { println("${}") }',
        'do main() { println("${${x}}") }',
        "do main() { println('ab') }",  # multi-char char literal
        'do main() { println("' + 'a' * 50000 + '") }',
    ]
    return rng.choice(cases)


def gen_garbage_type_abuse(rng):
    """Generate programs with nonsensical type usage."""
    cases = [
        "do main() { mut x int = \"hello\" }",
        "do main() { mut x string = 42 }",
        "do main() { mut x bool = 3.14 }",
        "do main() { mut x [int] = 5 }",
        "do main() { mut x int = true }",
        "do main() { const x int = x + 1 }",
        f"do main() {{ mut x [[[[[[[int]]]]]]] = {{}} }}",
        "do main() { mut x map[int:int:int] = {} }",
        "do main() { mut x map[] = {} }",
    ]
    return rng.choice(cases)


def gen_garbage_import_abuse(rng):
    """Generate programs with malformed imports."""
    cases = [
        "import\ndo main() {}",
        "import @\ndo main() {}",
        "import @nonexistent_module\ndo main() {}",
        "import @std @std\ndo main() {}",
        "import @std\nimport @std\nusing std\ndo main() { println(1) }",
        "using std\nimport @std\ndo main() {}",
        "import " + ", ".join(f"@mod{i}" for i in range(50)) + "\ndo main() {}",
    ]
    return rng.choice(cases)


def gen_mutation_valid_program(rng):
    """Take a valid program and randomly mutate it."""
    base = gen_valid_simple(rng)
    chars = list(base)
    num_mutations = rng.randint(1, 10)
    for _ in range(num_mutations):
        action = rng.choice(["delete", "insert", "replace", "swap"])
        if not chars:
            break
        if action == "delete":
            idx = rng.randint(0, len(chars) - 1)
            chars.pop(idx)
        elif action == "insert":
            idx = rng.randint(0, len(chars))
            chars.insert(idx, rng.choice(string.printable))
        elif action == "replace":
            idx = rng.randint(0, len(chars) - 1)
            chars[idx] = rng.choice(string.printable)
        elif action == "swap" and len(chars) > 1:
            i, j = rng.sample(range(len(chars)), 2)
            chars[i], chars[j] = chars[j], chars[i]
    return "".join(chars)


def gen_mutation_truncate(rng):
    """Generate a valid program and truncate it at a random point."""
    base = gen_valid_simple(rng)
    cut = rng.randint(1, len(base))
    return base[:cut]


def gen_mutation_repeat(rng):
    """Generate a valid program and repeat a random section."""
    base = gen_valid_simple(rng)
    if len(base) < 10:
        return base
    start = rng.randint(0, len(base) - 5)
    end = rng.randint(start + 1, min(start + 50, len(base)))
    chunk = base[start:end]
    repeats = rng.randint(2, 20)
    return base[:start] + chunk * repeats + base[end:]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _gen_statement(rng):
    """Generate a random statement for use inside a function body."""
    kind = rng.choice(["println", "var_int", "var_str", "var_bool", "assign"])
    if kind == "println":
        arg = rng.choice([
            f"{rng.randint(-100, 100)}",
            f'"msg_{rng.randint(0, 99)}"',
            f"{rng.choice(['true', 'false'])}",
            f'"x = ${{x}}"' if rng.random() < 0.3 else f'"{rng.randint(0, 99)}"',
        ])
        return f"println({arg})"
    elif kind == "var_int":
        return f"mut v_{rng.randint(0, 99)} int = {rng.randint(-100, 100)}"
    elif kind == "var_str":
        return f'mut s_{rng.randint(0, 99)} string = "val{rng.randint(0, 99)}"'
    elif kind == "var_bool":
        return f"mut b_{rng.randint(0, 99)} bool = {rng.choice(['true', 'false'])}"
    elif kind == "assign":
        return f"mut a_{rng.randint(0, 99)} int = {rng.randint(0, 50)} + {rng.randint(0, 50)}"
    return 'println("fallback")'


# ---------------------------------------------------------------------------
# Test runner
# ---------------------------------------------------------------------------

GENERATORS = {
    "valid": [
        ("simple", gen_valid_simple),
        ("structs", gen_valid_structs),
        ("enums", gen_valid_enums),
        ("control_flow", gen_valid_control_flow),
        ("arrays_maps", gen_valid_arrays_maps),
        ("interpolation", gen_valid_string_interpolation),
        ("func_refs", gen_valid_func_refs),
    ],
    "edge": [
        ("empty", gen_edge_empty),
        ("comments_only", gen_edge_comments_only),
        ("long_lines", gen_edge_long_lines),
        ("deep_nesting", gen_edge_deep_nesting),
        ("deep_expr", gen_edge_deep_expr),
        ("many_params", gen_edge_many_params),
        ("many_vars", gen_edge_many_vars),
    ],
    "garbage": [
        ("random_bytes", gen_garbage_random_bytes),
        ("random_tokens", gen_garbage_random_tokens),
        ("keyword_soup", gen_garbage_keyword_soup),
        ("broken_strings", gen_garbage_broken_strings),
        ("type_abuse", gen_garbage_type_abuse),
        ("import_abuse", gen_garbage_import_abuse),
    ],
    "mutation": [
        ("mutate_valid", gen_mutation_valid_program),
        ("truncate", gen_mutation_truncate),
        ("repeat", gen_mutation_repeat),
    ],
}


class FuzzResult:
    PASS = "pass"       # Compiled or errored cleanly
    CRASH = "crash"     # Segfault, abort, or nonzero exit without error message
    HANG = "hang"       # Exceeded timeout
    ICE = "ice"         # Internal compiler error (unexpected panic/assertion)


def run_compiler(source, mode="check"):
    """Run ezc on the given source. Returns (result_type, stderr, exit_code)."""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".ez", delete=False) as f:
        f.write(source)
        f.flush()
        tmp_path = f.name

    try:
        cmd = [EZC_PATH, mode, tmp_path]
        proc = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=TIMEOUT_SECONDS,
        )

        stderr = proc.stderr + proc.stdout
        code = proc.returncode

        # Negative return code means killed by signal (e.g., -11 = SIGSEGV)
        if code < 0:
            sig = -code
            sig_name = signal.Signals(sig).name if sig in signal._value2member_map_ else f"signal {sig}"
            return FuzzResult.CRASH, f"Killed by {sig_name}", code

        # Check for signs of internal error
        lower = stderr.lower()
        if any(s in lower for s in ["segmentation fault", "bus error",
                                     "abort", "assertion failed",
                                     "stack overflow", "core dumped"]):
            return FuzzResult.CRASH, stderr, code

        return FuzzResult.PASS, stderr, code

    except subprocess.TimeoutExpired:
        return FuzzResult.HANG, f"Timed out after {TIMEOUT_SECONDS}s", -1

    finally:
        try:
            os.unlink(tmp_path)
        except OSError:
            pass


def find_ezc():
    """Locate the ezc binary."""
    candidates = [
        os.environ.get("EZC_PATH"),
        os.path.join(os.path.dirname(__file__), "..", "ezc", "ezc"),
        "ezc",
    ]
    for c in candidates:
        if c and os.path.isfile(c) and os.access(c, os.X_OK):
            return os.path.abspath(c)
    return None


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    global EZC_PATH

    parser = argparse.ArgumentParser(description="EZ language compiler fuzzer")
    parser.add_argument("-n", "--iterations", type=int, default=500,
                        help="Number of fuzz iterations (default: 500)")
    parser.add_argument("--seed", type=int, default=None,
                        help="Random seed for reproducibility")
    parser.add_argument("--save-all", action="store_true",
                        help="Save all generated programs to fuzz_output/")
    parser.add_argument("--mode", choices=["all", "valid", "edge", "garbage", "mutation"],
                        default="all", help="Which generators to run")
    parser.add_argument("--ezc", type=str, default=None,
                        help="Path to ezc binary")
    parser.add_argument("--clean", action="store_true",
                        help="Remove all fuzz artifacts (crash files, fuzz_output/) and exit")
    args = parser.parse_args()

    if args.clean:
        import glob
        removed = 0
        for f in glob.glob("fuzz_crash_*.ez"):
            os.unlink(f)
            removed += 1
        if os.path.isdir("fuzz_output"):
            import shutil
            count = len(os.listdir("fuzz_output"))
            shutil.rmtree("fuzz_output")
            removed += count
        print(f"Cleaned up {removed} fuzz artifact(s).")
        sys.exit(0)

    EZC_PATH = args.ezc or find_ezc()
    if not EZC_PATH:
        print("Error: ezc binary not found. Build it first or pass --ezc PATH", file=sys.stderr)
        sys.exit(1)

    seed = args.seed if args.seed is not None else int(time.time())
    rng = random.Random(seed)
    print(f"Fuzzing ezc at: {EZC_PATH}")
    print(f"Seed: {seed}")
    print(f"Iterations: {args.iterations}")
    print()

    # Collect generators
    if args.mode == "all":
        gens = []
        for category in GENERATORS.values():
            gens.extend(category)
    else:
        gens = GENERATORS[args.mode]

    if args.save_all:
        os.makedirs("fuzz_output", exist_ok=True)

    # Stats
    stats = {FuzzResult.PASS: 0, FuzzResult.CRASH: 0,
             FuzzResult.HANG: 0, FuzzResult.ICE: 0}
    crashes = []
    start_time = time.time()

    for i in range(args.iterations):
        gen_name, gen_func = rng.choice(gens)
        source = gen_func(rng)

        if args.save_all:
            with open(f"fuzz_output/fuzz_{i:05d}_{gen_name}.ez", "w") as f:
                f.write(source)

        result, output, code = run_compiler(source)
        stats[result] += 1

        if result != FuzzResult.PASS:
            crash_file = f"fuzz_crash_{len(crashes):03d}_{gen_name}.ez"
            crash_path = os.path.abspath(crash_file)
            with open(crash_path, "w") as f:
                f.write(source)
            crashes.append({
                "iteration": i,
                "generator": gen_name,
                "result": result,
                "output": output[:500],
                "exit_code": code,
                "file": crash_path,
                "seed_offset": i,
            })
            print(f"  [{i:5d}] CRASH  ({gen_name}) -> {crash_path}")
            print(f"          {output[:200]}")
        else:
            if (i + 1) % 50 == 0:
                elapsed = time.time() - start_time
                rate = (i + 1) / elapsed
                print(f"  [{i + 1:5d}] {rate:.0f} tests/sec — "
                      f"{stats[FuzzResult.PASS]} pass, "
                      f"{stats[FuzzResult.CRASH]} crash, "
                      f"{stats[FuzzResult.HANG]} hang")

    # Summary
    elapsed = time.time() - start_time
    print()
    print("=" * 60)
    print(f"  Fuzzing complete: {args.iterations} iterations in {elapsed:.1f}s")
    print(f"  Seed: {seed}")
    print(f"  Rate: {args.iterations / elapsed:.0f} tests/sec")
    print()
    print(f"  Pass:  {stats[FuzzResult.PASS]}")
    print(f"  Crash: {stats[FuzzResult.CRASH]}")
    print(f"  Hang:  {stats[FuzzResult.HANG]}")
    print("=" * 60)

    if crashes:
        print()
        print(f"  {len(crashes)} CRASH(ES) FOUND:")
        for c in crashes:
            print(f"    [{c['iteration']}] {c['generator']}: {c['result']} "
                  f"(exit {c['exit_code']})")
            print(f"         File: {c['file']}")
            print(f"         Output: {c['output'][:150]}")
        print()
        print(f"  To reproduce: python3 scripts/fuzz.py --seed {seed}")
        sys.exit(1)
    else:
        print()
        print("  No crashes found.")
        sys.exit(0)


if __name__ == "__main__":
    main()
