/*
 * playground.js - EZ Playground JavaScript glue
 *
 * Loads the WASM-compiled ezc compiler and provides
 * check/compile/run functionality.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

let wasmReady = false;
let Module = null;

// Load the WASM module
async function initWasm() {
    const statusEl = document.getElementById('status-text');
    statusEl.textContent = 'Loading compiler...';

    try {
        // The Emscripten-generated JS file defines a Module factory
        Module = await createEzcModule({
            // Redirect stdout/stderr to capture output
            print: (text) => appendOutput(text, 'success'),
            printErr: (text) => appendOutput(text, 'error'),
        });
        wasmReady = true;
        statusEl.textContent = 'Ready';
    } catch (e) {
        statusEl.textContent = 'Failed to load compiler: ' + e.message;
        console.error('WASM load error:', e);
    }
}

function getSource() {
    return document.getElementById('editor').value;
}

function setOutput(html) {
    document.getElementById('output').innerHTML = html;
}

function appendOutput(text, className) {
    const el = document.getElementById('output');
    const span = document.createElement('span');
    span.className = className || '';
    span.textContent = text + '\n';
    el.appendChild(span);
}

function checkCode() {
    if (!wasmReady) {
        setOutput('<span class="error">Compiler not loaded yet. Please wait...</span>');
        return;
    }

    const source = getSource();
    const start = performance.now();

    try {
        // Call the WASM ezc_check function
        const result = Module.ccall('ezc_check', 'string', ['string'], [source]);
        const elapsed = (performance.now() - start).toFixed(1);

        if (result === 'ok') {
            setOutput('<span class="success">No errors found.</span>');
        } else {
            setOutput('<span class="error">' + escapeHtml(result) + '</span>');
        }

        document.getElementById('status-time').textContent = elapsed + 'ms';
        document.getElementById('status-text').textContent = result === 'ok' ? 'Check passed' : 'Errors found';
    } catch (e) {
        setOutput('<span class="error">Internal error: ' + escapeHtml(e.message) + '</span>');
    }
}

function runCode() {
    if (!wasmReady) {
        setOutput('<span class="error">Compiler not loaded yet. Please wait...</span>');
        return;
    }

    const source = getSource();
    const start = performance.now();
    setOutput(''); // Clear output

    try {
        // Call the WASM ezc_compile function to get generated C
        const result = Module.ccall('ezc_compile', 'string', ['string'], [source]);
        const elapsed = (performance.now() - start).toFixed(1);

        if (result.startsWith('error')) {
            setOutput('<span class="error">' + escapeHtml(result) + '</span>');
            document.getElementById('status-text').textContent = 'Compilation failed';
        } else {
            // Show generated C code (until we have TCC execution)
            setOutput(
                '<span class="info">/* Generated C code — execution coming soon */</span>\n\n' +
                '<span class="success">' + escapeHtml(result) + '</span>'
            );
            document.getElementById('status-text').textContent = 'Compiled successfully';
        }

        document.getElementById('status-time').textContent = elapsed + 'ms';
    } catch (e) {
        setOutput('<span class="error">Internal error: ' + escapeHtml(e.message) + '</span>');
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Keyboard shortcut: Ctrl/Cmd + Enter to run
document.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault();
        runCode();
    }
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'Enter') {
        e.preventDefault();
        checkCode();
    }
});

// Tab key inserts spaces in the editor
document.getElementById('editor').addEventListener('keydown', (e) => {
    if (e.key === 'Tab') {
        e.preventDefault();
        const textarea = e.target;
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        textarea.value = textarea.value.substring(0, start) + '    ' + textarea.value.substring(end);
        textarea.selectionStart = textarea.selectionEnd = start + 4;
    }
});

// Auto-init when the page loads
// The WASM module script (ezc.js) must be loaded before this
if (typeof createEzcModule !== 'undefined') {
    initWasm();
} else {
    // Wait for the script to load
    window.addEventListener('load', () => {
        if (typeof createEzcModule !== 'undefined') {
            initWasm();
        } else {
            document.getElementById('status-text').textContent =
                'Compiler WASM not found — build with: make playground';
        }
    });
}
