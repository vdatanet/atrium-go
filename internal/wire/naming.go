package wire

import "unicode"

// NamingCamel writes property names under the reference's camelCase policy: a
// leading run of capitals lowered all but the last, at every depth, and never
// applied to a dictionary key (spec 3.0.2, behaviours 1.13).
//
// It is one of the two policies the three declared content types answer under.
// The plain type and the `PascalCase` profile share NamingPascal; only
// `profile="CamelCase"` reaches this one
// `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`.
//
// The reference reaches this by setting one option and leaving another unset —
// `PropertyNamingPolicy = JsonNamingPolicy.CamelCase`, with `DictionaryKeyPolicy`
// never assigned
// `[source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:55-58 @ v10.11.11]`.
// That asymmetry is the whole of this file's difficulty: see rename.go.
const NamingCamel Naming = 1

// camelName converts one property name.
//
// # It is not "lower the first letter"
//
// behaviours 1.13 measured the reference's policy as .NET's, which lowers a
// leading run of capitals all but the last of them: `UICulture` becomes
// `uiCulture`, where lowering the first letter alone would give `uICulture`.
// Over the 1026 property names of the pinned document the two rules disagree on
// exactly one name, and `UICulture` is it — the other name with a leading run,
// `ETag`, becomes `eTag` under both. That is why the wrong rule survives a spot
// check, and why TestTheTwoRulesDisagreeOnExactlyOneName is written against the
// whole list rather than against a sample.
//
// The rule was measured over 281 conversions across nine endpoints under both
// profiles `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11,
// 2026-08-26]`.
//
// # What is inferred rather than measured
//
// Every name in that measurement was PascalCase, because every property name in
// the reference's own schema is (spec 3.0.1). Two clauses below can only fire on
// a name that is not — one that begins with a lower-case character, and a run
// broken by a space — and they are ⚠️ UNVERIFIED: they are what .NET's policy
// does, carried over so that this is that policy rather than an approximation of
// it, and no name this server sends can reach them.
//
// The loop works on runes where the reference works on UTF-16 code units. The
// two differ only for a capital letter outside the basic multilingual plane,
// which no property name has.
func camelName(name string) string {
	chars := []rune(name)

	// ⚠️ UNVERIFIED. A name that does not begin with a capital is returned
	// untouched, rather than having its leading run lowered from the second
	// character on: `aBC` stays `aBC` and does not become `abc`.
	if len(chars) == 0 || !unicode.IsUpper(chars[0]) {
		return name
	}

	for i := 0; i < len(chars); i++ {
		// A second character that is not a capital ends the run, which is the
		// ordinary case: `ServerName` is decided here, at i == 1.
		if i == 1 && !unicode.IsUpper(chars[i]) {
			break
		}

		hasNext := i+1 < len(chars)

		// The last capital of a run keeps its case when a lower-case character
		// follows it, because it belongs to the word that character starts.
		// This is the clause that makes `UICulture` into `uiCulture`.
		if i > 0 && hasNext && !unicode.IsUpper(chars[i+1]) {
			// ⚠️ UNVERIFIED. A space is not a word this capital could belong
			// to, so it is lowered after all.
			if chars[i+1] == ' ' {
				chars[i] = unicode.ToLower(chars[i])
			}
			break
		}

		chars[i] = unicode.ToLower(chars[i])
	}

	return string(chars)
}
