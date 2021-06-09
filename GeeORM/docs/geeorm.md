## `GEEORM`
### `浅谈ORM`
对象关系映射(`Object Relational Mapping`，简称`ORM`),通过描述对象和数据库之间的映射数据，将面向对象语言程序中的对象自动持久到关系数据库中。
对象与数据库的映射关系：

|        数据库        |         对象          |
| :------------------: | :-------------------: |
|     表(`table`)      |  类(`class/struct`)   |
|  记录(`record/row`)  |    对象(`object`)     |
| 字段(`field/column`) | 对象属性(`attribute`) |

 `log`框架实现：
```go
package log

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// log.Lshortfile 支持显示文件名和行号
var (
	errLog = log.New(os.Stdout,"\033[31m[error]\033[0m", log.LstdFlags|log.Lshortfile)
	infoLog = log.New(os.Stdout,"\033[34m[info]\033[0m", log.LstdFlags|log.Lshortfile)
	loggers = []*log.Logger{errLog, infoLog}

	mu sync.Mutex
)

var (
	Error = errLog.Println
	Errorf = errLog.Printf
	Info = infoLog.Println
	Infof = infoLog.Printf
)

// log levels
const (
	InfoLevel = iota
	ErrorLevel
	Distabled
)

func setLevel(level int) {
	mu.Lock()
	defer mu.Unlock()

	for _,logger := range loggers {
		logger.SetOutput(os.Stdout)
	}

	if ErrorLevel < level {
		errLog.SetOutput(ioutil.Discard)
	}

	if InfoLevel < level {
		infoLog.SetOutput(ioutil.Discard)
	}

}
```
`engine`实现
```go
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
```
`session`实现
```go
package sessions

import (
	"database/sql"
	"geeorm/geeorm/log"
	"strings"
)

type Session struct {
	db *sql.DB
	sql strings.Builder
	sqlVars []interface{}
}

func New(db *sql.DB) *Session {
	return &Session{db: db,}
}

func (s *Session)Clear() {
	s.sql.Reset()
	s.sqlVars = nil
}

func (s *Session)DB() *sql.DB {
	return s.db
}

func (s *Session)Raw(sql string, values ...interface{}) *Session {
	s.sql.WriteString(sql)
	s.sql.WriteString(" ")
	s.sqlVars = append(s.sqlVars, values...)
	return s
}

func (s *Session)Exec()(result sql.Result, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if result , err = s.DB().Exec(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}

	return
}

func (s *Session)QueryRow() *sql.Row {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	return s.DB().QueryRow(s.sql.String(), s.sqlVars...)
}

func (s *Session)QueryRows() (rows *sql.Rows, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if rows, err = s.DB().Query(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return 
}
```
### 对象表结构映射
实现`ORM`的第一步，将`GO`的数据类型映射为数据库的数据类型。
本文将数据库差异部分提取出来，抽象为`dialect`，方便代码的复用。
```go
package dialect

import "reflect"

var dialectMap = map[string]Dialect{}

type Dialect interface {
	DataTypeOf(typ reflect.Value) string
	TableExistsSQL(tableName string) (string, []interface{})
}

func RegisterDialect(name string, dialect Dialect) {
	dialectMap[name] = dialect
}

func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectMap[name]
	return 
}
```
通过上述接口，实现如下对`sqlite3`的支持：
```go
package dialect

import (
	"fmt"
	"reflect"
	"time"
)

type sqlite3 struct {}

var _ Dialect = (*sqlite3)(nil)

func init() {
	RegisterDialect("sqlite3", &sqlite3{})
}

func (s *sqlite3)DataTypeOf(typ reflect.Value) string {
	switch typ.Kind(){
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Uint, reflect.Uint8,reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Uint64, reflect.Int64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.String:
			return "text"
	case reflect.Array, reflect.Slice:
			return "blob"
	case reflect.Struct:
		if _,ok := typ.Interface().(time.Time); !ok {
			return "datetime"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Type().Name(), typ.Kind()))
}


func (s *sqlite3)TableExistsSQL(tableName string) (string, []interface{}){
	args := []interface{}{tableName}

	return "SELECT name FROM sqlite_master WHERE type='table' and name= ?", args
}
```
### `Schema`
如何实现数据库表与`GO`数据结构的映射？
> 表名(`table name`) --- 结构体名(`struct name`)
> 字段名和字段类型 --- 成员变量和类型
> 额外的约束条件 --- 成员变量`Tag`

```go
package dialect

import "reflect"

var dialectMap = map[string]Dialect{}

type Dialect interface {
	DataTypeOf(typ reflect.Value) string
	TableExistsSQL(tableName string) (string, []interface{})
}

func RegisterDialect(name string, dialect Dialect) {
	dialectMap[name] = dialect
}

func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectMap[name]
	return 
}
```

```go
package dialect

import (
	"fmt"
	"reflect"
	"time"
)

type sqlite3 struct {}

var _ Dialect = (*sqlite3)(nil)

func init() {
	RegisterDialect("sqlite3", &sqlite3{})
}

func (s *sqlite3)DataTypeOf(typ reflect.Value) string {
	switch typ.Kind(){
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Uint, reflect.Uint8,reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Uint64, reflect.Int64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.String:
			return "text"
	case reflect.Array, reflect.Slice:
			return "blob"
	case reflect.Struct:
		if _,ok := typ.Interface().(time.Time); !ok {
			return "datetime"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Type().Name(), typ.Kind()))
}


func (s *sqlite3)TableExistsSQL(tableName string) (string, []interface{}){
	args := []interface{}{tableName}

	return "SELECT name FROM sqlite_master WHERE type='table' and name= ?", args
}
```
数据库表操作：
```go
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
```
