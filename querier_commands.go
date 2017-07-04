package reform

import (
	"fmt"
	"strings"
)

func filteredColumnsAndValues(str Struct, columnsIn []string, isUpdate bool) (columns []string, values []interface{}, err error) {
	columnsSet := make(map[string]struct{}, len(columnsIn))
	for _, c := range columnsIn {
		columnsSet[c] = struct{}{}
	}

	// select columns from set and collect values
	view := str.View()
	allColumns := view.Columns()
	allValues := str.Values()
	columns = make([]string, 0, len(columnsSet))
	values = make([]interface{}, 0, len(columns))

	record, _ := str.(Record)
	var pk uint
	if record != nil {
		pk = view.(Table).PKColumnIndex()
	}

	for i, c := range allColumns {
		if _, ok := columnsSet[c]; ok {
			if isUpdate && record != nil && i == int(pk) {
				err = fmt.Errorf("reform: will not update PK column: %s", c)
				return
			}
			delete(columnsSet, c)
			columns = append(columns, c)
			values = append(values, allValues[i])
		}
	}

	// make error for extra columns
	if len(columnsSet) > 0 {
		columns = make([]string, 0, len(columnsSet))
		for c := range columnsSet {
			columns = append(columns, c)
		}
		// TODO make exported type for that error
		err = fmt.Errorf("reform: unexpected columns: %v", columns)
		return
	}

	return
}

func (q *Querier) insert(str Struct, columns []string, values []interface{}) error {
	for i, c := range columns {
		columns[i] = q.QuoteIdentifier(c)
	}
	placeholders := q.Placeholders(1, len(columns))

	view := str.View()
	record, _ := str.(Record)
	lastInsertIdMethod := q.LastInsertIdMethod()
	defaultValuesMethod := q.DefaultValuesMethod()

	var pk uint
	if record != nil {
		pk = view.(Table).PKColumnIndex()
	}

	// make query
	query := q.startQuery("INSERT") + " INTO " + q.QualifiedView(view)
	if len(columns) != 0 || defaultValuesMethod == EmptyLists {
		query += " (" + strings.Join(columns, ", ") + ")"
	}
	if record != nil && lastInsertIdMethod == OutputInserted {
		query += fmt.Sprintf(" OUTPUT INSERTED.%s", q.QuoteIdentifier(view.Columns()[pk]))
	}
	if len(placeholders) != 0 || defaultValuesMethod == EmptyLists {
		query += fmt.Sprintf(" VALUES (%s)", strings.Join(placeholders, ", "))
	} else {
		query += " DEFAULT VALUES"
	}
	if record != nil && lastInsertIdMethod == Returning {
		query += fmt.Sprintf(" RETURNING %s", q.QuoteIdentifier(view.Columns()[pk]))
	}

	switch lastInsertIdMethod {
	case LastInsertId:
		res, err := q.Exec(query, values...)
		if err != nil {
			return err
		}
		if record != nil && !record.HasPK() {
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}
			record.SetPK(id)
		}
		return nil

	case Returning, OutputInserted:
		var err error
		if record != nil {
			err = q.QueryRow(query, values...).Scan(record.PKPointer())
		} else {
			_, err = q.Exec(query, values...)
		}
		return err

	default:
		panic("reform: Unhandled LastInsertIdMethod. Please report this bug.")
	}
}

func (q *Querier) beforeInsert(str Struct) error {
	return q.callStructMethod(str, "BeforeInsert")
}

func (q *Querier) afterInsert(str Struct) error {
	return q.callStructMethod(str, "AfterInsert")
}

// Insert inserts a struct into SQL database table.
// If str has valid method "BeforeInsert", it calls BeforeInsert() before doing so.
// If str has valid method "AfterInsert", it calls AfterInsert() after doing so.
//
// It fills record's primary key field.
func (q *Querier) Insert(str Struct) error {
	if err := q.beforeInsert(str); err != nil {
		return err
	}

	view := str.View()
	values := str.Values()
	columns := view.Columns()
	record, _ := str.(Record)

	if record != nil {
		pk := view.(Table).PKColumnIndex()

		// cut primary key
		if !record.HasPK() {
			values = append(values[:pk], values[pk+1:]...)
			columns = append(columns[:pk], columns[pk+1:]...)
		}
	}

	err := q.insert(str, columns, values)

	if err == nil {
		return q.afterInsert(str)
	}

	return err
}

