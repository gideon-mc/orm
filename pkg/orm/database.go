package orm

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	internal "github.com/gideon-mc/orm/internal/orm"
	"github.com/gideon-mc/orm/internal/registry"
	"github.com/gideon-mc/orm/internal/source"
	_ "github.com/go-sql-driver/mysql"
)

// Proxy for sql.DB that attaches custom methods
type Database struct {
	SQL *sql.DB
}

// Creates a new Database after connecting to MySQL using DSN.
// Panics if driver fails to connect or fails to ping the database.
//
// Example:
// db := orm.NewDatabase(os.Getenv("DSN"))
// defer db.SQL.Close()
func NewDatabase(dsn string) *Database {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		internal.Logger.Panicf("Failed to connect: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		internal.Logger.Panicf("Failed to ping: %v", err)
	}

	internal.Logger.Print("Connected to the database")
	return &Database{SQL: db}
}

// Checks that the query returned at least one row.
// Useful for verifying that an entity exists.
// Panics if there was an error during performing db.Query().
func (db *Database) isSuccessfull(query string, args ...any) bool {
	rows, err := db.SQL.Query(fmt.Sprintf(query, args...))
	if err != nil {
		internal.Logger.Panic(err)
		return false
	}
	defer rows.Close()

	if rows.Next() {
		return true
	}
	return false
}

// Executes a query and completely discards the result.
// Panics if there has been an error performing db.Exec().
func (db *Database) VoidExec(query string, args ...any) {
	_, err := db.SQL.Exec(fmt.Sprintf(query, args...))
	if err != nil {
		internal.Logger.Panic(err)
	}
}

// Syncronizes a single table. Used by Database.SyncTables().
func (db *Database) syncTable(wg *sync.WaitGroup, table interface{}) {
	defer wg.Done()
	src := source.NewSource(table)

	if !db.isSuccessfull("SHOW TABLES LIKE %q", src.Name()) {
		db.CreateTable(src)
		return
	}

	row := db.SQL.QueryRow(fmt.Sprintf("SHOW CREATE TABLE %s", src.Name()))
	if row == nil {
		return
	}

	var name, value string
	err := row.Scan(&name, &value)
	if err != nil {
		internal.Logger.Panic(err)
	}
	fields := strings.Split(internal.Regexp.CREATE_TABLE_ROWS(value), ",")
	for _, field := range fields {
		if strings.Contains(field, "PRIMARY KEY") {
			continue
		}

		// FIXME: Dont iterate over all fields when ur literally inside of a field already
		// name := internal.Regexp.FIELD_NAME(field)
		// if !src.FieldsContain("%Name", name) {
		// 	db.VoidExec("ALTER TABLE %s DROP %s", src.Name(), name)
		// }
		//
		// split := strings.SplitN(strings.TrimSpace(field), " ", 3)
		// fieldType, fieldWith := split[1], split[2]
		// if !src.FieldsContain("%SQLType", fieldType) ||
		// 	!src.FieldsContain("%With", source.GetSQLWith(fieldWith)) {
		// 	fmt.Printf(
		// 		"ALTER TABLE %s MODIFY COLUMN %s %s %s\n",
		// 		src.Name(),
		// 		name,
		// 		fieldType,
		// 		fieldWith,
		// 	)
		// }
		// if !src.FieldsContain("`%Name` %Type %With", strings.TrimSpace(field)) {
		//     internal.Logger.Println(strings.TrimSpace(field), "\n", strings.Join(src.Fields("`%Name` %Type %With"), ";"))
		// }
	}
}

// Synchronizes tables with tables on the SQL server.
// It creates tables that don't exist
// and alters tables that don't match the structure.
//
// Make sure to orm.RegisterTables() first.
func (db *Database) SyncTables() {
	var wg sync.WaitGroup
	for _, table := range registry.Tables {
		wg.Add(1)
		go db.syncTable(&wg, table)
	}
	wg.Wait()
}

// Creates an SQL table. Does not check if table already exists.
// Refer to Database.SyncTables() for syncronization instead.
func (db *Database) CreateTable(src source.Source) {
	db.VoidExec(
		"CREATE TABLE %s (%s)",
		src.Name(),
		strings.Join(src.Fields("`%Name` %Type %With"), ","),
	)
	internal.Logger.Printf("Created table %q", src.Name())
}

// Creates a row if it doesn't exist.
// Uses the PRIMARY KEY to find row inside of database.
func ClaimEntity[T interface{}](db *Database, entity T) T {
	src := source.NewSource(entity)
	primaryKey := strings.SplitN(src.Fields("%Name=%Value")[0], "=", 2)

	if db.isSuccessfull(
		"SELECT %s FROM %s WHERE %s",
		primaryKey[0],
		src.Name(),
		fmt.Sprintf("%s=%q", primaryKey[0], primaryKey[1]),
	) {
		return entity
	}

	// TODO: Foreign keys (Pain in the ass)
	db.VoidExec(
		"INSERT INTO %s (%s) VALUES (%s)",
		src.Name(),
		strings.Join(src.Fields("%Name"), ","),
		strings.Join(src.Fields("%Value"), ","),
	)

	internal.Logger.Printf("Claimed entity in %q", src.Name())
	return entity
}

func DeleteEntity[T interface{}](db *Database, entity T) T {
	src := source.NewSource(entity)
	primaryKey := src.Fields("%Name=%Value")[0]

	db.VoidExec(
		"DELETE FROM %s WHERE %s",
		src.Name(),
		primaryKey,
	)

	internal.Logger.Printf("Deleted entity from %q", src.Name())
	return entity
}
