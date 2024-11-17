package go_xml

import (
	"fmt"
	"reflect"
	"strings"
)

func insertAttributeAtBeginning(attrs []Attribute, attr Attribute) []Attribute {
	newAttrs := make([]Attribute, len(attrs)+1)
	newAttrs[0] = attr
	copy(newAttrs[1:], attrs)
	return newAttrs
}

func hasNonEmptyChildren(node *ElementNode) bool {
	for _, child := range node.Children {
		switch c := child.(type) {
		case *ElementNode:
			return true
		case *TextNode:
			if c.Text != "" {
				return true
			}
		}
	}
	return false
}

func escapeString(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	for _, c := range s {
		switch c {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

func contains(options []string, opt string) bool {
	for _, o := range options {
		if o == opt {
			return true
		}
	}
	return false
}

func isEmptyValue(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return val.IsNil()
	}
	return false
}

func valueToString(val reflect.Value) string {
	// Dereference pointer types
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ""
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", val.Float())
	case reflect.String:
		return val.String()
	case reflect.Bool:
		return fmt.Sprintf("%t", val.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", val.Uint())
	default:
		return fmt.Sprintf("%v", val.Interface())
	}
}
