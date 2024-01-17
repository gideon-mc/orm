package orm

// A collection of tables.
// Use this instead of []orm.Table because it implements
// the correct SQL queries for getting and managing rows.
type Collection struct {
    Table *Table
}
