package dialects

import (
	"github.com/xaionaro/reform"
	"github.com/xaionaro/reform/dialects/mssql"
	"github.com/xaionaro/reform/dialects/mysql"
	"github.com/xaionaro/reform/dialects/postgresql"
	"github.com/xaionaro/reform/dialects/sqlite3"
	"github.com/xaionaro/reform/dialects/sqlserver"
)

// ForDriver returns reform Dialect for given driver string, or nil.
func ForDriver(driver string) reform.Dialect {
	switch driver {
	case "postgres":
		return postgresql.Dialect
	case "mysql":
		return mysql.Dialect
	case "sqlite3":
		return sqlite3.Dialect
	case "mssql":
		return mssql.Dialect
	case "sqlserver":
		return sqlserver.Dialect
	default:
		return nil
	}
}
