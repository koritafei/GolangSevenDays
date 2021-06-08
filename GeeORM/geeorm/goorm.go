package geeorm

import (
	"database/sql"
	"geeorm/geeorm/log"

	"geeorm/geeorm/sessions"
)

type Engine struct {
	db *sql.DB
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

	engine = &Engine{
		db:db,
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
	return sessions.New(engine.db)
}