package goper

import (
	"bytes"
	"regexp"
	"strings"
)

var camelingRegex = regexp.MustCompile("[0-9A-Za-z]+")

// A Table represents the metadata associated with a table (columns, datatypes, etc)
type Table struct {
	Name    string
	Columns []Column
}

// A column has a name and a database type (hackily converted to a gotype)
type Column struct {
	Name   string
	DbType string
}

var typemap map[string]string = map[string]string{
	"int":              "*int64",
	"int unsigned": 	"*uint64",
	"integer":          "*int64",
	"integer unsigned": "*uint64",
	"decimal":          "*float64", //fixme
	"varchar":          "*string",
	"text":             "*string",
	"mediumtext":       "*string",
	"longtext":         "*string",
	"float":            "*float64",
	"real":             "*float64",
	"datetime":         "*string",
	"timestamp":        "*string",
	"enum":             "*string",
	"date":             "*string",
	"double":           "float64",
	"double precision": "*float64",
	"boolean":          "*bool",
	"char":             "*string",
	"bit":              "*int64",
	"longblob":         "*int64",
	"blob":             "[]byte",
	"bytea":            "[]byte",
	"bigint unsigned":  "*uint64", // fixme
	"bigint": 			"*int64",
	"tinyint":          "*int64",
	"tinyint unsigned": "*uint64",
	"table":            "",
	"set":              "*int64",
}

// Return the go type a database column should be mapped to.
// We always use pointers to handle null
func (this *Column) GoType(table *Table) string {
	if this.Name == "id" {
		return "*" + CamelCase(table.Name) + "ID"
	}

	if strings.HasSuffix(this.Name, "_id") {
		return "*" + upperSpecificName(CamelCase(this.Name))
	}

	actualTypeParts := strings.Split(strings.ToLower(this.DbType), " ")
	for key, value := range typemap {
		expectedTypeParts := strings.Split(key, " ")
		match := true
		if len(expectedTypeParts) == len(actualTypeParts) {
			for i, portion := range actualTypeParts {
				match = match && strings.Index(portion, expectedTypeParts[i]) == 0
			}
			if match {
				return value
			}
		}
	}

	panic("Unknown type:" + this.DbType)
}

// A helper function that maps strings like product_variant to ProductVariant
func CamelCase(src string) string {
	byteSrc := []byte(src)
	chunks := camelingRegex.FindAll(byteSrc, -1)
	for idx, val := range chunks {
		chunks[idx] = bytes.Title(val)
	}
	return string(bytes.Join(chunks, nil))
}
