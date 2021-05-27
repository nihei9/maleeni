# maleeni

maleeni provides a compiler that generates a portable DFA for lexical analysis and a driver for golang.

## Installation

```sh
$ go install ./cmd/maleeni
```

## Usage

First, define your lexical specification in JSON format. As an example, let's write the definitions of whitespace, words, and punctuation.

```json
{
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

Save the above specification to a file. In this explanation, the file name is lexspec.json.

Next, generate a DFA from the lexical specification using `maleeni compile` command.

```sh
$ maleeni compile -l lexspec.json -o clexspec.json
```

If you want to make sure that the lexical specification behaves as expected, you can use `maleeni lex` command to try lexical analysis without having to implement a driver.
`maleeni lex` command outputs tokens in JSON format. For simplicity, print significant fields of the tokens in CSV format using jq command.

```sh
$ echo -n 'The truth is out there.' | maleeni lex clexspec.json | jq -r '[.kind_name, .text, .eof] | @csv'
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

When using the driver, please import `github.com/nihei9/maleeni/driver` and `github.com/nihei9/maleeni/spec` package.
You can use the driver easily in the following way:

```go
// Read your lexical specification file.
f, err := os.Open(path)
if err != nil {
    // error handling
}
data, err := ioutil.ReadAll(f)
if err != nil {
    // error handling
}
clexspec := &spec.CompiledLexSpec{}
err = json.Unmarshal(data, clexspec)
if err != nil {
    // error handling
}

// Generate a lexer.
lex, err := driver.NewLexer(clexspec, src)
if err != nil {
    // error handling
}

// Perform lexical analysis.
for {
    tok, err := lex.Next()
    if err != nil {
        // error handling
    }
    if tok.Invalid {
        // An error token appeared.
        // error handling
    }
    if tok.EOF {
        // The EOF token appeared.
        break
    }

    // Do something using `tok`.
}
```

## More Practical Usage

See also [this example](example/README.md).

## Lexical Specification Format

The lexical specification format to be passed to `maleeni compile` command is as follows:

top level object:

| Field   | Type                   | Nullable | Description                                                                                                           |
|---------|------------------------|----------|-----------------------------------------------------------------------------------------------------------------------|
| entries | array of entry objects | false    | An array of entries sorted by priority. The first element has the highest priority, and the last has the lowest priority. |

entry object:

| Field    | Type             | Nullable | Description                                                                                                           |
|----------|------------------|----------|-----------------------------------------------------------------------------------------------------------------------|
| kind     | string           | false    | A name of a token kind. The name must be unique, but duplicate names between fragments and non-fragments are allowed. |
| pattern  | string           | false    | A pattern in a regular expression                                                                                     |
| modes    | array of strings | true     | Mode names that an entry is enabled in (default: "default")                                                           |
| push     | string           | true     | A mode name that the lexer pushes to own mode stack when a token matching the pattern appears                         |
| pop      | bool             | true     | When `pop` is `true`, the lexer pops a mode from own mode stack.                                                      |
| fragment | bool             | true     | When `fragment` is `true`, its entry is a fragment.                                                                   |

See [Regular Expression Syntax](#regular-expression-syntax) for more details on the regular expression syntax.

## Regular Expression Syntax

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

The character property expressions match a character that has a specified character property of the Unicode. Currently, maleeni supports only General_Category.

| Example                     | Description                                        |
|-----------------------------|----------------------------------------------------|
| \p{General_Category=Letter} | any one character whose General_Category is Letter |
| \p{gc=Letter}               | the same as \p{General_Category=Letter}            |
| \p{Letter}                  | the same as \p{General_Category=Letter}            |
| \p{l}                       | the same as \p{General_Category=Letter}            |

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
