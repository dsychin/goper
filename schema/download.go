// schema -driver mysql|sqlite3|postgres -dsn dsn
// Generate a set of a golang structs
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/edocode/goper"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var dsn string
var driver string
var schema string
var logger *log.Logger
var verbose bool
var outfile string
var pkg string
var remove string
var create_column string
var update_column string
var ignore_tables string

func init() {
	flag.StringVar(&dsn, "dsn", "", "database dsn like 'user:password@tcp(127.0.0.1:3306)/main'")
	flag.StringVar(&driver, "driver", "mysql", "driver")
	flag.StringVar(&schema, "schema", "main", "schema")
	flag.StringVar(&outfile, "outfile", "", "file name ex: xxx specifys and xxx.go will be generated")
	flag.StringVar(&pkg, "package", "data", "package name")
	flag.StringVar(&remove, "remove", "", "remove string from head of type name")
	flag.StringVar(&create_column, "create_column", "create_time", "create time column name")
	flag.StringVar(&update_column, "update_column", "update_time", "update time column name")
	flag.StringVar(&ignore_tables, "ignore_tables", "", "specify table names separated with comma to exclude code generation target")
	flag.BoolVar(&verbose, "verbose", false, "Print debugging")
	flag.Parse()

	logger = log.New(goper.ColourStream{os.Stderr}, " [XXXX] ", log.LstdFlags)

	if dsn == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	conn, err := sql.Open(driver, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		panic(err)
	}
	err = conn.Ping()
	if err != nil {
		logger.Panic(err)
	} else if verbose {
		logger.Printf("Ping Worked\n")
	}
	var outSchema io.Writer
	if outfile == "" {
		outSchema = os.Stdout
	} else {
		f, err := os.Create(outfile)
		if err != nil {
			panic(err)
		}
		outSchema = f

		defer f.Close()
	}

	ignoreTables := make(map[string]bool)
	if ignore_tables != "" {
		tables := strings.Split(ignore_tables, ",")
		for i := 0; i < len(tables); i++ {
			ignoreTables[tables[i]] = true
		}
	}

	writer := &goper.SchemaWriter{
		Outfile:        outSchema,
		PackageName:    pkg,
		RemoveFromType: remove,
		CreateColumn:   create_column,
		UpdateColumn:   update_column,
		IgnoreTables:   ignoreTables,
	}
	//os.Stdout.Write([]byte(schema))
	err = writer.LoadSchema(driver, schema, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		panic(err)
	}
	if outfile != "" {
		exec.Command("go", "fmt", outfile).Run()
	}
}
