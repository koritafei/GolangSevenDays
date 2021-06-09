package Schema

import (
	"geeorm/geeorm/dialect"
	"testing"
)

type User struct {
	Name string `geeorm:"PRIMARY KEY"`
	Age int
}

var TestDial,_ = dialect.GetDialect("sqlite3")

func TestParse(t *testing.T) {
	schema := Parse(&User{}, TestDial)

	if "User" != schema.Name || 2 != len(schema.Fields){
		t.Fatal("failed to parse User struct")
	}

	if "PRIMARY KEY" != schema.GetField("Name").Tag {
		t.Fatal("failed to parse primary key")
	}
}
