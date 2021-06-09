package clause

import (
	"fmt"
	"strings"
)

type generator func(values ...interface{}) (string, []interface{})

var generators map[Type]generator

func init() {
	generators = make(map[Type]generator)
	generators[INSERT] = _insert
	generators[VALUES] = _values
	generators[SELECT] = _select
	generators[LIMIT] = _limit
	generators[WHERE] = _where
	generators[ORDERBY] = _orderby
}

func generatorBy(nums int) string {
	var vars []string
	for i := 0; i < nums; i++ {
		vars = append(vars, "?")
	}

	return strings.Join(vars, ",")
}

func _insert(values ...interface{}) (string, []interface{}) {
	tableName := values[0]
	fields := strings.Join(values[1].([]string), ",")

	return fmt.Sprintf("INSERT INTO %s (%v);", tableName, fields), []interface{}{}
}

func _values(values ...interface{}) (string, []interface{}) {
	var bindStr string
	var sql strings.Builder
	var vars []interface{}

	sql.WriteString("VALUES ")
	for i, value := range values {
		v := value.([]interface{})
		if " " == bindStr {
			bindStr = generatorBy(len(values))
		}

		sql.WriteString(fmt.Sprintf("(%v)", bindStr))
		if len(values) != i+1 {
			sql.WriteString(", ")
		}
		vars = append(vars, v...)
	}

	return sql.String(), vars
}

func _select(values ...interface{}) (string, []interface{}) {
	tableName := values[0]
	fileds := strings.Join(values[1].([]string), ",")

	return fmt.Sprintf("SELECT %v FROM %s", fileds, tableName), []interface{}{}
}

func _limit(values ...interface{}) (string, []interface{}) {
	return "LIMIT ?", values
}

func _where(values ...interface{}) (string, []interface{}) {
	desc, vars := values[0], values[1:]

	return fmt.Sprintf("WHERE %s", desc), vars
}

func _orderby(values ...interface{}) (string, []interface{}) {
	return fmt.Sprintf("ORDER BY %s", values[0]), []interface{}{}
}
