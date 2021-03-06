package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/cool-rest/rest-layer/schema"
)

var (
	// ErrNotImplemented is returned when the JSON schema encoding logic for a schema.FieldValidator has not (yet)
	// been implemented.
	ErrNotImplemented = errors.New("not implemented")
)

type errWriter struct {
	w   io.Writer
	err error
}

func (ew errWriter) writeFormat(format string, a ...interface{}) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, a...)
}

func (ew errWriter) writeString(s string) {
	if ew.err != nil {
		return
	}
	_, ew.err = ew.w.Write([]byte(s))
}

func (ew errWriter) write(b []byte) {
	if ew.err != nil {
		return
	}
	_, ew.err = ew.w.Write(b)
}

// boundariesToJSONSchema writes JSON Schema keys and values based on b, prefixed by a comma and without curly braces,
// to w. The prefixing comma is only written if at least one key/value pair is also written.
func boundariesToJSONSchema(w io.Writer, b *schema.Boundaries) error {
	if b == nil {
		return nil
	}
	ew := errWriter{w: w}

	if !math.IsNaN(b.Min) && !math.IsInf(b.Min, -1) {
		ew.writeFormat(`, "minimum": %s`, strconv.FormatFloat(b.Min, 'E', -1, 64))
	}
	if !math.IsNaN(b.Max) && !math.IsInf(b.Max, 1) {
		ew.writeFormat(`, "maximum": %s`, strconv.FormatFloat(b.Max, 'E', -1, 64))
	}
	return ew.err
}

// SchemaToJSONSchema writes JSON Schema keys and values based on v, without the outer curly braces, to w. Note
// that not all FieldValidator types are supported at the moment.
func validatorToJSONSchema(w io.Writer, v schema.FieldValidator) (err error) {
	if v == nil {
		return nil
	}
	ew := errWriter{w: w}
	switch t := v.(type) {
	case *schema.String:
		ew.writeString(`"type": "string"`)
		if t.Regexp != "" {
			ew.writeFormat(`, "pattern": %q`, t.Regexp)
		}
		if len(t.Allowed) > 0 {
			var allowed []string
			for _, value := range t.Allowed {
				allowed = append(allowed, fmt.Sprintf("%q", value))
			}
			ew.writeFormat(`, "enum": [%s]`, strings.Join(allowed, ", "))
		}
		if t.MinLen > 0 {
			ew.writeFormat(`, "minLength": %s`, strconv.FormatInt(int64(t.MinLen), 10))
		}
		if t.MaxLen > 0 {
			ew.writeFormat(`, "maxLength": %s`, strconv.FormatInt(int64(t.MaxLen), 10))
		}
	case *schema.Integer:
		ew.writeString(`"type": "integer"`)

		if len(t.Allowed) > 0 {
			var allowed []string
			for _, value := range t.Allowed {
				allowed = append(allowed, strconv.FormatInt(int64(value), 10))
			}
			ew.writeFormat(`, "enum": [%s]`, strings.Join(allowed, ","))
		}
		if ew.err == nil {
			ew.err = boundariesToJSONSchema(w, t.Boundaries)
		}
	case *schema.Float:
		ew.writeString(`"type": "number"`)
		if len(t.Allowed) > 0 {
			var allowed []string
			for _, value := range t.Allowed {
				allowed = append(allowed, strconv.FormatFloat(value, 'E', -1, 64))
			}
			ew.writeFormat(`, "enum": [%s]`, strings.Join(allowed, ","))
		}
		if ew.err == nil {
			ew.err = boundariesToJSONSchema(w, t.Boundaries)
		}
	case *schema.Array:
		ew.writeString(`"type": "array"`)
		if t.ValuesValidator != nil {
			ew.writeString(`, "items": {`)
			if ew.err == nil {
				ew.err = validatorToJSONSchema(w, t.ValuesValidator)
			}
			ew.writeString("}")
		}
	case *schema.Object:
		if ew.err == nil && t.Schema != nil {
			ew.err = schemaToJSONSchema(w, t.Schema)
		}
	case *schema.Time:
		ew.writeString(`"type": "string", "format": "date-time"`)
	case *schema.Bool:
		ew.writeString(`"type": "boolean"`)
	default:
		return ErrNotImplemented
	}
	return ew.err
}

// SchemaToJSONSchema writes JSON Schema keys and values based on s, without the outer curly braces, to w.
func schemaToJSONSchema(w io.Writer, s *schema.Schema) (err error) {
	if s == nil {
		return
	}

	ew := errWriter{w: w}
	if s.Description != "" {
		ew.writeFormat(`"description": %q, `, s.Description)
	}
	ew.writeString(`"type": "object", `)
	ew.writeString(`"additionalProperties": false, `)
	ew.writeString(`"properties": {`)
	var required []string
	var notFirst bool
	for key, field := range s.Fields {
		if notFirst {
			ew.writeString(", ")
		}
		notFirst = true
		ew.writeFormat("%q: {", key)
		if field.Description != "" {
			ew.writeFormat(`"description": %q, `, field.Description)
		}
		if field.Required {
			required = append(required, fmt.Sprintf("%q", key))
		}
		if field.ReadOnly {
			ew.writeFormat(`"readOnly": %t, `, field.ReadOnly)
		}
		if ew.err == nil {
			ew.err = validatorToJSONSchema(w, field.Validator)
		}
		if field.Default != nil {
			b, err := json.Marshal(field.Default)
			if err != nil {
				return err
			}
			ew.writeString(`, "default": `)
			ew.write(b)
		}
		ew.writeString("}")
		if ew.err != nil {
			break
		}
	}
	ew.writeString("}")

	if len(required) > 0 {
		ew.writeFormat(`, "required": [%s]`, strings.Join(required, ", "))
	}
	return ew.err
}

// Encoder writes the JSON Schema representation of a schema.Schema to an output stream. Note that only a sub-set of the
// FieldValidator types in the schema package is supported at the moment. Custom validators are also not yet handled.
// Attempting to encode a schema containing such fields will result in a ErrNotImplemented error.
type Encoder struct {
	io.Writer
}

// NewEncoder returns a new JSONSchema Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

// Encode writes the JSON Schema representation of s to the stream, followed by a newline character.
func (e *Encoder) Encode(s *schema.Schema) error {
	ew := errWriter{w: e.Writer}
	ew.writeString("{")
	if ew.err == nil {
		ew.err = schemaToJSONSchema(e.Writer, s)
	}
	ew.writeString("}\n")
	return ew.err
}
