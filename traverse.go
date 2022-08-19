package objpath

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// TraverseStruct traverse struct
func TraverseStruct(v reflect.Value, fn func(value reflect.Value, field *reflect.StructField, parentField *reflect.StructField)) {
	traverseStruct(v, nil, fn)
}
func traverseStruct(v reflect.Value, parentField *reflect.StructField, fn func(value reflect.Value, field *reflect.StructField, parentField *reflect.StructField)) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)
		if fieldType.Anonymous {
			// unwrap indrect
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					continue
				}
				field = field.Elem()
			}
			if field.Kind() != reflect.Struct {
				continue
			}

			traverseStruct(field, &fieldType, fn)
			continue
		}
		fn(field, &fieldType, parentField)
	}
}

// GetExportedJSONName get json name that will appear in marshaled json
func GetExportedJSONName(fieldType *reflect.StructField) (jsonName string, omitEmpty bool) {
	fieldName := fieldType.Name
	// / must be exported
	if strings.ToUpper(fieldName[0:1]) != fieldName[0:1] {
		return
	}
	jsonTag := fieldType.Tag.Get("json")
	idx := strings.Index(jsonTag, ",")
	jsonName = jsonTag
	if idx >= 0 {
		jsonName = jsonTag[:idx]
		jsonOpts := strings.Split(jsonTag[idx+1:], ",")
		for _, opt := range jsonOpts {
			if opt == "omitempty" {
				omitEmpty = true
				break
			}
		}
	}
	if jsonName == "-" {
		jsonName = ""
		return // ignored
	}
	// omit empty
	if jsonName == "" {
		jsonName = fieldName
	}
	return
}

func IsZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	if v.IsZero() {
		return true
	}
	switch v.Kind() {
	case reflect.Map:
		return v.Len() == 0
	case reflect.Slice:
		return v.Len() == 0
	}
	return false
}

// GetPrimitiveString returns v's underlying int,uint,bool's
// string, ignoring their String() method.(unlike fmt.Sprint())
func GetPrimitiveString(v reflect.Value) (string, bool) {
	switch v.Kind() {
	case reflect.String:
		return v.String(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), true
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), true
	case reflect.Float64, reflect.Float32:
		return fmt.Sprint(v.Float()), true
	default:
		return "", false
	}
}
