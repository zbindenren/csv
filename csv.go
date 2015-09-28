package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/oleiade/reflections"
)

var (
	ErrNoStruct           = errors.New("interface is not a struct")
	ErrNoValidRecords     = errors.New("no valid records found")
	ErrHeaderNotComplete  = errors.New("header not complete")
	ErrUnsupportedCSVType = errors.New("unsupported csv type")
)

type stringSlice []string

type fieldInfo struct {
	position   int
	headerName string
	fieldName  string
	kind       reflect.Kind
}

type fieldInfos []fieldInfo

// isComplete checks if the all field positions could be detected from the csv file.
func (fieldInfos *fieldInfos) isComplete() bool {
	for _, fieldInfo := range *fieldInfos {
		if fieldInfo.position < 0 {
			return false
		}
	}
	return true
}

// createFieldInfos creates the fieldInfos for a struct s.
// Only information from the struct (headerName, fieldName and kind) is available,
// all field positions are initialized with an invalid value of -1
func createFieldInfos(s interface{}) (fieldInfos, error) {
	if reflect.TypeOf(s).Kind() != reflect.Struct {
		return nil, ErrNoStruct
	}
	fieldInfos := []fieldInfo{}
	headerNameMap := map[string]interface{}{} // to detect duplicate csv tag names
	fieldNames, err := reflections.Fields(s)
	if err != nil {
		return nil, err
	}
	for _, fieldName := range fieldNames {
		headerName, err := reflections.GetFieldTag(s, fieldName, "csv")
		if err != nil {
			return nil, err
		}
		// csv fieldtags that contain a dash are ignored
		if strings.Contains(headerName, "-") {
			continue
		}
		if _, ok := headerNameMap[headerName]; ok {
			return nil, fmt.Errorf("duplicate csv tag name: %s", headerName)
		}
		headerNameMap[headerName] = nil
		kind, err := reflections.GetFieldKind(s, fieldName)
		if err != nil {
			return nil, err
		}
		if len(headerName) == 0 {
			return nil, fmt.Errorf("empty csv tag for field: %s", fieldName)
		}
		fieldInfos = append(fieldInfos, fieldInfo{
			headerName: headerName,
			fieldName:  fieldName,
			position:   -1,
			kind:       kind,
		})
	}
	return fieldInfos, nil
}

func (s stringSlice) pos(item string) int {
	for i, v := range s {
		if item == v {
			return i
		}
	}
	return -1
}

// Marshaler reads a csv file and unmarshalls it to an endpoint struct.
type Marshaler struct {
	Reader         *csv.Reader
	fieldInfos     fieldInfos
	endPointStruct interface{}
	errors         ParseErrors
	Lazy           bool // if true, marshaler does not exit on first cvs.ParseError but coninues and appends errors
}

// NewMarshaler returns a new Marshaler
func NewMarshaler(endPointStruct interface{}, r io.Reader) (*Marshaler, error) {
	fieldInfos, err := createFieldInfos(endPointStruct)
	if err != nil {
		return nil, err
	}
	cr := csv.NewReader(r)
	return &Marshaler{
		Reader:         cr,
		fieldInfos:     fieldInfos,
		endPointStruct: endPointStruct,
		errors:         ParseErrors{},
	}, nil
}

// ParseErrors is a slice of csv.ParseError
type ParseErrors []csv.ParseError

// Error returns te ParseErrors as string
func (errs ParseErrors) Error() string {
	s := ""
	for _, err := range errs {
		s = s + fmt.Sprintf("line:%d,position:%d,err:%s\n", err.Line, err.Column, err.Error)
	}
	return s
}

// Unmarshal parses a csv file and stores its value to a list of entpoint structs
func (m *Marshaler) Unmarshal() ([]interface{}, error) {
	structs := *new([]interface{})

	line := 0
	for {
		line++
		var record stringSlice
		record, err := m.Reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			if !m.Lazy {
				return nil, err
			}
			if pe, ok := err.(*csv.ParseError); ok {
				m.errors = append(m.errors, *pe)
			}
			continue
		}
		if line == 1 { // first line contains header information
			for i, fieldInfo := range m.fieldInfos {
				index := record.pos(fieldInfo.headerName)
				if index >= 0 {
					m.fieldInfos[i].position = index
				}
			}
			if !m.fieldInfos.isComplete() {
				return nil, &csv.ParseError{Err: ErrHeaderNotComplete}
			}
			continue
		}
		// if len(m.fieldInfos) > len(record) {
		// return nil, &csv.ParseError{Line: line, Err: errors.New("bla")}
		// }
		sPtr := reflect.New(reflect.TypeOf(m.endPointStruct)).Interface()
		for _, fieldInfo := range m.fieldInfos {
			var (
				value interface{}
				err   error
			)
			switch fieldInfo.kind {
			case reflect.Bool:
				value, err = strconv.ParseBool(record[fieldInfo.position])
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				value, err = strconv.Atoi(record[fieldInfo.position])
			case reflect.Float32, reflect.Float64:
				value, err = strconv.ParseFloat(record[fieldInfo.position], 64)
			case reflect.String:
				value = record[fieldInfo.position]
			default:
				err = ErrUnsupportedCSVType
			}
			if err != nil {
				m.errors = append(m.errors, csv.ParseError{
					Column: fieldInfo.position,
					Line:   line,
					Err:    err,
				})
				break
			}
			reflections.SetField(sPtr, fieldInfo.fieldName, value)
		}
		v := reflect.ValueOf(sPtr).Elem().Interface()
		structs = append(structs, v)
	}
	if len(m.errors) == 0 {
		return structs, nil
	}
	return structs, m.errors
}
