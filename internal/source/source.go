package source

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/stoewer/go-strcase"
)

// Represents source of a certain variable.
type Source struct {
	T reflect.Type
	V reflect.Value
}

// Get a new Source of the type/value of an variable.
func NewSource(element any) Source {
	return Source{
		T: reflect.TypeOf(element),
		V: reflect.ValueOf(element),
	}
}

// Get name of the variable.
func (src Source) Name() string {
	if src.T == nil {
		return ""
	}
	return strcase.SnakeCase(src.T.Name())
}

// Get slice of all fields formatted.
func (src Source) Fields(format string) []string {
	fields := []string{}

	for i := 1; i < src.T.NumField(); i++ {
		fields = append(fields, src.FormatField(
			format,
			src.T.Field(i),
			i,
		))
	}

	return fields
}

// Get all fields that were explicitly specified.
func (src Source) DefinedFields(format string) []string {
	fields := []string{}

	for i := 1; i < src.T.NumField(); i++ {
		fmt.Println(src.V)
		if false {
			fields = append(fields, src.FormatField(
				format,
				src.T.Field(i),
				i,
			))
		}
	}

	return fields
}

// Check if formatted fields contain the specified value.
func (src Source) FieldsContain(format string, value string) bool {
	fields := src.Fields(format)
	value = strings.ToLower(value)

	for _, field := range fields {
		if strings.ToLower(field) == value {
			return true
		}
	}

	return false
}
