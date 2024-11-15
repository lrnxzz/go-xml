package go_xml

import (
	"reflect"
	"strings"
	"sync"
)

type fieldMeta struct {
	Name      string
	FieldType reflect.StructField
}

var fieldCache sync.Map

func GetFieldMetadata(t reflect.Type) []fieldMeta {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.([]fieldMeta)
	}

	var fields []fieldMeta
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		xmlTag := field.Tag.Get("xml")
		if xmlTag == "-" {
			continue
		}
		tagParts := strings.Split(xmlTag, ",")
		tagName := tagParts[0]
		if tagName == "" {
			tagName = field.Name
		}
		fields = append(fields, fieldMeta{
			Name:      tagName,
			FieldType: field,
		})
	}

	fieldCache.Store(t, fields)
	return fields
}
