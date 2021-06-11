package geeorm

import (
	"database/sql"
	"fmt"
	"geeorm/geeorm/dialect"
	"geeorm/geeorm/log"
	"strings"

	"geeorm/geeorm/sessions"
)

type Engine struct {
	db *sql.DB
	dialect dialect.Dialect
}

func NewEngine(driver, source string) (engine *Engine,err error) {
	db, err := sql.Open(driver, source)
	if err != nil {
		log.Error(err)
		return 
	}


	if err = db.Ping(); err != nil {
		log.Error(err)
		return 
	}

	dial, ok := dialect.GetDialect(driver)
	if !ok {
		log.Error("Dialect %s Not Found", driver)
		return 
	}

	engine = &Engine{
		db:db,
		dialect:dial,
	}


	log.Info("Connect database sucessfully!")

	return 
}


func (engine *Engine)Close() {
	if err := engine.db.Close(); err != nil {
		log.Error("Failed to close database")
		return
	}
	log.Info("Database close successfully")
}

func (engine *Engine)NewSession() *sessions.Session {
	return sessions.New(engine.db, engine.dialect)
}

type TxFunc func(session *sessions.Session) (interface{}, error)

func (engine *Engine) Transaction(f TxFunc) (interface{}, error) {
	s := engine.NewSession()
	var err error
	if err = s.Begin(); err != nil {
		return nil, err
	}

	defer func() {
		if p:= recover();p!=nil {
			_ = s.Rollback()
			panic(p)
		} else if err != nil {
			_ = s.Rollback()
		} else {
			err = s.Commit()
		}
	}()

	return f(s)
}

func difference(a, b []string) (diff []string) {
	mapB := make(map[string]bool)

	for _,v := range b {
		mapB[v] = true
	}

	for _,v := range a {
		if _, ok := mapB[v]; !ok {
			diff = append(diff,v)
		}
	}
	return 
}

func (engine *Engine) Migrate(value interface{}) error {
	_, err := engine.Transaction(
		func (s *sessions.Session) (result interface{}, err error) {
			if !s.Model(value).HasTable() {
				log.Info("table %s does not exist", s.RefTable().Name)
				return nil, s.CreateTable()
			}

			table := s.RefTable()
			rows, _ := s.Raw(fmt.Sprintf("SELECT * FORM %s LIMIT 1", table.Name)).QueryRows()
			columns, _ := rows.Columns()
			addCols := difference(table.FieldName,columns)
			delCols := difference(columns, table.FieldName)

			log.Info("add cols %v, del cols %v", addCols, delCols)
			for _, col := range addCols {
				f := table.GetField(col)
				sqlStr := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table.Name, f.Name, f.Type)
				if _, err = s.Raw(sqlStr).Exec(); err != nil {
					return 
				}
			}

			if len(delCols) == 0 {
				return 
			}

			tmp := "tmp_" + table.Name
			fieldStr := strings.Join(table.FieldName, ", ")
			s.Raw(fmt.Sprintf("CREATE TABLE %s AS SELECT %s from %s;", tmp, fieldStr, table.Name))
			s.Raw(fmt.Sprintf("DROP TABLE %s;", table.Name))
			s.Raw(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", tmp, table.Name))

			_, err = s.Exec()
			return 
		})



	return err
}
