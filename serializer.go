package go_xml

import (
	"fmt"
	"reflect"
	"strings"
)

type MarshalOptions struct {
	Indent          string
	XMLHeader       bool
	Namespace       string
	RootTag         string
	Compress        bool
	SelfClosingTags []string
}

func Marshal(v interface{}, opts *MarshalOptions) ([]byte, error) {
	if opts == nil {
		opts = &MarshalOptions{}
	}

	rootTag := opts.RootTag
	if rootTag == "" {
		rootTag = reflect.TypeOf(v).Name()
	}

	node, err := structToNode(reflect.ValueOf(v), opts, rootTag)
	if err != nil {
		return nil, err
	}

	if node == nil {
		return nil, fmt.Errorf("returned node is nil")
	}

	buf := acquireBuffer()
	defer releaseBuffer(buf)

	var selfClosingTags []string
	if opts != nil {
		selfClosingTags = opts.SelfClosingTags
	}

	encoder := NewEncoder(buf, selfClosingTags, opts.Indent)

	if opts.XMLHeader {
		_, err := buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		if err != nil {
			return nil, err
		}
	}

	if opts.Namespace != "" {
		if elementNode, ok := node.(*ElementNode); ok {
			elementNode.Attributes = append(elementNode.Attributes, Attribute{
				Name:  "xmlns",
				Value: opts.Namespace,
			})
		}
	}

	err = node.Accept(encoder)
	if err != nil {
		return nil, err
	}

	if opts.Compress {
		compressor := acquireCompressor()
		defer releaseCompressor(compressor)

		compressedBuf, err := compressor.Compress(buf)
		if err != nil {
			return nil, err
		}
		defer releaseBuffer(compressedBuf)
		return compressedBuf.Bytes(), nil
	}

	return buf.Bytes(), nil
}

func structToNode(val reflect.Value, opts *MarshalOptions, xmlTag string) (Node, error) {
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	typ := val.Type()
	kind := val.Kind()

	var tagHierarchy []string
	if xmlTag != "" {
		tagHierarchy = strings.Split(xmlTag, ">")
	}

	if kind == reflect.Struct {
		var name string
		if len(tagHierarchy) > 0 {
			name = tagHierarchy[0]
			tagHierarchy = tagHierarchy[1:]
		} else {
			name = typ.Name()
		}

		element := acquireElementNode()
		element.Name = name

		if opts != nil {
			for _, tag := range opts.SelfClosingTags {
				if tag == name {
					element.SelfClose = true
					break
				}
			}
		}

		fields := GetFieldMetadata(typ)

		for _, fieldMeta := range fields {
			field := fieldMeta.FieldType
			fieldValue := val.FieldByIndex(field.Index)
			xmlTag := field.Tag.Get("xml")
			if xmlTag == "-" {
				continue
			}
			tagParts := strings.Split(xmlTag, ",")
			tagName := tagParts[0]
			if tagName == "" {
				tagName = field.Name
			}
			tagOptions := ""
			if len(tagParts) > 1 {
				tagOptions = tagParts[1]
			}

			if tagOptions == "attr" {
				attrValue := valueToString(fieldValue)
				element.Attributes = append(element.Attributes, Attribute{
					Name:  tagName,
					Value: attrValue,
				})
			} else {
				fullTag := tagName
				if len(tagHierarchy) > 0 {
					fullTag = strings.Join(append(tagHierarchy, tagName), ">")
				}
				childNode, err := structToNode(fieldValue, opts, fullTag)
				if err != nil {
					return nil, err
				}
				if childNode != nil {
					element.Children = append(element.Children, childNode)
				}
			}
		}

		if element.SelfClose && len(element.Children) == 0 && len(element.Attributes) == 0 {
			return element, nil
		} else {
			element.SelfClose = false
		}

		return element, nil
	}

	if kind == reflect.Slice || kind == reflect.Array {
		if len(tagHierarchy) == 0 {
			return nil, fmt.Errorf("missing XML tag for slice or array")
		}
		name := tagHierarchy[0]
		tagHierarchy = tagHierarchy[1:]

		nodes := make([]Node, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			childNode, err := structToNode(val.Index(i), opts, strings.Join(tagHierarchy, ">"))
			if err != nil {
				return nil, err
			}
			if childNode != nil {
				nodes = append(nodes, childNode)
			}
		}

		element := acquireElementNode()
		element.Name = name
		element.Children = nodes

		if opts != nil {
			for _, tag := range opts.SelfClosingTags {
				if tag == name && len(nodes) == 0 {
					element.SelfClose = true
					break
				}
			}
		}

		return element, nil
	}

	if len(tagHierarchy) > 0 {
		name := tagHierarchy[0]
		tagHierarchy = tagHierarchy[1:]

		element := acquireElementNode()
		element.Name = name

		textContent := valueToString(val)
		if textContent == "" {
			if opts != nil {
				for _, tag := range opts.SelfClosingTags {
					if tag == name {
						element.SelfClose = true
						break
					}
				}
			}
		}

		if !element.SelfClose {
			textNode := acquireTextNode()
			textNode.Text = textContent
			element.Children = append(element.Children, textNode)
		}

		return element, nil
	}

	textNode := acquireTextNode()
	textNode.Text = valueToString(val)
	return textNode, nil
}

func valueToString(val reflect.Value) string {
	switch val.Kind() {
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", val.Float())
	case reflect.String:
		return val.String()
	default:
		return fmt.Sprintf("%v", val.Interface())
	}
}
