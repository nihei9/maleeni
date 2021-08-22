# Changelog

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
