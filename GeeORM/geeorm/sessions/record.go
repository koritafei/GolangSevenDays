package sessions

import (
	"errors"
	"geeorm/geeorm/clause"
	"reflect"
)

func (s *Session) Insert (values ...interface{}) (int64, error) {
	s.CallMethod(BeforeInsert, nil)
	recordValues := make([]interface{}, 0)

	for _, value := range values {
		table := s.Model(value).RefTable()
		s.clause.Set(clause.INSERT, table.Name, table.FieldName)
		recordValues = append(recordValues, table.RecordValues(value))
	}

	s.clause.Set(clause.VALUES, recordValues...)
	sql, vars := s.clause.Build(clause.INSERT, clause.VALUES)
	result, err := s.Raw(sql, vars...).Exec()

	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterQuery, values)
	return result.RowsAffected()
}


func (s *Session) Find(values interface{}) error {
	s.CallMethod(BeforeQuery, nil)
	destSlice := reflect.Indirect(reflect.ValueOf(values))
	destType := destSlice.Type().Elem()
	table := s.Model(reflect.New(destType).Elem().Interface()).RefTable()

	s.clause.Set(clause.SELECT, table.Name, table.FieldName)
	sql, vars := s.clause.Build(clause.SELECT, clause.WHERE, clause.ORDERBY, clause.LIMIT)
	row, err := s.Raw(sql, vars...).QueryRows()
	if err != nil {
		return err
	}

	for row.Next() {
		dest := reflect.New(destType).Elem()
		var values []interface{}
		for _, name := range table.FieldName{
			values = append(values, dest.FieldByName(name).Addr().Interface())
		}

		if err := row.Scan(values...); err != nil {
			return err
		}
		destSlice.Set(reflect.Append(destSlice, dest))
		s.CallMethod(AfterQuery, dest.Addr().Interface())
	}

	return row.Close()
}

func (s *Session)Update(kv ...interface{}) (int64, error) {
	s.CallMethod(BeforeUpdate, nil)
	m,ok := kv[0].(map[string]interface{})
	if !ok {
		m = make(map[string]interface{})
		for i:=0;i<len(kv);i++{
			m[kv[i].(string)] = kv[i+1]
		}
	}

	s.clause.Set(clause.UPDATE, s.RefTable().Name, m)
	sql, vars := s.clause.Build(clause.UPDATE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterUpdate, kv)
	return result.RowsAffected()
}

func (s *Session)Delete() (int64, error) {
	s.CallMethod(BeforeDelete, nil)
	s.clause.Set(clause.DELETE, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.DELETE, clause.WHERE)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	s.CallMethod(AfterDelete, nil)
	return result.RowsAffected()
}

func (s *Session)Count() (int64, error) {
	s.clause.Set(clause.COUNT, s.RefTable().Name)
	sql, vars := s.clause.Build(clause.COUNT, clause.WHERE)
	row := s.Raw(sql, vars...).QueryRow()

	var tmp int64
	if err := row.Scan(&tmp); err != nil {
		return 0, err
	}

	return tmp, nil
}

func (s *Session)Limit(num int) *Session {
	s.clause.Set(clause.LIMIT, num)
	return s
}

func (s *Session)Where(desc string, args...interface{}) *Session {
	var vars []interface{}
	s.clause.Set(clause.WHERE, append(append(vars, desc), args...)...)
	return s
}

func (s *Session)OrderBy(desc string) *Session {
	s.clause.Set(clause.ORDERBY, desc)

	return s
}

func (s *Session)First(value interface{}) error {
	dest := reflect.Indirect(reflect.ValueOf(value))
	destSlice := reflect.New(reflect.SliceOf(dest.Type())).Elem()

	if err := s.Limit(1).Find(destSlice.Addr().Interface()); err != nil {
		return err
	}

	if destSlice.Len() == 0 {
		return errors.New("NOT FOUND")
	}

	dest.Set(destSlice.Index(0))

	return nil
}