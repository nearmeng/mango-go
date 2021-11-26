// Package mysql db mysql support plugin.
package mysql

import (
	"bytes"
	"fmt"
	"text/template"
)

// CreateTable create table by empty record.
// warning: this function is for test.
func (t *DB) CreateTable(record Record) error {
	tpl, err := template.New("CreateTable").Parse(_CreateTableTemplate)
	if err != nil {
		return fmt.Errorf("create table fullname:%s err:%w", record.ProtoReflect().Descriptor().FullName(), err)
	}
	ti, err := newTableInfo(record.ProtoReflect().Descriptor())
	if err != nil {
		return err
	}
	b := &bytes.Buffer{}
	err = tpl.Execute(b, ti)
	if err != nil {
		return fmt.Errorf("create table sql fullname:%s err:%w", record.ProtoReflect().Descriptor().FullName(), err)
	}
	sqlText := b.String()
	_, err = t.sql.ExecContext(t.ctx, sqlText)
	return err
}

// DropTable drop table by empty record.
func (t *DB) DropTable(record Record) error {
	sqlText := fmt.Sprintf("drop table if exists %s", record.ProtoReflect().Descriptor().Name())
	if _, err := t.sql.Exec(sqlText); err != nil {
		return err
	}
	return nil
}

// ExistTable test if table exist.
func (t *DB) ExistTable(record Record) bool {
	sqlText := fmt.Sprintf("desc %s", record.ProtoReflect().Descriptor().Name())
	if _, err := t.sql.ExecContext(t.ctx, sqlText); err != nil {
		return false
	}
	return true
}
