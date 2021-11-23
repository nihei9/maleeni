# maleeni

maleeni is a lexer generator for golang. maleeni also provides a command to perform lexical analysis to allow easy debugging of your lexical specification.

[![Test](https://github.com/nihei9/maleeni/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/nihei9/maleeni/actions/workflows/test.yml)

## Installation

Compiler:

```sh
$ go install github.com/nihei9/maleeni/cmd/maleeni@latest
```

Code Generator:

```sh
$ go install github.com/nihei9/maleeni/cmd/maleeni-go@latest
```

## Usage

### 1. Define your lexical specification

First, define your lexical specification in JSON format. As an example, let's write the definitions of whitespace, words, and punctuation.

```json
{
    "name": "statement",
    "entries": [
        {
            "kind": "whitespace",
            "pattern": "[\\u{0009}\\u{000A}\\u{000D}\\u{0020}]+"
        },
        {
            "kind": "word",
            "pattern": "[0-9A-Za-z]+"
        },
        {
            "kind": "punctuation",
            "pattern": "[.,:;]"
        }
    ]
}
```

Save the above specification to a file in UTF-8. In this explanation, the file name is `statement.json`.

### 2. Compile the lexical specification

Next, generate a DFA from the lexical specification using `maleeni compile` command.

```sh
$ maleeni compile -l statement.json -o statementc.json
```

### 3. Debug (Optional)

If you want to make sure that the lexical specification behaves as expected, you can use `maleeni lex` command to try lexical analysis without having to generate a lexer. `maleeni lex` command outputs tokens in JSON format. For simplicity, print significant fields of the tokens in CSV format using jq command.

⚠️ An encoding that `maleeni lex` and the driver can handle is only UTF-8.

```sh
$ echo -n 'The truth is out there.' | maleeni lex statementc.json | jq -r '[.kind_name, .lexeme, .eof] | @csv'
"word","The",false
"whitespace"," ",false
"word","truth",false
"whitespace"," ",false
"word","is",false
"whitespace"," ",false
"word","out",false
"whitespace"," ",false
"word","there",false
"punctuation",".",false
"","",true
```

The JSON format of tokens that `maleeni lex` command prints is as follows:

| Field        | Type              | Description                                                                                                                                           |
|--------------|-------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| mode_id      | integer           | An ID of a lex mode.                                                                                                                                  |
| mode_name    | string            | A name of a lex mode.                                                                                                                                 |
| kind_id      | integer           | An ID of a kind. This is unique among all modes.                                                                                                      |
| mode_kind_id | integer           | An ID of a lexical kind. This is unique only within a mode. Note that you need to use `KindID` field if you want to identify a kind across all modes. |
| kind_name    | string            | A name of a lexical kind.                                                                                                                             |
| row          | integer           | A row number where a lexeme appears.                                                                                                                  |
| col          | integer           | A column number where a lexeme appears. Note that `col` is counted in code points, not bytes.                                                         |
| lexeme       | array of integers | A byte sequense of a lexeme.                                                                                                                          |
| eof          | bool              | When this field is `true`, it means the token is the EOF token.                                                                                       |
| invalid      | bool              | When this field is `true`, it means the token is an error token.                                                                                      |

### 4. Generate the lexer

Using `maleeni-go` command, you can generate a source code of the lexer to recognize your lexical specification.

```sh
$ maleeni-go statementc.json
```

The above command generates the lexer and saves it to `statement_lexer.go` file. By default, the file name will be `{spec name}_lexer.json`. To use the lexer, you need to call `NewLexer` function defined in `statement_lexer.go`. The following code is a simple example. In this example, the lexer reads a source code from stdin and writes the result, tokens, to stdout.

```go
package main

import (
    "fmt"
    "os"
)

func main() {
    lex, err := NewLexer(NewLexSpec(), os.Stdin)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    for {
        tok, err := lex.Next()
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        if tok.EOF {
            break
        }
        if tok.Invalid {
            fmt.Printf("invalid: %#v\n", string(tok.Lexeme))
        } else {
            fmt.Printf("valid: %v: %#v\n", KindIDToName(tok.KindID), string(tok.Lexeme))
        }
    }
}
```

Please save the above source code to `main.go` and create a directory structure like the one below.

```
/project_root
├── statement_lexer.go ... Lexer generated from the compiled lexical specification (the result of `maleeni-go`).
└── main.go .............. Caller of the lexer.
```

Now, you can perform the lexical analysis.

```sh
$ echo -n 'I want to believe.' | go run main.go statement_lexer.go
valid: word: "I"
valid: whitespace: " "
valid: word: "want"
valid: whitespace: " "
valid: word: "to"
valid: whitespace: " "
valid: word: "believe"
valid: punctuation: "."
```

## More Practical Usage

See also [this example](example/README.md).

## Lexical Specification Format

The lexical specification format to be passed to `maleeni compile` command is as follows:

top level object:

| Field   | Type                   | Domain | Nullable | Description                                                                                                               |
|---------|------------------------|--------|----------|---------------------------------------------------------------------------------------------------------------------------|
| name    | string                 | id     | false    | A specification name.                                                                                                     |
| entries | array of entry objects | N/A    | false    | An array of entries sorted by priority. The first element has the highest priority, and the last has the lowest priority. |

entry object:

| Field    | Type             | Domain | Nullable | Description                                                                                                           |
|----------|------------------|--------|----------|-----------------------------------------------------------------------------------------------------------------------|
| kind     | string           | id     | false    | A name of a token kind. The name must be unique, but duplicate names between fragments and non-fragments are allowed. |
| pattern  | string           | regexp | false    | A pattern in a regular expression                                                                                     |
| modes    | array of strings | N/A    | true     | Mode names that an entry is enabled in (default: "default")                                                           |
| push     | string           | id     | true     | A mode name that the lexer pushes to own mode stack when a token matching the pattern appears                         |
| pop      | bool             | N/A    | true     | When `pop` is `true`, the lexer pops a mode from own mode stack.                                                      |
| fragment | bool             | N/A    | true     | When `fragment` is `true`, its entry is a fragment.                                                                   |

See [Identifier](#identifier) and [Regular Expression](#regular-expression) for more details on `id` domain and `regexp` domain.

## Identifier

`id` represents an identifier and must follow the rules below:

* `id` must be a lower snake case. It can contain only `a` to `z`, `0` to `9`, and `_`.
* The first and last characters must be one of `a` to `z`.

## Regular Expression

`regexp` represents a regular expression. Its syntax is below:

⚠️ In JSON, you need to write `\` as `\\`.

### Composites

Concatenation and alternation allow you to combine multiple characters or multiple patterns into one pattern.

| Example  | Description           |
|----------|-----------------------|
| abc      | matches just 'abc'    |
| abc\|def | one of 'abc' or 'def' |

### Single Characters

In addition to using ordinary characters, there are other ways to represent a single character:

* dot expression
* bracket expressions
* code point expressions
* character property expressions
* escape sequences

The dot expression matches any one chracter.

| Example | Description       |
|---------|-------------------|
| .       | any one character |

The bracket expressions are represented by enclosing characters in `[ ]` or `[^ ]`. `[^ ]` is negation of `[ ]`. For instance, `[ab]` matches one of 'a' or 'b', and `[^ab]` matches any one character except 'a' and 'b'.

| Example | Description                                      |
|---------|--------------------------------------------------|
| [abc]   | one of 'a', 'b', or 'c'                          |
| [^abc]  | any one character except 'a', 'b', or 'c'        |
| [a-z]   | one in the range of 'a' to 'z'                   |
| [a-]    | 'a' or '-'                                       |
| [-z]    | '-' or 'z'                                       |
| [-]     | '-'                                              |
| [^a-z]  | any one character except the range of 'a' to 'z' |
| [a^]    | 'a' or '^'                                       |

The code point expressions match a character that has a specified code point. The code points consists of a four or six digits hex string.

| Example    | Description               |
|------------|---------------------------|
| \u{000A}   | U+0A (LF)                 |
| \u{3042}   | U+3042 (hiragana あ)      |
| \u{01F63A} | U+1F63A (grinning cat 😺) |

The character property expressions match a character that has a specified character property of the Unicode. Currently, maleeni supports only `General_Category` and `White_Space`. When you omitted the equal symbol and a right-side value, maleeni interprets a symbol in `\p{...}` as the `General_Category` value.

| Example                     | Description                                        |
|-----------------------------|----------------------------------------------------|
| \p{General_Category=Letter} | any one character whose General_Category is Letter |
| \p{gc=Letter}               | the same as \p{General_Category=Letter}            |
| \p{Letter}                  | the same as \p{General_Category=Letter}            |
| \p{l}                       | the same as \p{General_Category=Letter}            |
| \p{White_Space=yes}         | any one character whose White_Space is yes         |
| \p{wspace=yes}              | the same as \p{White_Space=yes}                    |

As you escape the special character with `\`, you can write a rule that matches the special character itself.
The following escape sequences are available outside of bracket expressions.

| Example | Description |
|---------|-------------|
| \\.     | '.'         |
| \\?     | '?'         |
| \\*     | '*'         |
| \\+     | '+'         |
| \\(     | '('         |
| \\)     | ')'         |
| \\[     | '['         |
| \\\|    | '\|'        |
| \\\\    | '\\'        |

The following escape sequences are available inside bracket expressions.

| Example | Description |
|---------|-------------|
| \\^     | '^'         |
| \\-     | '-'         |
| \\]     | ']'         |

### Repetitions

The repetitions match a string that repeats the previous single character or group.

| Example | Description      |
|---------|------------------|
| a*      | zero or more 'a' |
| a+      | one or more 'a'  |
| a?      | zero or one 'a'  |

### Grouping

`(` and `)` groups any patterns.

| Example   | Description                                            |
|-----------|--------------------------------------------------------|
| a(bc)*d   | matches 'ad', 'abcd', 'abcbcd', and so on              |
| (ab\|cd)+ | matches 'ab', 'cd', 'abcd', 'cdab', abcdab', and so on |

### Fragment

The fragment is a feature that allows you to define a part of a pattern. This feature is useful for decomposing complex patterns into simple patterns and for defining common parts between patterns.
A fragment entry is defined by an entry whose `fragment` field is `true`, and is referenced by a fragment expression (`\f{...}`).
Fragment patterns can be nested, but they are not allowed to contain circular references.

For instance, you can define [an identifier of golang](https://golang.org/ref/spec#Identifiers) as follows:

```json
{
    "name": "id",
    "entries": [
        {
            "fragment": true,
            "kind": "unicode_letter",
            "pattern": "\\p{Letter}"
        },
        {
            "fragment": true,
            "kind": "unicode_digit",
            "pattern": "\\p{Number}"
        },
        {
            "fragment": true,
            "kind": "letter",
            "pattern": "\\f{unicode_letter}|_"
        },
        {
            "kind": "identifier",
            "pattern": "\\f{letter}(\\f{letter}|\\f{unicode_digit})*"
        }
    ]
}
```

## Lex Mode

Lex Mode is a feature that allows you to separate a DFA transition table for each mode.

`modes` field of an entry in a lexical specification indicates in which mode the entry is enabled. If `modes` field is empty, the entry is enabled only in the default mode. The compiler groups the entries and generates a DFA for each mode. Thus the driver can switch the transition table by switching modes. The mode switching follows `push` or `pop` field of each entry.

For instance, you can define a subset of [the string literal of golang](https://golang.org/ref/spec#String_literals) as follows:

```json
{
    "name": "string",
    "entries": [
        {
            "kind": "string_open",
            "pattern": "\"",
            "push": "string"
        },
        {
            "modes": ["string"],
            "kind": "char_seq",
            "pattern": "[^\\u{000A}\"\\\\]+"
        },
        {
            "modes": ["string"],
            "kind": "escaped_char",
            "pattern": "\\\\[abfnrtv\\\\'\"]"
        },
        {
            "modes": ["string"],
            "kind": "escape_symbol",
            "pattern": "\\\\"
        },
        {
            "modes": ["string"],
            "kind": "newline",
            "pattern": "\\u{000A}"
        },
        {
            "modes": ["string"],
            "kind": "string_close",
            "pattern": "\"",
            "pop": true
        },
        {
            "kind": "identifier",
            "pattern": "[A-Za-z_][0-9A-Za-z_]*"
        }
    ]
}
```

In the above specification, when the `"` mark appears in default mode (it's the initial mode), the driver transitions to the `string` mode and interprets character sequences (`char_seq`) and escape sequences (`escaped_char`). When the `"` mark appears the next time, the driver returns to the `default` mode.

```sh
$ echo -n '"foo\nbar"foo' | maleeni lex stringc.json | jq -r '[.mode_name, .kind_name, .lexeme, .eof] | @csv'
"default","string_open","""",false
"string","char_seq","foo",false
"string","escaped_char","\n",false
"string","char_seq","bar",false
"string","string_close","""",false
"default","identifier","foo",false
"default","","",true
```

The input string enclosed in the `"` mark (`foo\nbar`) are interpreted as the `char_seq` and the `escaped_char`, while the outer string (`foo`) is interpreted as the `identifier`. The same string `foo` is interpreted as different types because of the different modes in which they are interpreted.