// InsertColumns inserts a struct into SQL database table with specified columns.
// Other columns are omitted from generated INSERT statement.
// If str has valid method "BeforeInsert", it calls BeforeInsert() before doing so.
// If str has valid method "AfterInsert", it calls AfterInsert() after doing so.
//
// It fills record's primary key field.
func (q *Querier) InsertColumns(str Struct, columns ...string) error {
	if err := q.beforeInsert(str); err != nil {
		return err
	}

	columns, values, err := filteredColumnsAndValues(str, columns, false)
	if err != nil {
		return err
	}

	err = q.insert(str, columns, values)

	if err == nil {
		return q.afterInsert(str)
	}

	return err
}

// InsertMulti inserts several structs into SQL database table with single query.
// If they has valid method "BeforeInsert", it calls BeforeInsert() before doing so.
//
// All structs should belong to the same view/table.
// All records should either have or not have primary key set.
// It doesn't fill primary key fields.
// Given all these limitations, most users should use Querier.Insert in a loop, not this method.
func (q *Querier) InsertMulti(structs ...Struct) error {
	if len(structs) == 0 {
		return nil
	}

	// check that view is the same
	view := structs[0].View()
	for _, str := range structs {
		if str.View() != view {
			return fmt.Errorf("reform: different tables in InsertMulti: %s and %s", view.Name(), str.View().Name())
		}
	}

	for _, str := range structs {
		err := q.callStructMethod(str, "BeforeInsert")
		if err != nil {
			return err
		}
	}

	// check if all PK are present or all are absent
	record, _ := structs[0].(Record)
	if record != nil {
		for _, str := range structs {
			rec, _ := str.(Record)
			if record.HasPK() != rec.HasPK() {
				return fmt.Errorf("reform: PK in present in one struct and absent in other: first: %s, second: %s",
					record, rec)
			}
		}
	}

	columns := view.Columns()
	for i, c := range columns {
		columns[i] = q.QuoteIdentifier(c)
	}

	var pk uint
	if record != nil && !record.HasPK() {
		pk = view.(Table).PKColumnIndex()
		columns = append(columns[:pk], columns[pk+1:]...)
	}

	placeholders := q.Placeholders(1, len(columns)*len(structs))
	query := fmt.Sprintf("%s INTO %s (%s) VALUES ",
		q.startQuery("INSERT"),
		q.QualifiedView(view),
		strings.Join(columns, ", "),
	)
	for i := 0; i < len(structs); i++ {
		query += fmt.Sprintf("(%s), ", strings.Join(placeholders[len(columns)*i:len(columns)*(i+1)], ", "))
	}
	query = query[:len(query)-2] // cut last ", "

	values := make([]interface{}, 0, len(placeholders))
	for _, str := range structs {
		v := str.Values()
		if record != nil && !record.HasPK() {
			v = append(v[:pk], v[pk+1:]...)
		}
		values = append(values, v...)
	}

	_, err := q.Exec(query, values...)
	return err
}

func (q *Querier) update(record Record, columns []string, values []interface{}) error {
	for i, c := range columns {
		columns[i] = q.QuoteIdentifier(c)
	}
	placeholders := q.Placeholders(1, len(columns))

	p := make([]string, len(columns))
	for i, c := range columns {
		p[i] = c + " = " + placeholders[i]
	}
	table := record.Table()
	query := fmt.Sprintf("%s %s SET %s WHERE %s = %s",
		q.startQuery("UPDATE"),
		q.QualifiedView(table),
		strings.Join(p, ", "),
		q.QuoteIdentifier(table.Columns()[table.PKColumnIndex()]),
		q.Placeholder(len(columns)+1),
	)

	args := append(values, record.PKValue())
	res, err := q.Exec(query, args...)
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra == 0 {
		return ErrNoRows
	}
	if ra > 1 {
		panic(fmt.Sprintf("reform: %d rows by UPDATE by primary key. Please report this bug.", ra))
	}
	return nil
}

