package go_xml

import (
	"io"
	"strings"
)

type Encoder struct {
	w           io.Writer
	selfClosing map[string]bool
	indent      string
	depth       int
}

func NewEncoder(w io.Writer, selfClosingTags []string, indent string) *Encoder {
	selfClosing := make(map[string]bool)
	for _, tag := range selfClosingTags {
		selfClosing[tag] = true
	}
	return &Encoder{
		w:           w,
		selfClosing: selfClosing,
		indent:      indent,
		depth:       0,
	}
}

func (e *Encoder) writeIndent() error {
	if e.indent != "" {
		_, err := e.w.Write([]byte(strings.Repeat(e.indent, e.depth)))
		return err
	}
	return nil
}

func (e *Encoder) VisitElement(node *ElementNode) error {
	if e.depth > 0 {
		if _, err := e.w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if err := e.writeIndent(); err != nil {
		return err
	}

	if _, err := e.w.Write([]byte("<" + node.Name)); err != nil {
		return err
	}

	for _, attr := range node.Attributes {
		if _, err := e.w.Write([]byte(" " + attr.Name + "=\"" + escapeString(attr.Value) + "\"")); err != nil {
			return err
		}
	}

	shouldSelfClose := node.SelfClose || (e.selfClosing[node.Name] && !hasNonEmptyChildren(node))

	if shouldSelfClose {
		if _, err := e.w.Write([]byte("/>")); err != nil {
			return err
		}
		releaseElementNode(node)
		return nil
	}

	if _, err := e.w.Write([]byte(">")); err != nil {
		return err
	}

	e.depth++
	for _, child := range node.Children {
		if err := child.Accept(e); err != nil {
			return err
		}
	}
	e.depth--

	if len(node.Children) > 0 {
		if _, isElement := node.Children[len(node.Children)-1].(*ElementNode); isElement {
			if _, err := e.w.Write([]byte("\n")); err != nil {
				return err
			}
			if err := e.writeIndent(); err != nil {
				return err
			}
		}
	}

	if _, err := e.w.Write([]byte("</" + node.Name + ">")); err != nil {
		return err
	}
	releaseElementNode(node)
	return nil
}

func (e *Encoder) VisitText(node *TextNode) error {
	if _, err := e.w.Write([]byte(escapeString(node.Text))); err != nil {
		return err
	}
	releaseTextNode(node)
	return nil
}
