package wire

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// jsonMarshalerType is the one interface that takes a subtree out of this
// walk's hands, and that is not a gap: the reference's naming policy is applied
// by the object converter from a type's property metadata, so a converter that
// writes its own members is not renamed there either
// `[source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:34-45,55-58 @ v10.11.11]`.
//
// encoding.TextMarshaler is deliberately not checked beside it. One would be a
// guard no case can reach: a TextMarshaler is written as a JSON string, and a
// string is neither an object nor an array, so the walk copies it before ever
// asking what produced it.
var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

// renameProperties rewrites the property names of an encoded document under the
// camelCase policy, and leaves dictionary keys exactly as they were.
//
// # Why it needs the value the document came from
//
// spec 3.0.2 converts property names at every depth and never converts a
// dictionary key. After encoding, the two are the same thing: `"Id":` is four
// bytes either way, and no amount of looking at the document says whether they
// came from a field or from a map. plan 10 rejected a pass over the bytes for
// exactly this reason — "the conversion has to happen where a field is still a
// field".
//
// So the walk carries the value beside the bytes. The bytes decide the
// structure, which is what makes this safe: where the two disagree — a type
// that wrote its own JSON, a shape this walk cannot account for — the bytes are
// still the encoder's and the document stays well formed. What the value
// decides is only the question the bytes cannot answer, whether an object's keys
// are property names or dictionary keys.
//
// Everything else is copied byte for byte. Numbers, strings, `null` and the
// spelling of every escape are the encoder's own output, so the camelCase body
// differs from the PascalCase one in its property names and in nothing else —
// which is what AC-9 asserts.
//
// # What it refuses
//
// A value that cannot be told apart is an error rather than a silent copy. A
// body half converted is the failure behaviours 1.13 describes from the other
// side: a client that asked for camelCase and got PascalCase gets an empty
// object out of its decoder, and it would get one here for the fields the walk
// gave up on. Write sends nothing when this returns an error, so the caller can
// still refuse.
func renameProperties(document []byte, value reflect.Value) ([]byte, error) {
	r := &renamer{src: document, out: make([]byte, 0, len(document))}

	if err := r.value(value); err != nil {
		return nil, err
	}

	r.copyWhitespace()
	if r.i != len(r.src) {
		return nil, fmt.Errorf("wire: %d bytes after the end of the document", len(r.src)-r.i)
	}
	return r.out, nil
}

// renamer is one walk: the encoder's document, how far into it we are, and the
// document being built beside it.
type renamer struct {
	src []byte
	i   int
	out []byte
}

// keys says what an object's keys are, and types the value under one.
//
// The two answers travel together because they come from the same fact. A
// struct's keys are property names and its members are typed by its fields; a
// map's keys are dictionary keys and its members are typed by its element.
type keys struct {
	convert bool
	child   func(name string) reflect.Value
}

// value copies one JSON value, converting the property names inside it.
func (r *renamer) value(target reflect.Value) error {
	r.copyWhitespace()

	switch r.peek() {
	case 0:
		return fmt.Errorf("wire: the document ends where a value was expected")
	case '{':
		return r.object(target)
	case '[':
		return r.array(target)
	default:
		return r.copyValue()
	}
}

// object copies one JSON object, having decided from the value what its keys
// are.
func (r *renamer) object(target reflect.Value) error {
	v, opaque := resolve(target)
	if opaque {
		return r.copyValue()
	}

	switch {
	case v.IsValid() && v.Kind() == reflect.Struct:
		return r.members(structKeys(v))
	case v.IsValid() && v.Kind() == reflect.Map:
		return r.members(mapKeys(v))
	default:
		return fmt.Errorf(
			"wire: an object written from %s cannot be renamed: a property name "+
				"and a dictionary key are the same bytes, and only the value "+
				"tells them apart", describe(target))
	}
}

