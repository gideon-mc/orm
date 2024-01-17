package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	internal "github.com/gideon-mc/orm/internal/orm"
	"github.com/gideon-mc/orm/internal/registry"
	"github.com/gideon-mc/orm/internal/source"
)

// Represents a single table/entity.
//
// Automatically sets PRIMARY KEY to "Id" field.
// The "Id" field is transformed into <table_name>_id in database.
// Foreign keys should start with "Fk" followed by the table struct name and
// have the type of the struct.
//
// Example:
//
//		type MyTable struct {
//		    *orm.Table
//		    Id string `type:"char(32)" with:""`
//	     IsCool bool `type:"boolean" with:"NOT NULL"`
//	     FkAnimal Animal `type:"" with:"NOT NULL"`
//		}
type Table struct {
	DB *Database
}

func isFieldDefined(field reflect.Value) bool {
	return field.Interface() != reflect.Zero(field.Type()).Interface()
}

func JoinMapValues(input map[string]string, sep string) string {
	values := make([]string, len(input))
	for _, value := range input {
		values = append(values, fmt.Sprintf("%v", value))
	}
	return strings.Join(values, sep)
}

func getQueryResult(rows *sql.Rows, cols []string) []string {
	rawResult := make([][]byte, len(cols))
	result := make([]string, len(cols))

	dest := make([]interface{}, len(cols))
	for i := range rawResult {
		dest[i] = &rawResult[i]
	}

	if !rows.Next() {
		return result
	}

	err := rows.Scan(dest...)
	if err != nil {
		internal.Logger.Panicln("Failed to scan row", err)
		return result
	}

	for i, raw := range rawResult {
		if raw == nil {
			result[i] = "\\N"
		} else {
			result[i] = string(raw)
		}
	}

	return result
}

// Get all fields that aren't equal to default value.
// Meanining they were explicitly set.
func GetProperties(table interface{}, format string, is_defined_only bool) []string {
	src := source.NewSource(table)
	fields := []string{}

	for i := 1; i < src.T.NumField(); i++ {
		field := src.V.Field(i)
		if is_defined_only == isFieldDefined(field) {
			fields = append(fields, src.FormatField(
				format,
				src.T.Field(i),
				i,
			))
		}
	}

	return fields
}

func parseString[T any](fn func(string) (T, error), value string) T {
	result, err := fn(value)
	if err != nil {
		internal.Logger.Panic(err)
	}
	return result
}

func parseStringWithBaseAndBits[T any](
	fn func(string, int, int) (T, error),
	value string,
	base int,
	bits int,
) T {
	result, err := fn(value, base, bits)
	if err != nil {
		internal.Logger.Panic(err)
	}
	return result
}

func parseStringWithBits[T any](fn func(string, int) (T, error), value string, bits int) T {
	result, err := fn(value, bits)
	if err != nil {
		internal.Logger.Panic(err)
	}
	return result
}

func setStructField(field reflect.Value, value string, destType string, is_recursive bool) {
	switch destType {
	case "string":
		field.SetString(value)
	case "bool":
		field.SetBool(parseString(strconv.ParseBool, value))
	case "int8":
		field.SetInt(parseStringWithBaseAndBits[int64](strconv.ParseInt, value, 10, 8))
	case "int16":
		field.SetInt(parseStringWithBaseAndBits[int64](strconv.ParseInt, value, 10, 16))
	case "int32":
		field.SetInt(parseStringWithBaseAndBits[int64](strconv.ParseInt, value, 10, 32))
	case "int64", "int":
		field.SetInt(parseStringWithBaseAndBits[int64](strconv.ParseInt, value, 10, 64))
	case "uint8":
		field.SetUint(parseStringWithBaseAndBits[uint64](strconv.ParseUint, value, 10, 8))
	case "uint16":
		field.SetUint(parseStringWithBaseAndBits[uint64](strconv.ParseUint, value, 10, 16))
	case "uint32":
		field.SetUint(parseStringWithBaseAndBits[uint64](strconv.ParseUint, value, 10, 32))
	case "uint64", "uint":
		field.SetUint(parseStringWithBaseAndBits[uint64](strconv.ParseUint, value, 10, 64))
	case "float32":
		field.SetFloat(parseStringWithBits[float64](strconv.ParseFloat, value, 32))
	case "float64":
		field.SetFloat(parseStringWithBits[float64](strconv.ParseFloat, value, 64))
	default:
		if !is_recursive {
			return
		}
	}
}

// Fills all undefined entity fields from database.
// Only works for a single entity.
// To query multiple entities use orm.Collection
//
// Example:
// orm.Populate(db, MyTable{Id:"hello"})
//
// is_recursive tells whether to query all underlying foreign objects.
// Can reduce unnecessary database queries.
func Populate[T interface{}](db *Database, table T, is_recursive bool) (T, bool) {
	src := source.NewSource(table)
	queryProps := GetProperties(table, "%Name=%Value", true)
	emptyProps := GetProperties(table, "%SQLName", false)
	emptyFields := GetProperties(table, "%RawName:%Variable", false)

	rows, err := db.SQL.Query(fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s LIMIT 1",
		strings.Join(emptyProps, ", "),
		src.Name(),
		strings.Join(queryProps, " AND "),
	))
	if err != nil {
		internal.Logger.Panic(err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		internal.Logger.Panicln("Failed to get columns", err)
		return table, false
	}

	has_written := false
	result := getQueryResult(rows, cols)
	for i, field := range emptyFields {
		if len(result[i]) == 0 {
			continue
		}
		has_written = true
		split := strings.SplitN(field, ":", 2)
		setStructField(
			reflect.ValueOf(&table).Elem().FieldByName(split[0]),
			result[i],
			split[1],
			is_recursive,
		)
	}

	return table, has_written
}

// Register tables before you use them. It will NOT perform any operations
// on the database. It simply creates a local map of all existing tables.
func RegisterTables(tables ...interface{}) {
	for _, table := range tables {
		src := source.NewSource(table)
		registry.Tables[src.Name()] = table
	}
}
