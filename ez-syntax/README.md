# EZ Language Syntax Highlighting

Syntax highlighting for the EZ programming language.

## Installation

### For VS Code

1. Copy the `ez-syntax` folder to your VS Code extensions directory:
   - **macOS/Linux**: `~/.vscode/extensions/`
   - **Windows**: `%USERPROFILE%\.vscode\extensions\`

2. Restart VS Code

3. Open any `.ez` file and syntax highlighting will work automatically!

Alternatively, you can symlink this directory:
```bash
ln -s /Users/marshallburns/code/Project-Zen/ez-syntax ~/.vscode/extensions/ez-language
```

### For Zed

Zed can use TextMate grammars too!

1. Create Zed's extensions directory:
```bash
mkdir -p ~/.config/zed/extensions/ez
```

2. Copy the grammar file:
```bash
cp syntaxes/ez.tmLanguage.json ~/.config/zed/extensions/ez/
cp language-configuration.json ~/.config/zed/extensions/ez/
```

3. Create a Zed extension manifest at `~/.config/zed/extensions/ez/extension.json`:
```json
{
  "name": "EZ",
  "version": "0.1.0",
  "description": "EZ language support",
  "grammars": {
    "ez": {
      "path": "ez.tmLanguage.json",
      "scope_name": "source.ez"
    }
  },
  "languages": {
    "ez": {
      "extensions": ["ez"],
      "line_comment": "//",
      "block_comment": ["/*", "*/"]
    }
  }
}
```

4. Restart Zed

## Features

- Keywords: `do`, `temp`, `const`, `if`, `or`, `otherwise`, `for`, `for_each`, etc.
- Types: `int`, `float`, `string`, `char`, `bool`, arrays, structs
- Operators: arithmetic, logical, comparison, membership
- Comments: `//` and `/* */`
- Attributes: `@std`, `@suppress()`, etc.
- Module imports: `import @std`, `import alias@module`
- Functions and built-ins
- Enums and struct highlighting

## What's Highlighted

- **Purple/Blue**: Keywords (`if`, `for`, `do`, `temp`, `const`)
- **Green**: Strings and characters
- **Orange**: Numbers
- **Yellow/Gold**: Functions
- **Red/Pink**: Types and type names
- **Gray**: Comments
- **Cyan**: Constants (`true`, `false`, `nil`)

Colors depend on your VS Code/Zed theme!