// members copies the members of an object, converting each key or not as k
// says, and typing each value from the name it was written under.
func (r *renamer) members(k keys) error {
	r.copyByte() // '{'

	r.copyWhitespace()
	if r.peek() == '}' {
		r.copyByte()
		return nil
	}

	for {
		r.copyWhitespace()
		raw, name, err := r.readKey()
		if err != nil {
			return err
		}

		if k.convert {
			quoted, err := encodeCompact(camelName(name))
			if err != nil {
				return err
			}
			r.out = append(r.out, quoted...)
		} else {
			r.out = append(r.out, raw...)
		}

		r.copyWhitespace()
		if r.peek() != ':' {
			return fmt.Errorf("wire: no colon after the key %q", name)
		}
		r.copyByte()

		if err := r.value(k.child(name)); err != nil {
			return err
		}

		r.copyWhitespace()
		switch r.peek() {
		case ',':
			r.copyByte()
		case '}':
			r.copyByte()
			return nil
		default:
			return fmt.Errorf("wire: no comma or closing brace after the member %q", name)
		}
	}
}

// array copies one JSON array, typing each element from the value's own.
func (r *renamer) array(target reflect.Value) error {
	v, opaque := resolve(target)
	if opaque {
		return r.copyValue()
	}
	if !v.IsValid() || (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) {
		return fmt.Errorf("wire: an array written from %s cannot be walked", describe(target))
	}

	r.copyByte() // '['

	r.copyWhitespace()
	if r.peek() == ']' {
		r.copyByte()
		return nil
	}

	for index := 0; ; index++ {
		var element reflect.Value
		if index < v.Len() {
			element = v.Index(index)
		}
		if err := r.value(element); err != nil {
			return err
		}

		r.copyWhitespace()
		switch r.peek() {
		case ',':
			r.copyByte()
		case ']':
			r.copyByte()
			return nil
		default:
			return fmt.Errorf("wire: no comma or closing bracket after element %d", index)
		}
	}
}

// readKey consumes one object key and returns both spellings of it: the bytes
// the encoder wrote, for a dictionary key that must not move, and the name it
// stands for, for a property name about to be converted and for the lookup that
// types its value.
func (r *renamer) readKey() (raw []byte, name string, err error) {
	if r.peek() != '"' {
		return nil, "", fmt.Errorf("wire: an object member does not start with a key")
	}

	end, err := endOfString(r.src, r.i)
	if err != nil {
		return nil, "", err
	}
	raw = r.src[r.i:end]
	r.i = end

	// A key with no backslash in it is its own content, which is every property
	// name this server sends. Anything else is decoded properly rather than
	// guessed at.
	if !strings.ContainsRune(string(raw), '\\') {
		return raw, string(raw[1 : len(raw)-1]), nil
	}
	if err := json.Unmarshal(raw, &name); err != nil {
		return nil, "", fmt.Errorf("wire: unreadable object key %s: %w", raw, err)
	}
	return raw, name, nil
}

// copyValue copies one whole JSON value without looking inside it.
func (r *renamer) copyValue() error {
	end, err := endOfValue(r.src, r.i)
	if err != nil {
		return err
	}
	r.out = append(r.out, r.src[r.i:end]...)
	r.i = end
	return nil
}

// peek is the byte the walk is on, or zero at the end of the document.
func (r *renamer) peek() byte {
	if r.i >= len(r.src) {
		return 0
	}
	return r.src[r.i]
}

// copyByte copies the byte the walk is on.
func (r *renamer) copyByte() {
	r.out = append(r.out, r.src[r.i])
	r.i++
}

// copyWhitespace copies any whitespace between two tokens. The encoder writes
// none, so this is what keeps the walk from depending on that.
func (r *renamer) copyWhitespace() {
	for r.i < len(r.src) {
		switch r.src[r.i] {
		case ' ', '\t', '\n', '\r':
			r.copyByte()
		default:
			return
		}
	}
}

