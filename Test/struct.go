package main

import (
	"fmt"
	"regexp"
)

type TestA struct {
	name    string
	address string
}

type TestB struct {
	*TestA
	age int
}

func main() {
	a := &TestA{
		name:    "testa",
		address: "adsfda",
	}
	b := &TestB{
		age:   10,
		TestA: a,
	}

	b.TestA = &TestA{
		name:    "asdas",
		address: "asdasdasdasda",
	}

	fmt.Printf("%v, name:%s, address: %s, age %d\n", b, b.name, b.address, b.age)

	url := "http://fadf.com?__asda__&sda=__aaa__"
	fmt.Println(url)

	re1, _ := regexp.Compile("\\_\\_\\w+\\_\\_")
	url = re1.ReplaceAllString(url, "")
	fmt.Println(url)
}
