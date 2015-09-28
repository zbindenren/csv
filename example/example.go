package main

import (
	"fmt"
	"strings"

	"github.com/zbindenren/csv"
)

type TestStruct struct {
	Field0         string  `csv:"FIELD_0"`
	Field1         int     `csv:"FIELD_1"`
	Field2         bool    `csv:"FIELD_2"`
	Field3         float64 `csv:"FIELD_3"`
	IngnoredStruct bool    `csv:"-"`
}

func main() {

	data := `FIELD_0;FIELD_1;FIELD_2;FIELD_3
string1;1;true;1.14
string2;2;true;2.14
string3;3;true;3.14`

	r := strings.NewReader(data)
	m, err := csv.NewMarshaler(TestStruct{}, r)
	if err != nil {
		panic(err)
	}
	m.Reader.Comma = ';'
	result, err := m.Unmarshal()
	if err != nil {
		panic(err)
	}
	for _, item := range result {
		if t, ok := item.(TestStruct); ok {
			fmt.Println(t.Field0)
		}
	}
}