// resolve unwraps a value to the one the encoder wrote a container from.
//
// opaque means the value writes its own JSON and its bytes are not this walk's
// to interpret. An invalid value with opaque false is a value that should have
// been a container and is not, which callers turn into an error.
func resolve(v reflect.Value) (resolved reflect.Value, opaque bool) {
	for v.IsValid() {
		t := v.Type()
		if t.Implements(jsonMarshalerType) {
			return reflect.Value{}, true
		}

		switch v.Kind() {
		case reflect.Pointer, reflect.Interface:
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		default:
			return v, false
		}
	}
	return reflect.Value{}, false
}

// structKeys says that an object's keys are property names, and types each one
// from the field it was written from.
func structKeys(v reflect.Value) keys {
	fields := structFields(v.Type())

	return keys{
		convert: true,
		child: func(name string) reflect.Value {
			index, ok := fields[name]
			if !ok {
				return reflect.Value{}
			}
			return fieldByIndex(v, index)
		},
	}
}

// mapKeys says that an object's keys are dictionary keys, which spec 3.0.2
// never converts, and types each value from the map itself.
//
// This is the branch the whole design is for. `ProviderIds`, `ImageTags` and
// `ImageBlurHashes` are dictionaries whose keys are provider and image names,
// and the reference leaves them alone because it sets `PropertyNamingPolicy`
// and never sets `DictionaryKeyPolicy` (behaviours 1.13).
func mapKeys(v reflect.Value) keys {
	keyType := v.Type().Key()
	elementType := v.Type().Elem()

	return keys{
		convert: false,
		child: func(name string) reflect.Value {
			if keyType.Kind() == reflect.String {
				return v.MapIndex(reflect.ValueOf(name).Convert(keyType))
			}
			// A key written from an integer or from a TextMarshaler cannot be
			// turned back into the key it came from here. The element type
			// still says what shape the value has, which is enough unless the
			// element is an interface or a pointer — and then the walk refuses
			// rather than converting half of it. No model in this API has a
			// dictionary keyed by anything but a string.
			return reflect.New(elementType).Elem()
		},
	}
}

// fieldByIndex walks an index path the way encoding/json does, without the
// panic reflect.Value.FieldByIndex raises on a nil embedded pointer. The
// encoder writes no member for a field under one, so an invalid value here
// only ever reaches a member that is not there.
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

// structFieldCache holds one field map per struct type. The maps are built
// once and never written to afterwards, so the walk of a thousand-item list
// reflects over the item type once rather than a thousand times.
var structFieldCache sync.Map // reflect.Type -> map[string][]int

// structFields maps the property names encoding/json writes for a struct type
// to the field each was written from.
func structFields(t reflect.Type) map[string][]int {
	if cached, ok := structFieldCache.Load(t); ok {
		return cached.(map[string][]int)
	}

	fields := buildStructFields(t)
	structFieldCache.Store(t, fields)
	return fields
}

