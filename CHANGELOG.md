# Changelog

## v0.5.1

* [fe865a8](https://github.com/nihei9/maleeni/commit/fe865a812401c2c612f2cd17cedd4728dc4798f7) - Generate constant values representing mode IDs, mode names, kind IDs, and kind names.
* [7be1d27](https://github.com/nihei9/maleeni/commit/7be1d273429765907af0abad182666d77eb557e4) - Add `name` field to the lexical specification. maleeni uses the `name` field to generate a source file name of the lexer. For instance, when the name is _my\_lex_, the source file of the lexer is named _my\_lex\_lexer_.
* [cf4f533](https://github.com/nihei9/maleeni/commit/cf4f53332e9d99a3a9eccfe69e70f98769862c3a) - Keep the order of AST nodes constant. This change is intended to output the same transition table for the same inputs.
* [9f3a334](https://github.com/nihei9/maleeni/commit/9f3a33484b61b4291bf4093dbe145fb01a452299) - Remove `--debug` option from compile command.
* [a8ed73f](https://github.com/nihei9/maleeni/commit/a8ed73f786fa9dd28965e4bf915022eb4a90bbba) - Disallow upper cases in an identifier.
* [12658e0](https://github.com/nihei9/maleeni/commit/12658e068eb0ff4bde0cddfda6145ee34b800166) - Format the source code of a lexer maleeni-go generates.
* [60a5089](https://github.com/nihei9/maleeni/commit/60a508960e71c73c5a8b72eb60ab0ac39d4f347d) - Remove the `ModeName` and `KindName` fields from the `driver.Token` struct.

[Changes](https://github.com/nihei9/maleeni/compare/v0.5.0...v0.5.1)

## v0.5.0

* [6332aaf](https://github.com/nihei9/maleeni/commit/6332aaf0b6caf7e23d7b4ca59c06f193bfbf7329) - Remove `--debug` option from `maleeni lex` command.
* [f691b5c](https://github.com/nihei9/maleeni/commit/f691b5cb74492b97cc4adc9d02bf39633e768503), [96a555a](https://github.com/nihei9/maleeni/commit/96a555a00f000704c618c226485fa6d87ce66d9d) - Add `maleeni-go` command to generate a lexer that recognizes a specific lexical specification.

[Changes](https://github.com/nihei9/maleeni/compare/v0.4.0...v0.5.0)

## v0.4.0

* [893ebf5](https://github.com/nihei9/maleeni/commit/893ebf5524067c778650462b5efd1640fe6b54a7) - Use Go 1.16.
* [#1](https://github.com/nihei9/maleeni/issues/1), [82efd35](https://github.com/nihei9/maleeni/commit/82efd35e6f99af0eff0430fc32b825d5cb38ac4d) - Add lexeme positions to tokens.

[Changes](https://github.com/nihei9/maleeni/compare/v0.3.0...v0.4.0)

## v0.3.0

* [03e3688](https://github.com/nihei9/maleeni/commit/03e3688e3928c88c12107ea734c35281c814e0c0) - Add unique kind IDs to tokens.
* [2433c27](https://github.com/nihei9/maleeni/commit/2433c27f26bc1be2d9b33f6550482abc48fa31ef) - Change APIs.

[Changes](https://github.com/nihei9/maleeni/compare/v0.2.0...v0.3.0)

## v0.2.0

* [7e169f8](https://github.com/nihei9/maleeni/commit/7e169f85726a1a1067d08e92cbbb2707ffb4d7b0) - Support passive mode transition.
* [a30fe0c](https://github.com/nihei9/maleeni/commit/a30fe0c6abd9ffbaff20af3da00eeea50d407f42) - Add a function `spec.EscapePattern` to escape the special characters that appear in patterns.

[Changes](https://github.com/nihei9/maleeni/compare/v0.1.0...v0.2.0)

## v0.1.0

* maleeni v0.1.0, this is the first release, supports the following features.
  * Definitions of lexical specifications by regular expressions
	* Alternation: `|`
	* Dot expression: `.`
	* Bracket expressions: `[...]`, `[^...]`
	* Code point expressions: `\u{...}`
	* Character property expressions: `\p{...}`
	* Repetition: `*`, `+`, `?`
	* Grouping: `(...)`
	* Fragment: `\f{...}`
  * Mode transition

[Commits](https://github.com/nihei9/maleeni/commits/v0.1.0)
