# A derivative of "reform"

This's a derivative of a beautiful project "[reform](https://github.com/go-reform/reform)". Please look at the upstream code first.

This project just adds some functionality to the upstream ORM to make it usable in ActiveRecord ways. You shouldn't use this fork unless you want to use this functionality.

Added functionality:
* `ModelName.Select()` — a wrapper for SelectRows() and NextRow(): it makes a query and collects the result into a slice. See example below.
* `ModelName.First()` — the same as Select() but return only one record as structure (not as slice of structures)
* `ModelName.Order({orderString|field, order[, field, order …]})` — creates a scope to be used by Select() or First() with predefined order
* `ModelName.Group({fieldName0[, fieldName1[, fieldName2 [...]]]})` — creates a scope to be used by Select() or First() with predefined "GROUP BY"
* `{ModelName|scope}.Limit(limit)` — sets SQL "LIMIT" parameter
* `{ModelName|scope}.Order({orderString|field, order[, field, order …]})` — sets SQL "ORDER" parameter
* `{ModelName|scope}.Group({fieldName0[, fieldName1[, fieldName2 [...]]]})` — sets SQL "GROUP" parameter
* `{ModelName|scope}.Where({whereString|object})` — adds "WHERE" condition
* `{ModelName|scope}.Log(true, author, comment)` — enable logging to another table
* `object.Delete()` — deletes and object using it's primary key
* `object.Insert()` — saves new object properties
* `object.Update()` — updates object properties using it's primary key
* `object.Save()` — saves new object if primary key is zeroed or updates object properties using it's primary key if it's not zeroed
* `ModelNameTable.CreateTableIfNotExists(db)` — create table for model `ModelName` in database `db` (of type `*reform.Db`) [works only for sqlite3, at the moment]

Also:
* you can add a magic comment `//reformOptions:imitateGorm` to act more like [gorm](https://github.com/jinzhu/gorm): automatically generate column names and use tag "gorm" instead of "reform".
* you can use nested (embedded) structures, like:

```go
//go:generate reform

//reform:t2
type T2 struct {
	Var2 int `reform:"var2"`
	Var3 T1  `reform:"var3,embedded"`
}

type T1 struct {
	Var1 int `reform:"var1"`
}
```

`t2` columns will be: `var2` and `var3__var1`.

## Quick start

`1`. Create a model

```
mkdir -p $GOPATH/test/{models,bin}
cat > $GOPATH/test/models/RawRecord.go << EOF
package models

import "time"
//go:generate reform

//reform:raw_records
type rawRecord struct {
	Id        int       \`reform:"id,pk"\`
	CreatedAt time.Time \`reform:"created_at"\`
	SensorId  int       \`reform:"sensor_id"\`
	ChannelId int       \`reform:"channel_id"\`
	RawValue  int       \`reform:"raw_value"\`
}
EOF
```

* Magic comment `//go:generate` forces command `go generate` to run `reform`.
* Magic comment `//reform:raw_records` forces to use table `raw_records` for the model.

`2`. Create a controller for this model

```
cat > $GOPATH/test/bin/bin.go << EOF
package main

import (
        "log"

        "fmt"

        "database/sql"
        "github.com/xaionaro/reform"
        "github.com/xaionaro/reform/dialects/sqlite3"

        "../models"
)

func main() {
        simpleDB, err := sql.Open("sqlite3", "../db")
        if err != nil {
                panic(fmt.Errorf("Cannot connect to DB: %s", err.Error()))
        }

        logger := log.New(os.Stderr, "SQL: ", log.Flags())

        db := reform.NewDB(simpleDB, sqlite3.Dialect, reform.NewPrintfLogger(logger.Printf))

        models.RawRecord.SetDefaultDB(db)

        rawRecords,err := models.RawRecord.Select()
        if err != nil {
                panic(fmt.Errorf("ORM error: %s", err.Error()))
        }

        fmt.Printf("records: %v\n", rawRecords)

        return
}
EOF
```

`3`. Create the table

```
cd $GOPATH/test
sqlite3 db 'CREATE TABLE raw_records (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, sensor_id SMALLINT DEFAULT -1, channel_id SMALLINT DEFAULT -1, raw_value SMALLINT DEFAULT -1); INSERT INTO raw_records (raw_value) VALUES (1)'
```


`4`. Download dependencies

```
cd bin
go get
go get github.com/xaionaro/reform/reform
```

`5`. Generate ORM-related code by `reform`:

```
reform $GOPATH/test/models
```

`6`. Run

```
go run bin.go
```

Expected result is:
```
$ go run bin.go
SQL: 2016/06/27 14:11:52 >>> SELECT "raw_records"."id", "raw_records"."created_at", "raw_records"."sensor_id", "raw_records"."channel_id", "raw_records"."raw_value" FROM "raw_records"
SQL: 2016/06/27 14:11:52 <<< SELECT "raw_records"."id", "raw_records"."created_at", "raw_records"."sensor_id", "raw_records"."channel_id", "raw_records"."raw_value" FROM "raw_records"  622.785µs
records: [Id: 1 (int), CreatedAt: 2016-06-27 11:11:34 +0000 UTC (time.Time), SensorId: -1 (int), ChannelId: -1 (int), RawValue: 1 (int)]
```

## More examples

See more examples here:

* [https://devel.mephi.ru/dyokunev/dc-thermal-logger/src/master/server/httpsite/app/init.go](https://devel.mephi.ru/dyokunev/dc-thermal-logger/src/master/server/httpsite/app/init.go)
* [https://devel.mephi.ru/dyokunev/dc-thermal-logger/src/master/server/httpsite/app/controllers/Dashboard.go](https://devel.mephi.ru/dyokunev/dc-thermal-logger/src/master/server/httpsite/app/controllers/Dashboard.go)

## Troubleshooting

1. Select() returns empty slice while it shouldn't. Probably there's a conversion problem. For example at the moment this ORM (at least this fork) doesn't work with NULL-valued "int"-s. You may try to print err.Error() before "break" line of Select() function in your `……_reform.go` file.
