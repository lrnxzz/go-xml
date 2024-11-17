package go_xml

import (
	"encoding/xml"
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

	node, err := structToNode(reflect.ValueOf(v), opts, []string{rootTag})
	if err != nil {
		return nil, fmt.Errorf("error converting structure to node: %w", err)
	}

	if node == nil {
		return nil, fmt.Errorf("returned node is null")
	}

	buf := acquireBuffer()
	defer releaseBuffer(buf)

	encoder := NewEncoder(buf, opts.SelfClosingTags, opts.Indent)

	if opts.XMLHeader {
		if _, err := buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"); err != nil {
			return nil, err
		}
		if opts.Indent != "" {
			buf.WriteString("\n")
		}
	}

	if opts.Namespace != "" {
		if elementNode, ok := node.(*ElementNode); ok {
			if !elementNode.HasAttribute("xmlns") {
				elementNode.Attributes = insertAttributeAtBeginning(elementNode.Attributes, Attribute{
					Name:  "xmlns",
					Value: opts.Namespace,
				})
			}
		}
	}

	if err := node.Accept(encoder); err != nil {
		return nil, fmt.Errorf("error encoding node: %w", err)
	}

	if opts.Compress {
		compressor := acquireCompressor()
		defer releaseCompressor(compressor)

		compressedBuf, err := compressor.Compress(buf)
		if err != nil {
			return nil, fmt.Errorf("error compressing data: %w", err)
		}
		defer releaseBuffer(compressedBuf)
		return compressedBuf.Bytes(), nil
	}

	return buf.Bytes(), nil
}

func insertAttributeAtBeginning(attrs []Attribute, attr Attribute) []Attribute {
	newAttrs := make([]Attribute, len(attrs)+1)
	newAttrs[0] = attr
	copy(newAttrs[1:], attrs)
	return newAttrs
}

func structToNode(val reflect.Value, opts *MarshalOptions, tagHierarchy []string) (Node, error) {
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	if len(tagHierarchy) == 0 {
		return nil, fmt.Errorf("tag hierarchy is empty")
	}

	currentTag := tagHierarchy[0]
	remainingTags := tagHierarchy[1:]

	switch val.Kind() {
	case reflect.Struct:
		return handleStructNode(val, currentTag, opts)
	case reflect.Slice, reflect.Array:
		return handleSliceNode(val, currentTag, remainingTags, opts)
	default:
		return handleSimpleNode(val, currentTag)
	}
}

func handleStructNode(val reflect.Value, currentTag string, opts *MarshalOptions) (Node, error) {
	element := acquireElementNode()
	element.Name = currentTag

	typ := val.Type()
	fields := GetFieldMetadata(typ)

	for _, fieldMeta := range fields {
		field := fieldMeta.FieldType
		fieldValue := val.FieldByIndex(field.Index)

		if field.Type == reflect.TypeOf(xml.Name{}) {
			continue
		}

		xmlTag := field.Tag.Get("xml")
		if xmlTag == "-" {
			continue
		}

		tagParts := strings.Split(xmlTag, ",")
		tagName := tagParts[0]
		if tagName == "" {
			tagName = field.Name
		}

		var tagOptions []string
		if len(tagParts) > 1 {
			tagOptions = tagParts[1:]
		}

		if err := processField(element, fieldValue, tagName, tagOptions, opts); err != nil {
			return nil, err
		}
	}

	return element, nil
}

func handleSliceNode(val reflect.Value, currentTag string, remainingTags []string, opts *MarshalOptions) (Node, error) {
	element := acquireElementNode()
	element.Name = currentTag

	for i := 0; i < val.Len(); i++ {
		itemValue := val.Index(i)
		childNode, err := structToNode(itemValue, opts, remainingTags)
		if err != nil {
			return nil, err
		}
		if childNode != nil {
			element.Children = append(element.Children, childNode)
		}
	}

	return element, nil
}

func handleSimpleNode(val reflect.Value, currentTag string) (Node, error) {
	element := acquireElementNode()
	element.Name = currentTag

	textNode := acquireTextNode()
	textNode.Text = valueToString(val)
	element.Children = append(element.Children, textNode)

	return element, nil
}

func processField(element *ElementNode, fieldValue reflect.Value, tagName string, tagOptions []string, opts *MarshalOptions) error {
	if contains(tagOptions, "attr") {
		attrValue := valueToString(fieldValue)
		element.Attributes = append(element.Attributes, Attribute{
			Name:  tagName,
			Value: attrValue,
		})
		return nil
	}

	if contains(tagOptions, "omitempty") && isEmptyValue(fieldValue) {
		return nil
	}

	var childTags []string
	if strings.Contains(tagName, ">") {
		childTags = strings.Split(tagName, ">")
	} else {
		childTags = []string{tagName}
	}

	return processChildTags(element, fieldValue, childTags, opts)
}

func processChildTags(element *ElementNode, fieldValue reflect.Value, childTags []string, opts *MarshalOptions) error {
	currentElement := element

	for i := 0; i < len(childTags)-1; i++ {
		newElement := acquireElementNode()
		newElement.Name = childTags[i]
		currentElement.Children = append(currentElement.Children, newElement)
		currentElement = newElement
	}

	lastTag := childTags[len(childTags)-1]

	if fieldValue.Kind() == reflect.Slice || fieldValue.Kind() == reflect.Array {
		for i := 0; i < fieldValue.Len(); i++ {
			childValue := fieldValue.Index(i)
			childNode, err := structToNode(childValue, opts, []string{lastTag})
			if err != nil {
				return err
			}
			if childNode != nil {
				currentElement.Children = append(currentElement.Children, childNode)
			}
		}
	} else {
		childNode, err := structToNode(fieldValue, opts, []string{lastTag})
		if err != nil {
			return err
		}
		if childNode != nil {
			currentElement.Children = append(currentElement.Children, childNode)
		}
	}

	return nil
}