// buildStructFields applies encoding/json's own rules for which fields a struct
// writes and under what names: the tag name where there is one, the field name
// otherwise, `-` skipped, unexported fields skipped, and an embedded struct
// without a tag name flattened into its parent.
//
// It is a reading of those rules rather than the rules themselves, so a name
// this reading misses is a name whose value the walk cannot type. That is an
// error and not a silent copy, which is what keeps the disagreement visible.
func buildStructFields(t reflect.Type) map[string][]int {
	type candidate struct {
		index  []int
		depth  int
		tagged bool
	}
	type embedded struct {
		t     reflect.Type
		index []int
	}

	byName := map[string][]candidate{}
	visited := map[reflect.Type]bool{}
	current := []embedded{{t: t}}

	for depth := 0; len(current) > 0; depth++ {
		var next []embedded

		for _, level := range current {
			if visited[level.t] {
				continue
			}
			visited[level.t] = true

			for i := 0; i < level.t.NumField(); i++ {
				field := level.t.Field(i)

				fieldType := field.Type
				if fieldType.Kind() == reflect.Pointer {
					fieldType = fieldType.Elem()
				}

				tag := field.Tag.Get("json")
				if tag == "-" {
					continue
				}
				name, _, _ := strings.Cut(tag, ",")
				if !isValidTagName(name) {
					name = ""
				}

				index := make([]int, 0, len(level.index)+1)
				index = append(append(index, level.index...), i)

				if field.Anonymous && !field.IsExported() {
					// An unexported embedded struct still promotes its exported
					// fields; an unexported embedded anything else is not
					// reachable at all.
					if fieldType.Kind() == reflect.Struct {
						next = append(next, embedded{t: fieldType, index: index})
					}
					continue
				}
				if !field.IsExported() {
					continue
				}
				if field.Anonymous && name == "" && fieldType.Kind() == reflect.Struct {
					next = append(next, embedded{t: fieldType, index: index})
					continue
				}

				tagged := name != ""
				if name == "" {
					name = field.Name
				}
				byName[name] = append(byName[name], candidate{index: index, depth: depth, tagged: tagged})
			}
		}

		current = next
	}

	fields := make(map[string][]int, len(byName))
	for name, candidates := range byName {
		shallowest := candidates[0].depth
		for _, c := range candidates {
			if c.depth < shallowest {
				shallowest = c.depth
			}
		}

		var winners []candidate
		for _, c := range candidates {
			if c.depth == shallowest {
				winners = append(winners, c)
			}
		}
		if len(winners) > 1 {
			// A tie at the same depth is written by whichever field is tagged,
			// and by neither when that does not settle it.
			var tagged []candidate
			for _, c := range winners {
				if c.tagged {
					tagged = append(tagged, c)
				}
			}
			winners = tagged
		}
		if len(winners) == 1 {
			fields[name] = winners[0].index
		}
	}

	return fields
}

// isValidTagName reads a json tag's name half the way encoding/json does: a
// name it would reject is not a name, and the field keeps its own.
func isValidTagName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

// describe names a value in an error message, including the case where there is
// no value to name.
func describe(v reflect.Value) string {
	if !v.IsValid() {
		return "a value this package could not account for"
	}
	return "a " + v.Type().String()
}

// endOfValue returns the index just past the JSON value starting at i.
func endOfValue(src []byte, i int) (int, error) {
	if i >= len(src) {
		return 0, fmt.Errorf("wire: the document ends where a value was expected")
	}

	switch src[i] {
	case '"':
		return endOfString(src, i)
	case '{', '[':
		return endOfContainer(src, i)
	default:
		// A number or one of the three bare literals, which end where the
		// document's own punctuation begins.
		end := i
		for end < len(src) && !isValueEnd(src[end]) {
			end++
		}
		if end == i {
			return 0, fmt.Errorf("wire: %q starts no value", src[i])
		}
		return end, nil
	}
}

// endOfContainer returns the index just past the object or array starting at i,
// counting brackets and stepping over strings so that a bracket inside one is
// not counted.
func endOfContainer(src []byte, i int) (int, error) {
	depth := 0

	for i < len(src) {
		switch src[i] {
		case '"':
			end, err := endOfString(src, i)
			if err != nil {
				return 0, err
			}
			i = end
		case '{', '[':
			depth++
			i++
		case '}', ']':
			depth--
			i++
			if depth == 0 {
				return i, nil
			}
		default:
			i++
		}
	}

	return 0, fmt.Errorf("wire: the document ends inside a container")
}

// endOfString returns the index just past the JSON string starting at i. A
// backslash consumes the byte after it, which is the same parity count the
// escape pass makes: an escaped quote is a character of the value and never the
// end of it.
func endOfString(src []byte, i int) (int, error) {
	if i >= len(src) || src[i] != '"' {
		return 0, fmt.Errorf("wire: no string where one was expected")
	}

	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case '"':
			return j + 1, nil
		}
	}

	return 0, fmt.Errorf("wire: the document ends inside a string")
}

// isValueEnd reports whether a byte ends an unquoted value.
func isValueEnd(c byte) bool {
	switch c {
	case ',', '}', ']', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}
