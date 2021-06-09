package sessions

import (
	"fmt"
	"geeorm/geeorm/Schema"
	"geeorm/geeorm/log"
	"reflect"
	"strings"
)

// 封装数据库表操作

func (s *Session) Model(value interface{}) *Session{
	if s.refTable == nil || reflect.TypeOf(value) != reflect.TypeOf(s.refTable.Model) {
		s.refTable = Schema.Parse(value, s.dialect)
	}

	return s
}

func (s *Session) RefTable() *Schema.Schema {
	if s.refTable == nil {
		log.Error("Model is not set")
	}

	return s.refTable
}

func (s *Session) CreateTable() error {
	table := s.refTable
	var cloumns []string
	for _, field := range table.Fields {
		cloumns = append(cloumns, fmt.Sprintf("%s %s %s", field.Name, field.Type, field.Tag))
	}

	desc := strings.Join(cloumns, ",")
	_, err := s.Raw(fmt.Sprintf("CREATE TABLE %s (%s);", table.Name, desc)).Exec()

	return err
}

func (s *Session) DropTable() error {
	_, err := s.Raw(fmt.Sprintf("DROP TABLE IF EXISTS %s;", s.RefTable().Name)).Exec()

	return err
}

func (s *Session) HasTable() bool {
	sql, value := s.dialect.TableExistsSQL(s.RefTable().Name)
	row := s.Raw(sql, value...).QueryRow()

	var tmp string
	_ = row.Scan(&tmp)

	return tmp == s.RefTable().Name
}