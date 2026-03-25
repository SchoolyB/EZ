# WASM Playground — Frozen

This WASM playground depends on the Go interpreter packages that were removed in EZ 3.0.
It will not compile until it is ported to use the ezc compiler via Emscripten (ezc + TCC in WASM).

For now, the pre-built `ez.wasm` binary from v2.x still works for the web demo.
