package go_xml

import (
	"sync"
)

type Node interface {
	Accept(visitor Visitor) error
	Reset()
}

type Visitor interface {
	VisitElement(node *ElementNode) error
	VisitText(node *TextNode) error
}

type Attribute struct {
	Name  string
	Value string
}

type ElementNode struct {
	Name       string
	Attributes []Attribute
	Children   []Node
	SelfClose  bool
}

type TextNode struct {
	Text string
}

var (
	elementNodePool = sync.Pool{
		New: func() interface{} {
			return &ElementNode{
				Attributes: make([]Attribute, 0, 5),
				Children:   make([]Node, 0, 5),
			}
		},
	}

	textNodePool = sync.Pool{
		New: func() interface{} {
			return &TextNode{}
		},
	}
)

func acquireElementNode() *ElementNode {
	node := elementNodePool.Get().(*ElementNode)
	node.Reset()
	return node
}

func releaseElementNode(node *ElementNode) {
	elementNodePool.Put(node)
}

func acquireTextNode() *TextNode {
	node := textNodePool.Get().(*TextNode)
	node.Reset()
	return node
}

func releaseTextNode(node *TextNode) {
	textNodePool.Put(node)
}

func (n *ElementNode) Accept(visitor Visitor) error {
	return visitor.VisitElement(n)
}

func (n *ElementNode) Reset() {
	n.Name = ""
	n.Attributes = n.Attributes[:0]
	n.Children = n.Children[:0]
	n.SelfClose = false
}

func (n *TextNode) Accept(visitor Visitor) error {
	return visitor.VisitText(n)
}

func (n *TextNode) Reset() {
	n.Text = ""
}

func (n *ElementNode) HasAttribute(name string) bool {
	for _, attr := range n.Attributes {
		if attr.Name == name {
			return true
		}
	}
	return false
}
