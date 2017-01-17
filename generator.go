package goper

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var logger *log.Logger

func init() {
	logger = log.New(ColourStream{os.Stderr}, " [XXXX] ", log.LstdFlags)
}

func upperSpecificName(str string) string {
	return regexp.MustCompile("(Id|Url)$").ReplaceAllStringFunc(str, strings.ToUpper)
}

// A SchemaWriter writes a set of tables to the writer denoted by Outfile
type SchemaWriter struct {
	PackageName    string
	Outfile        io.Writer
	RemoveFromType string
	Tables         []*Table
}

// Write the schema
func (this *SchemaWriter) WriteSchema() {
	//Write the package declaration
	fmt.Fprintf(this.Outfile, "package %s\n\n", this.PackageName)
	for _, table := range this.Tables {
		this.WriteType(table)
	}
}

// Write an individual table
func (this *SchemaWriter) WriteType(table *Table) {
	fmt.Fprintf(this.Outfile, "\ntype %s struct {\n", CamelCase(table.Name))
	maxln := 0
	for _, column := range table.Columns {
		if len(column.Name) > maxln {
			maxln = len(column.Name)
		}

	}
	var tableColumns []Column
	for _, column := range table.Columns {
		if column.DbType != "table" {
			this.WriteField(&column, maxln)
		} else {
			tableColumns = append(tableColumns, column)
		}
	}
	if len(tableColumns) > 0 {
		fmt.Fprint(this.Outfile, "\n")
	}
	for _, column := range tableColumns {
		this.WriteTableField(&column, maxln)
	}

	fmt.Fprintf(this.Outfile, "}\n")
}

func (this *SchemaWriter) WriteFunc(table *Table) {
	ct := CamelCase(table.Name)
	t := table.Name

	hasId := regexp.MustCompile("_id$")

	fmt.Fprintf(this.Outfile,
		`
type %sDB struct {
	SchemaContext
}

func (this *SchemaContext) %s() *%sDB {
	return &%sDB{*this}
}

func (this *%s) Table() string {
    return "%s"
}

func (this *%sDB) Get(id int) *%s {
    row := %s{}
    sql := "select * FROM %s WHERE id = ? LIMIT 1"
    err := this.db.Get(&row, sql, id)
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return nil
        } else {
            panic(err)
        }
    }
    return &row
}
`,
		t,
		ct, t, t,
		ct, t,
		t, ct, ct, t,
	)
	hasMultiId := 0
	for _, table_column := range table.Columns {
		if hasId.MatchString(table_column.Name) {
			hasMultiId++
			if hasMultiId > 1 {
				return
			}
		}
	}

	for _, col := range table.Columns {
		cn := col.Name
		ccn := upperSpecificName(CamelCase(cn))
		if hasId.MatchString(col.Name) {
			fmt.Fprintf(this.Outfile,
				`
func (this *%sDB) GetBy%s(id int) *[]%s {
    rows := []%s{}
    sql := "select * FROM %s WHERE %s = ?"
    err := this.db.Select(&rows, sql, id)
    if err != nil {
        if err.Error() == "sql: no rows in result set" {
            return nil
        } else {
            panic(err)
        }
    }
    return &rows
}

`,
				t, ccn, ct, ct, t, cn,
			)
		}
	}
}

// Write an individual field
func (this *SchemaWriter) WriteField(column *Column, maxln int) {
	maxlnstr := strconv.Itoa(maxln)

	name := upperSpecificName(CamelCase(column.Name))
	fmt.Fprintf(this.Outfile, "\t%-"+maxlnstr+"s %-10s `json:\"%s\" db:\"%s\"`\n",
		name, string(column.GoType()[1:]), column.Name, column.Name)
}

// Write an individual table field
func (this *SchemaWriter) WriteTableField(column *Column, maxln int) {
	ccn := CamelCase(column.Name)
	ccnType := ccn
	if this.RemoveFromType != "" {
		ccnType = regexp.MustCompile("^"+this.RemoveFromType).ReplaceAllString(ccn, "")
	}
	maxlnstr := strconv.Itoa(maxln)

	fmt.Fprintf(this.Outfile, "\t%-"+maxlnstr+"s %-10s `json:\"%s\" db:\"%s\"`\n",
		ccn, "*"+ccnType, column.Name, column.Name)
}

// Load the database schema into memory using introspection, populating .Tables
func (this *SchemaWriter) LoadSchema(driver string, schema string, db *sql.DB) error {
	dialect := DialectByDriver(driver)
	logger.Printf("schema: %s\n", schema)
	logger.Printf("db: %v\n", db)
	tables, err := db.Query(dialect.ListTables(schema))
	if err != nil {
		return err
	}
	fmt.Fprintf(this.Outfile, "package %s\n\n", this.PackageName)
	fmt.Fprint(this.Outfile, `
import (
    "github.com/jmoiron/sqlx"
    "fmt"
)

type SchemaContext struct {
    db *sqlx.DB
}
func (this *SchemaContext) ConnectDB(user, password, host, database string) error {
	if this.db != nil {
		this.db.Close()
		this.db = nil
	}
	db := sqlx.MustConnect("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, host, database))
	this.db = db
	return nil
}

`)
	for tables.Next() {
		var ignored sql.NullString
		t := new(Table)
		tables.Scan(&t.Name)
		cols, err := db.Query(dialect.ListColumns(schema, *t))
		if err != nil {
			return err
		}
		for cols.Next() {
			c := new(Column)
			if strings.EqualFold(dialect.Name(), "sqlite3") {
				err = cols.Scan(&ignored, &c.Name, &c.DbType,
					&ignored, &ignored, &ignored)
			} else {
				err = cols.Scan(&c.Name, &c.DbType)
			}
			if err != nil {
				panic(err)
			}
			t.Columns = append(t.Columns, *c)
			re := regexp.MustCompile("^(?P<table_name>.+)_id$")
			if re.MatchString(c.Name) {
				match := re.FindSubmatchIndex([]byte(c.Name))
				var dst []byte
				dst = re.ExpandString(dst, "$table_name", c.Name, match)
				newColumn := Column{
					Name:   string(dst),
					DbType: "table",
				}
				t.Columns = append(t.Columns, newColumn)
			}
		}
		this.Tables = append(this.Tables, t)
		this.WriteType(t)
	}

	// fmt.Fprintf(this.OutfileFunc, "import (\"fmt\")\n")
	for _, table := range this.Tables {
		this.WriteFunc(table)
	}

	return nil
}
