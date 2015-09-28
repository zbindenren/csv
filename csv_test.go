package csv

import (
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
)

type TestStruct struct {
	Field0         string  `csv:"FIELD_0"`
	Field1         int     `csv:"FIELD_1"`
	Field2         bool    `csv:"FIELD_2"`
	Field3         float64 `csv:"FIELD_3"`
	IngnoredStruct bool    `csv:"-"`
}

var (
	dataWithJunk = `Junk Line
Again Junk
Again;Junk
FIELD_0;FIELD_1;FIELD_2;FIELD_3
string1;1;true;1.14
junk; junk
#asdf
And junk;
string2;2;true;2.14
string3;3;true;3.14`

	wrongTypes = `FIELD_0;FIELD_1;FIELD_2;FIELD_3
string1;1;true;1.14
string2;2;notvalid;2.14
string3;3;true;3.14`

	notEnoughFields = `FIELD_0;FIELD_2;FIELD_1;FIELD_3
string1;true;1
string2;true;2;1.14
string3;true;3`

	tooManyFields = `FIELD_0;FIELD_1;FIELD_2;FIELD_3
string1;1;true;1.14;to much
string2;2;false;2.14
string3;3;true;3.14;really to much`

	notEnoughHeaders = `FIELD_0;FIELD_2;FIELD_1
string1;true;1;1.14
string2;true;2;2.14
string3;true;3;3.14`

	noValidRecord = `FIELD_0;FIELD_2;FIELD_1
string1;true;1
string2;true;2
string3;true;3`

	firstLine = TestStruct{
		Field0: "string1",
		Field1: 1,
		Field2: true,
		Field3: 1.14,
	}
)

func TestCsvHeadersValidStructs(t *testing.T) {
	good := TestStruct{}
	correctFieldInfos := fieldInfos{
		fieldInfo{
			position:   -1,
			headerName: "FIELD_0",
			fieldName:  "Field0",
			kind:       reflect.String,
		},
		fieldInfo{
			position:   -1,
			headerName: "FIELD_1",
			fieldName:  "Field1",
			kind:       reflect.Int,
		},
		fieldInfo{
			position:   -1,
			headerName: "FIELD_2",
			fieldName:  "Field2",
			kind:       reflect.Bool,
		},
		fieldInfo{
			position:   -1,
			headerName: "FIELD_3",
			fieldName:  "Field3",
			kind:       reflect.Float64,
		},
	}
	generatedFieldInfos, err := createFieldInfos(good)
	if err != nil {
		t.Fatalf("error occured in TestCsvHeaders: %s", err)
	}
	if len(generatedFieldInfos) != 4 {
		t.Errorf("wrong number of haeders generated: %v", generatedFieldInfos)
	}

	for i, fi := range correctFieldInfos {
		if !reflect.DeepEqual(generatedFieldInfos[i], fi) {
			t.Errorf("wrong haeders generated - want: %v, got: %v", fi, generatedFieldInfos[i])
		}
	}

}

func TestCsvHeadersInvalidStructs(t *testing.T) {
	type InvalidStruct1 struct {
		Field0 string `csv:"FIELD_0"`
		Field1 int    `csv:"FIELD_0"` //duplicate tag definition
	}

	type InvalidStruct2 struct {
		Field0 string `csv` // empty csv tag
		Field1 int    `csv:"FIELD_1"`
	}

	type InvalidStruct3 struct {
		Field0 string `csv:` // empty csv tag
		Field1 int    `csv:"FIELD_1"`
	}
	noStruct := "string"
	invalidStructs := []interface{}{InvalidStruct1{}, InvalidStruct2{}, InvalidStruct3{}, noStruct}
	for _, invalid := range invalidStructs {
		if _, err := createFieldInfos(invalid); err == nil {
			t.Error("createHeaders did not produce error for bad struct")
		}
	}
}

func TestUnmarshalValidCSV(t *testing.T) {
	goodData := []string{
		`FIELD_0;FIELD_1;FIELD_2;FIELD_3
string1;1;true;1.14
string2;2;true;2.14
string3;3;true;3.14`,
		`FIELD_0;FIELD_2;FIELD_1;FIELD_3
string1;true;1;1.14
string2;true;2;2.14
string3;true;3;3.14`,
		`FIELD_2;FIELD_1;FIELD_3;FIELD_0
true;1;1.14;string1
true;2;2.14;string2
true;3;3.14;string3`,
	}
	for _, d := range goodData {
		r := strings.NewReader(d)
		m, err := NewMarshaler(TestStruct{}, r)
		m.Reader.Comma = ';'
		if err != nil {
			t.Fatal(err)
		}

		result, err := m.Unmarshal()
		if err != nil {
			t.Fatalf("error in UnmarshalCSV: %s", err)
		}
		if !reflect.DeepEqual(result[0], firstLine) {
			t.Errorf("wrong value '%v' for first line '%v'", result[0], firstLine)
		}
	}
}

func TestUnmarshallInvalidCSV(t *testing.T) {
	var parseErrorsTests = map[string]struct {
		data string
		err  error
	}{
		"not enough fields": {`FIELD_0;FIELD_2;FIELD_1;FIELD_3
string1;true;1
string2;true;2;1.14
string3;true;3`, csv.ErrFieldCount},
		"to many fields": {`FIELD_0;FIELD_1;FIELD_2;FIELD_3
string1;1;true;1.14;to much
sring2;2;false;2.14
string3;3;true;3.14;really to much`, csv.ErrFieldCount},
		"not enough headers": {`FIELD_0;FIELD_2;FIELD_1
string1;true;1;1.14
string2;true;2;2.14
string3;true;3;3.14`, ErrHeaderNotComplete},
	}

	for name, test := range parseErrorsTests {
		r := strings.NewReader(test.data)
		m, err := NewMarshaler(TestStruct{}, r)
		m.Reader.Comma = ';'
		if err != nil {
			t.Fatal(err)
		}
		_, err = m.Unmarshal()
		if err == nil {
			t.Errorf("no error occured for test '%s', but it should", name)
		} else {
			if pe, ok := err.(*csv.ParseError); ok {
				if pe.Err != test.err {
					t.Errorf("wrong error for test '%s': got: %s, wanted %s", name, pe, test.err)
				}
			} else {
				t.Errorf("test '%s': did not produce cve.ParseError, but should", name)
			}
		}
	}

	wrongTypes := `FIELD_0;FIELD_1;FIELD_2;FIELD_3
string1;notvalide;true;1.14
string2;2;notvalid;2.14
string3;3;true;not.valid`
	r := strings.NewReader(wrongTypes)
	m, err := NewMarshaler(TestStruct{}, r)
	m.Reader.Comma = ';'
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Unmarshal()
	if err == nil {
		t.Error("no error occured for wrong types test, but it should have")
	} else {
		if pe, ok := err.(ParseErrors); ok {
			if len(pe) != 3 {
				t.Errorf("not enouhg errors produced for wrong types test, got %d, want %d", len(pe), 3)
			}
		} else {
			t.Errorf("wrong error produced for wrong types test: ", err)
		}
	}

}
