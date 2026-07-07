package toon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

var _ = sort.Strings

type Options struct {
	Indent       string
	StripMeta    bool
	MetaPatterns []string
}

func DefaultOptions() Options {
	return Options{
		Indent:       "  ",
		StripMeta:    false,
		MetaPatterns: []string{"created_at", "updated_at"},
	}
}

func LLMOptions() Options {
	opts := DefaultOptions()
	opts.StripMeta = true
	return opts
}

func Encode(v any) (string, error) {
	return EncodeWith(v, DefaultOptions())
}

func EncodeLLM(v any) (string, error) {
	return EncodeWith(v, LLMOptions())
}

func EncodeWith(v any, opts Options) (string, error) {
	var buf bytes.Buffer
	e := &encoder{w: &buf, opts: opts, depth: 0}
	if err := e.encodeValue(v, ""); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func WriteTo(w io.Writer, v any) error {
	e := &encoder{w: w, opts: DefaultOptions(), depth: 0}
	return e.encodeValue(v, "")
}

type encoder struct {
	w     io.Writer
	opts  Options
	depth int
}

func (e *encoder) indent() string {
	return strings.Repeat(e.opts.Indent, e.depth)
}

func (e *encoder) encodeValue(v any, key string) error {
	if v == nil {
		return e.writeKV(key, "~")
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return e.writeKV(key, "~")
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		if out, ok := marshalScalar(rv); ok {
			if needsQuote(out) {
				return e.writeKV(key, quoteString(out))
			}
			return e.writeKV(key, out)
		}
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return e.encodeSlice(key, rv)
	case reflect.Map:
		return e.encodeMap(key, rv)
	case reflect.Struct:
		return e.encodeStruct(key, rv)
	case reflect.String:
		s := rv.String()
		if needsQuote(s) {
			return e.writeKV(key, quoteString(s))
		}
		return e.writeKV(key, s)
	case reflect.Bool:
		return e.writeKV(key, strconv.FormatBool(rv.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.writeKV(key, strconv.FormatInt(rv.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e.writeKV(key, strconv.FormatUint(rv.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		return e.writeKV(key, strconv.FormatFloat(rv.Float(), 'f', -1, 64))
	default:
		return e.writeKV(key, fmt.Sprintf("%v", rv.Interface()))
	}
}

func marshalScalar(rv reflect.Value) (string, bool) {
	if t, ok := rv.Interface().(time.Time); ok {
		return t.Format(time.RFC3339Nano), true
	}
	if rv.CanAddr() {
		if m, ok := rv.Addr().Interface().(json.Marshaler); ok {
			b, err := m.MarshalJSON()
			if err != nil {
				return "", false
			}
			s := string(b)
			if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
				s = s[1 : len(s)-1]
			}
			return s, true
		}
	} else {
		rv2 := reflect.New(rv.Type()).Elem()
		rv2.Set(rv)
		if m, ok := rv2.Addr().Interface().(json.Marshaler); ok {
			b, err := m.MarshalJSON()
			if err != nil {
				return "", false
			}
			s := string(b)
			if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
				s = s[1 : len(s)-1]
			}
			return s, true
		}
	}
	return "", false
}

func (e *encoder) writeKV(key, value string) error {
	prefix := e.indent()
	if key != "" {
		prefix += key + ": "
	}
	_, err := fmt.Fprintf(e.w, "%s%s\n", prefix, value)
	return err
}

func (e *encoder) encodeSlice(key string, rv reflect.Value) error {
	if rv.Len() == 0 {
		return e.writeKV(key, "[]")
	}
	allStructs := true
	for i := 0; i < rv.Len(); i++ {
		ev := rv.Index(i)
		for ev.Kind() == reflect.Ptr {
			ev = ev.Elem()
		}
		if ev.Kind() != reflect.Struct {
			allStructs = false
			break
		}
	}
	if allStructs && e.canTabular(rv) {
		return e.encodeTabular(key, rv)
	}
	return e.encodeList(key, rv)
}

func (e *encoder) canTabular(rv reflect.Value) bool {
	if rv.Len() == 0 {
		return false
	}
	first := rv.Index(0)
	for first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	if first.Kind() != reflect.Struct {
		return false
	}
	firstType := first.Type()
	header := structFields(firstType)
	if len(header) == 0 {
		return false
	}
	for i := 1; i < rv.Len(); i++ {
		ev := rv.Index(i)
		for ev.Kind() == reflect.Ptr {
			ev = ev.Elem()
		}
		if ev.Type() != firstType {
			return false
		}
		for _, f := range header {
			fv := ev.FieldByName(f)
			if !fv.IsValid() {
				return false
			}
			if isComplex(fv) {
				return false
			}
		}
	}
	return true
}

func (e *encoder) encodeTabular(key string, rv reflect.Value) error {
	first := rv.Index(0)
	for first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	allHeader := structFields(first.Type())
	header := make([]string, 0, len(allHeader))
	headerStrs := make([]string, 0, len(allHeader))
	for i, fname := range allHeader {
		tagName := fieldTagOrName(first.Type().Field(i))
		if e.opts.StripMeta && e.isMetaField(tagName) {
			continue
		}
		header = append(header, fname)
		headerStrs = append(headerStrs, tagName)
	}
	headerLine := "{" + strings.Join(headerStrs, ",") + "}"
	if key == "" {
		if _, err := fmt.Fprintf(e.w, "%s[%d]%s:\n", e.indent(), rv.Len(), headerLine); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(e.w, "%s%s[%d]%s:\n", e.indent(), key, rv.Len(), headerLine); err != nil {
			return err
		}
	}
	e.depth++
	for i := 0; i < rv.Len(); i++ {
		ev := rv.Index(i)
		for ev.Kind() == reflect.Ptr {
			ev = ev.Elem()
		}
		vals := make([]string, len(header))
		for j, fname := range header {
			fv := ev.FieldByName(fname)
			vals[j] = formatScalar(fv)
		}
		if _, err := fmt.Fprintf(e.w, "%s%s\n", e.indent(), strings.Join(vals, ",")); err != nil {
			return err
		}
	}
	e.depth--
	return nil
}

func (e *encoder) encodeStruct(key string, rv reflect.Value) error {
	if key != "" {
		if _, err := fmt.Fprintf(e.w, "%s%s:\n", e.indent(), key); err != nil {
			return err
		}
		e.depth++
		defer func() { e.depth-- }()
	}
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := fieldTagOrName(f)
		if name == "-" {
			continue
		}
		if e.opts.StripMeta && e.isMetaField(name) {
			continue
		}
		if hasOmitempty(f) && isZero(rv.Field(i)) {
			continue
		}
		fv := rv.Field(i)
		if err := e.encodeValue(fv.Interface(), name); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) isMetaField(name string) bool {
	for _, p := range e.opts.MetaPatterns {
		if name == p {
			return true
		}
	}
	return false
}

func hasOmitempty(f reflect.StructField) bool {
	tag := f.Tag.Get("json")
	return strings.Contains(tag, ",omitempty")
}

func isZero(rv reflect.Value) bool {
	if !rv.IsValid() {
		return true
	}
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		return rv.IsNil() || rv.Len() == 0
	case reflect.String:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Struct:
		z := reflect.Zero(rv.Type()).Interface()
		return reflect.DeepEqual(rv.Interface(), z)
	default:
		return false
	}
}

func (e *encoder) encodeList(key string, rv reflect.Value) error {
	if key != "" {
		if _, err := fmt.Fprintf(e.w, "%s%s[%d]:\n", e.indent(), key, rv.Len()); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(e.w, "%s[%d]:\n", e.indent(), rv.Len()); err != nil {
			return err
		}
	}
	e.depth++
	for i := 0; i < rv.Len(); i++ {
		ev := rv.Index(i)
		if _, err := fmt.Fprintf(e.w, "%s- ", e.indent()); err != nil {
			return err
		}
		prevDepth := e.depth
		e.depth = 0
		err := e.encodeValue(ev.Interface(), "")
		if err != nil {
			e.depth = prevDepth
			return err
		}
		e.depth = prevDepth
	}
	e.depth--
	return nil
}

func (e *encoder) encodeMap(key string, rv reflect.Value) error {
	if rv.Len() == 0 {
		return e.writeKV(key, "{}")
	}
	if key != "" {
		if _, err := fmt.Fprintf(e.w, "%s%s:\n", e.indent(), key); err != nil {
			return err
		}
		e.depth++
		defer func() { e.depth-- }()
	}
	keys := rv.MapKeys()
	sorted := make([]string, 0, len(keys))
	keyMap := map[string]reflect.Value{}
	for _, k := range keys {
		ks := fmt.Sprintf("%v", k.Interface())
		sorted = append(sorted, ks)
		keyMap[ks] = rv.MapIndex(k)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		pi := priority(sorted[i])
		pj := priority(sorted[j])
		if pi != pj {
			return pi < pj
		}
		return sorted[i] < sorted[j]
	})
	for _, ks := range sorted {
		if err := e.encodeValue(keyMap[ks].Interface(), ks); err != nil {
			return err
		}
	}
	return nil
}

func priority(k string) int {
	switch k {
	case "ok":
		return 0
	case "data":
		return 1
	case "error":
		return 2
	default:
		return 10
	}
}

func structFields(t reflect.Type) []string {
	out := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if fieldTagOrName(f) == "-" {
			continue
		}
		out = append(out, f.Name)
	}
	return out
}

func fieldTagOrName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "-"
	}
	if tag == "" {
		return f.Name
	}
	if comma := strings.Index(tag, ","); comma >= 0 {
		tag = tag[:comma]
	}
	if tag == "" {
		return f.Name
	}
	return tag
}

func isComplex(rv reflect.Value) bool {
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	if _, ok := rv.Interface().(time.Time); ok {
		return false
	}
	if _, ok := rv.Addr().Interface().(json.Marshaler); ok {
		return false
	}
	return true
}

func formatScalar(rv reflect.Value) string {
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "~"
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return "~"
	}
	if t, ok := rv.Interface().(time.Time); ok {
		return t.Format(time.RFC3339Nano)
	}
	if rv.Kind() == reflect.Struct {
		if t, ok := rv.Addr().Interface().(json.Marshaler); ok {
			b, err := t.MarshalJSON()
			if err == nil {
				s := string(b)
				if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
					s = s[1 : len(s)-1]
				}
				if needsQuote(s) {
					return quoteString(s)
				}
				return s
			}
		}
	}
	switch rv.Kind() {
	case reflect.String:
		s := rv.String()
		if needsQuote(s) {
			return quoteString(s)
		}
		return s
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", rv.Interface())
	}
}

func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	if strings.ContainsAny(s, ",\":\n\\#[]{}") {
		return true
	}
	if strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return true
	}
	if s == "true" || s == "false" || s == "null" || s == "~" {
		return true
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	return false
}

func quoteString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
