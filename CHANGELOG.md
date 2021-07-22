# Changelog

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
	* Character property expressions: `\p{Letter}`
	* Repetition: `*`, `+`, `?`
	* Grouping: `(...)`
	* Fragment: `\f{...}`
  * Mode transition

[Commits](https://github.com/nihei9/maleeni/commits/v0.1.0)
