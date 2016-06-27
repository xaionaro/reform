# a derivative of reform

This's a derivative of a great project "[reform](https://github.com/go-reform/reform)". Please look at the upstream code first.

This project just adds some functionality to the upstream ORM.

## Quick start

1. Create a model

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

2. Generate ORM-related code by `reform`:

```
reform $GOPATH/test/models
```

3. Create a controller for this model

```
cat > $GOPATH/test/bin/bin.go << EOF
package main

import (
        "log"

        "fmt"

        "database/sql"
        "gopkg.in/reform.v1"
        "gopkg.in/reform.v1/dialects/sqlite3"

        "../models"
)

func main() {
        simpleDB, err := sql.Open("sqlite3", "../db")
        if err != nil {
                panic(fmt.Errorf("Cannot connect to DB: %s", err.Error()))
        }

        logger := log.New(os.Stderr, "SQL: ", log.Flags())

        db := reform.NewDB(simpleDB, sqlite3.Dialect, reform.NewPrintfLogger(logger.Printf))

        rawRecords,err := models.RawRecord.Select(db)
        if err != nil {
                panic(fmt.Errorf("ORM error: %s", err.Error()))
        }

        fmt.Printf("records: %v\n", rawRecords)

        return
}
EOF
```

4. Create the table

```
cd $GOPATH/test
sqlite3 db 'CREATE TABLE raw_records (id INTEGER PRIMARY KEY AUTOINCREMENT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, sensor_id SMALLINT DEFAULT -1, channel_id SMALLINT DEFAULT -1, raw_value SMALLINT DEFAULT -1); INSERT INTO raw_records (raw_value) VALUES (1)'
```


5. Download dependencies

```
cd bin
go get
```

6. Run

```
go run bin.go
```

## Troubleshooting

1. Select() returns empty slice while it shouldn't. Probably there's a conversion problem. For example at the moment this ORM (at least this fork) doesn't work with NULL-valued "int"-s. You may try to print err.Error() before "break" line of Select() function in your `……_reform.go` file.
