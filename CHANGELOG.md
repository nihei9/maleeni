# Changelog

## v0.6.1

* [#5](https://github.com/nihei9/maleeni/issues/5), [ec2233e](https://github.com/nihei9/maleeni/commit/ec2233e894245aa963598dcc4f7e144a4c6c3192) - Avoid panic on spelling inconsistencies errors.
* [#7](https://github.com/nihei9/maleeni/issues/7), [bff52b5](https://github.com/nihei9/maleeni/commit/bff52b5cfbe3701f37f73c57ff81249f8d647174) - Fix the calculation of inverse bracket expressions.

[Changes](https://github.com/nihei9/maleeni/compare/v0.6.0...v0.6.1)

## v0.6.0

* [12bfeb8](12bfeb83ae4a804d05c7f6eab5c6b2b972b7d8d2) - Refactor the UCD file parsers.
* [2359623](2359623e6e1a85047953ff8838850d5c0685430b) - Fix key of `generalCategoryCodePoints` map. Use the abbreviation `cn` of the general category value `unassigned` as a key of `generalCategoryCodePoints` map.
* [bedf0c1](bedf0c1c72a2e13e08fbaa221b8a4c3ccf3a57a7), [6ebbc8f](6ebbc8f9829bf0f3127367769c662d1a8f881a2d), [e9af227](e9af22730e68908f46c9aee3b35e133d34191bef), [301d02d](301d02dd659ae8dea326684984710729401b92d1) - Support `White_Space`, `Lowercase`, `Uppercase`, `Alphabetic`, and `Script` properties. (Meet [RL1.2 of UTS #18](https://unicode.org/reports/tr18/#RL1.2) partially)
* [10d0c5d](10d0c5dfeb9749f4226f86d5ac915718c5bec5c9) - Make character properties available in an inverse expression (Make [^\p{...}] available).
* [5ebc2f4](5ebc2f4e9aa55bb77d82da7d9915a4140cddfbfb) - Move all UCD-related processes to _ucd_ package.
* [cb9d92f](cb9d92f0b4e0097579f6e5da1dc6e2f063b532a9) - Make contributory properties unavailable except internal use.
* [f0870a4](f0870a4d2ec589bf5de268a54d51c1da197ed882) - Remove default value's code points of `General_Category`.
* [847bcc7](847bcc7c63e900e4abc2cba58dbeb85d36967624) - Move UTF8-related processes to _utf8_ package.
* [19b68a5](19b68a5ca013c1ff7562e608db7964973fd691b2), [e3195d8](e3195d8e77c84b036a1ec1e3e03dc6e6aba3c8a1), [d595194](d595194791483a71c5afaff2aa3f4b575a9d22b7), [4321811](4321811c496d877eb452a38081109b96e12bd1be) - Use `CPTree` and `byteTree` instead of AST. `CPTree` represents characters in code points, while `byteTree` represents characters in UTF-8 byte sequences. In the past, when we excluded a part of a character range, like `[^...]`, we needed to subtract byte sequences from each other. This process is complicated. Therefore, we simplified the UTF-8 related processing by calculating the character range in code points and then converting it to UTF-8 byte sequences.
* [46d49df](46d49df654e9e152680717830aec70b65e8c507c) - Make character properties unavailable in bracket expressions.
* [3ec662c](3ec662c34841bb5bcf05166d1b9efd800b1e9ea3) - Add tests of _compiler/parser_ package.
* [a630029](a630029b6cd4a1e61025f6c0a40e198b90802946) - Remove `--lex-spec` option from `maleeni compile` command.

[Changes](https://github.com/nihei9/maleeni/compare/v0.5.1...v0.6.0)

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
