package source

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gideon-mc/orm/internal/registry"
	"github.com/stoewer/go-strcase"
)

func getName(src *Source, field reflect.StructField, index int) string {
	if field.Name == "Id" {
		return fmt.Sprintf("%s_id", src.Name())
	}
	if strings.HasPrefix(field.Name, "Fk") {
		return fmt.Sprintf("%s_id", strcase.SnakeCase(field.Name))
	}
	return strcase.SnakeCase(field.Name)
}

func getType(src *Source, field reflect.StructField, index int) string {
	if strings.HasPrefix(field.Name, "Fk") {
		src := NewSource(registry.Tables[field.Type.Name()])
		return src.Name()
	}
	return field.Tag.Get("type")
}

// All available field formattings.
var Field = map[string]func(*Source, reflect.StructField, int) string{
	"%Name": getName,
	"%Type": getType,
	"%With": func(src *Source, field reflect.StructField, index int) string {
		if field.Name == "Id" {
			return fmt.Sprintf("PRIMARY KEY %s", field.Tag.Get("with"))
		}
		return field.Tag.Get("with")
	},
	"%Tags": func(src *Source, field reflect.StructField, index int) string {
		return fmt.Sprintf("%v", field.Tag)
	},
	"%Value": func(src *Source, field reflect.StructField, index int) string {
		if src.V.Field(index).Kind() == reflect.Struct {
			table := registry.Tables[strcase.SnakeCase(strings.ReplaceAll(field.Name, "Fk", ""))]
			tableValue := reflect.ValueOf(table)
			return fmt.Sprintf("%v", tableValue.FieldByName("Id"))
		}
		if src.V.Field(index).Kind() == reflect.String {
			return fmt.Sprintf("%q", src.V.Field(index).Interface())
		}
        if field.Tag.Get("type") == "timestamp" {
            return fmt.Sprintf("FROM_UNIXTIME(%v)", src.V.Field(index).Interface())
        }
		return fmt.Sprintf("%v", src.V.Field(index).Interface())
	},
	"%RawName": func(src *Source, field reflect.StructField, index int) string {
		return field.Name
	},
	"%Variable": func(src *Source, field reflect.StructField, index int) string {
		return field.Type.Name()
	},
	"%SQLType": func(src *Source, field reflect.StructField, index int) string {
		t := getType(src, field, index)
		t = strings.ReplaceAll(t, "boolean", "tinyint(1)")
		return t
	},
	"%SQLName": func(src *Source, field reflect.StructField, index int) string {
		name := getName(src, field, index)
		switch getType(src, field, index) {
		case "timestamp":
			return fmt.Sprintf("UNIX_TIMESTAMP(%s)", name)
		}
		return name
	},
}

// Formats the field based on source.Format.
func (src *Source) FormatField(format string, field reflect.StructField, index int) string {
	result := format
	for key, value := range Field {
		result = strings.ReplaceAll(result, key, value(src, field, index))
	}
	return strings.TrimSpace(strings.ReplaceAll(result, "  ", " "))
}

// Replace all keywords with how they are stored in the actual database.
func GetSQLWith(value string) string {
	value = strings.ReplaceAll(value, "true", "'1'")
	value = strings.ReplaceAll(value, "false", "'0'")
	return value
}
