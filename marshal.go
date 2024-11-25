package go_xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

const (
	xmlHeader = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
)

type MarshalOptions struct {
	Indent          string
	XMLHeader       bool
	Namespace       string
	RootTag         string
	Compress        bool
	SelfClosingTags []string
	SpacedSelfClose bool
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

	encoder := NewEncoder(buf, opts.SelfClosingTags, opts.Indent, opts.SpacedSelfClose)

	if opts.XMLHeader {
		if _, err := buf.WriteString(xmlHeader); err != nil {
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
		return compressBuffer(buf)
	}

	return buf.Bytes(), nil
}

func compressBuffer(buf *bytes.Buffer) ([]byte, error) {
	compressor := acquireCompressor()
	defer releaseCompressor(compressor)

	compressedBuf, err := compressor.Compress(buf)
	if err != nil {
		return nil, fmt.Errorf("error compressing data: %w", err)
	}
	defer releaseBuffer(compressedBuf)
	return compressedBuf.Bytes(), nil
}

func structToNode(val reflect.Value, opts *MarshalOptions, tagHierarchy []string) (Node, error) {
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	currentTag := ""
	remainingTags := tagHierarchy
	if len(tagHierarchy) > 0 {
		currentTag = tagHierarchy[0]
		remainingTags = tagHierarchy[1:]
	}

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

		if field.Anonymous {
			if err := processAnonymousField(element, fieldValue, opts); err != nil {
				return nil, err
			}
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

		if field.Type == reflect.TypeOf(xml.Name{}) {
			if xmlName, ok := fieldValue.Interface().(xml.Name); ok && xmlName.Local != "" {
				element.Name = xmlName.Local
			}
			continue
		}

		if err := processField(element, fieldValue, tagName, tagOptions, opts); err != nil {
			return nil, err
		}
	}

	return element, nil
}

func processAnonymousField(element *ElementNode, fieldValue reflect.Value, opts *MarshalOptions) error {
	embeddedNode, err := structToNode(fieldValue, opts, []string{})
	if err != nil {
		return err
	}
	if embeddedElement, ok := embeddedNode.(*ElementNode); ok {
		element.Attributes = append(element.Attributes, embeddedElement.Attributes...)
		element.Children = append(element.Children, embeddedElement.Children...)
	}
	return nil
}

func handleSliceNode(val reflect.Value, currentTag string, remainingTags []string, opts *MarshalOptions) (Node, error) {
	element := acquireElementNode()
	element.Name = currentTag

	childNodes := make([]Node, val.Len())
	wg := sync.WaitGroup{}
	errChan := make(chan error, 1)
	var mutex sync.Mutex

	for i := 0; i < val.Len(); i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			childValue := val.Index(index)
			childNode, err := structToNode(childValue, opts, remainingTags)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
			mutex.Lock()
			childNodes[index] = childNode
			mutex.Unlock()
		}(i)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	for _, node := range childNodes {
		if node != nil {
			element.Children = append(element.Children, node)
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
