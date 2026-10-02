package database

import "reflect"

// Dialect captures the two things that differ between the SQL backends:
// identifier quoting and bind-parameter style. Everything else is shared, so a
// single Table implementation serves MySQL, PostgreSQL and SQLite.
type Dialect struct {
	// ParamStyle is "q" for "?" placeholders or "pg" for "$1" numbering.
	ParamStyle string
	// AutoInc renders the auto-increment primary key clause.
	AutoInc string
	// Ignore is the insert prefix that makes a duplicate a no-op.
	Ignore string
	// OnConflict is appended for PostgreSQL, which has no IGNORE form.
	OnConflict string
}

var (
	// MySQL dialect.
	MySQLDialect = Dialect{ParamStyle: "q", AutoInc: "AUTO_INCREMENT", Ignore: "INSERT IGNORE "}

	// PostgreSQL dialect.
	PostgreSQLDialect = Dialect{ParamStyle: "pg", AutoInc: "GENERATED ALWAYS AS IDENTITY", OnConflict: " ON CONFLICT DO NOTHING"}

	// SQLite dialect.
	SQLiteDialect = Dialect{ParamStyle: "q", AutoInc: "AUTOINCREMENT", Ignore: "INSERT OR IGNORE "}
)

// Quote wraps an identifier for the dialect.
func (d Dialect) Quote(s string) string {
	if d.ParamStyle == "pg" {
		return `"` + s + `"`
	}
	return "`" + s + "`"
}

// QuoteTable wraps a possibly dotted reference such as "users" or "main.users".
func (d Dialect) QuoteTable(s string) string {
	out := ""
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '.' {
			out += d.Quote(s[start:i])
			if i < len(s) {
				out += "."
			}
			start = i + 1
		}
	}
	return out
}

// placeholder renders the nth bind parameter, 1-based.
func (d Dialect) placeholder(n int) string {
	if d.ParamStyle == "pg" {
		return "$" + itoa(n)
	}
	return "?"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// SQLType maps a Go kind to a column type usable across the three backends.
func (d Dialect) SQLType(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "TEXT"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INTEGER"
	case reflect.Int64, reflect.Uint64:
		return "BIGINT"
	case reflect.Bool:
		if d.ParamStyle == "pg" {
			return "BOOLEAN"
		}
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		if d.ParamStyle == "pg" {
			return "DOUBLE PRECISION"
		}
		return "DOUBLE"
	default:
		return "TEXT"
	}
}
