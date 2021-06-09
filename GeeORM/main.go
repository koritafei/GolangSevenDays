package main

import (
	"fmt"
	"geeorm/geeorm"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	engine,_ := geeorm.NewEngine("sqlite3", "gee.db")
	defer engine.Close()
	s := engine.NewSession()
	s.Raw("DROP TABLE IF EXISTS User;").Exec()
	s.Raw("CREATE TABLE User (Name text);").Exec()
  s.Raw("CREATE TABLE User (Name text);").Exec()

	result , _ := s.Raw("INSERT INTO User(`Name`) Values(?),(?)", "Tom", "Jack").Exec()
	count , _ := result.RowsAffected()
	
	fmt.Printf("result %v, count %v\n", result, count)
	results, _ := s.Raw("SELECT * FROM User;").Exec()
	fmt.Printf("results %v\n", results)
}