func (q *Querier) beforeUpdate(record Record) error {
	if !record.HasPK() {
		return ErrNoPK
	}

	return q.callStructMethod(record, "BeforeUpdate")
}

func (q *Querier) afterUpdate(record Record) error {
	return q.callStructMethod(record, "AfterUpdate")
}

// Update updates all columns of row specified by primary key in SQL database table with given record.
// If record has valid method "BeforeUpdate", it calls BeforeUpdate() before doing so.
// If record has valid method "AfterUpdate", it calls AfterUpdate() before doing so.
//
// Method returns ErrNoRows if no rows were updated.
// Method returns ErrNoPK if primary key is not set.
func (q *Querier) Update(record Record) error {
	if err := q.beforeUpdate(record); err != nil {
		return err
	}

	table := record.Table()
	values := record.Values()
	columns := table.Columns()

	// cut primary key
	pk := table.PKColumnIndex()
	values = append(values[:pk], values[pk+1:]...)
	columns = append(columns[:pk], columns[pk+1:]...)

	err := q.update(record, columns, values)

	if err == nil {
		return q.afterUpdate(record)
	}
	return err
}

// UpdateColumns updates specified columns of row specified by primary key in SQL database table with given record.
// Other columns are omitted from generated UPDATE statement.
// If record has valid method "BeforeUpdate", it calls BeforeUpdate() before doing so.
// If record has valid method "AfterUpdate", it calls AfterUpdate() before doing so.
//
// Method returns ErrNoRows if no rows were updated.
// Method returns ErrNoPK if primary key is not set.
func (q *Querier) UpdateColumns(record Record, columns ...string) error {
	if err := q.beforeUpdate(record); err != nil {
		return err
	}

	columns, values, err := filteredColumnsAndValues(record, columns, true)
	if err != nil {
		return err
	}

	if len(values) == 0 {
		// TODO make exported type for that error
		return fmt.Errorf("reform: nothing to update")
	}

	err = q.update(record, columns, values)

	if err == nil {
		return q.afterUpdate(record)
	}
	return err
}

// Save saves record in SQL database table.
// If primary key is set, it first calls Update and checks if row was updated.
// If primary key is absent or no row was updated, it calls Insert.
func (q *Querier) Save(record Record) error {
	if record.HasPK() {
		err := q.Update(record)
		if err != ErrNoRows {
			return err
		}
	}

	return q.Insert(record)
}

func (q *Querier) beforeDelete(record Record) error {
	if !record.HasPK() {
		return ErrNoPK
	}

	return q.callStructMethod(record, "BeforeDelete")
}

func (q *Querier) afterDelete(record Record) error {
	return q.callStructMethod(record, "AfterDelete")
}

// Delete deletes record from SQL database table by primary key.
// If record has valid method "BeforeDelete", it calls BeforeDelete() before doing so.
// If record has valid method "AfterDelete", it calls AfterDelete() before doing so.
//
// Method returns ErrNoRows if no rows were deleted.
// Method returns ErrNoPK if primary key is not set.
func (q *Querier) Delete(record Record) error {
	err := q.beforeDelete(record)
	if err != nil {
		return err
	}

	table := record.Table()
	pk := table.PKColumnIndex()
	query := fmt.Sprintf("%s FROM %s WHERE %s = %s",
		q.startQuery("DELETE"),
		q.QualifiedView(table),
		q.QuoteIdentifier(table.Columns()[pk]),
		q.Placeholder(1),
	)

	res, err := q.Exec(query, record.PKValue())
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra == 0 {
		return ErrNoRows
	}
	if ra > 1 {
		panic(fmt.Sprintf("reform: %d rows by DELETE by primary key. Please report this bug.", ra))
	}
	return q.afterDelete(record)
}

// DeleteFrom deletes rows from view with tail and args and returns a number of deleted rows.
//
// Method never returns ErrNoRows.
func (q *Querier) DeleteFrom(view View, tail string, args ...interface{}) (uint, error) {
	query := fmt.Sprintf("%s FROM %s %s",
		q.startQuery("DELETE"),
		q.QualifiedView(view),
		tail,
	)

	res, err := q.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return uint(ra), nil
}
