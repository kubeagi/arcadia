/*
Copyright 2023 KubeAGI.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package pkg

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
)

const (
	TitleFlag = "#"
)

type Tree struct {
	Text     string
	Children []*Node
}

type Node struct {
	RawTitle  string
	FullTitle string
	Text      string
	Level     int
	Question  string
	Children  []*Node
}

func NewTree(title string) *Tree {
	return &Tree{Text: title}
}

func (t *Tree) ParseFile(fileName string) (err error) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	var buf bytes.Buffer
	var lastNode *Node
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, TitleFlag) {
			level := strings.Count(line, TitleFlag) // 因为是自己格式化数据，暂时不考虑格式不对的情况
			title, question, _ := strings.Cut(line, " // ")
			node := &Node{Level: level, RawTitle: strings.TrimPrefix(title, strings.Repeat(TitleFlag, level)), Question: question}
			if level == 1 {
				node.FullTitle = node.RawTitle
			}
			if lastNode != nil {
				lastNode.Text = buf.String()
			}
			t.AddNode(node, lastNode)
			buf.Reset()
			lastNode = node
		} else {
			if _, err = buf.WriteString(line + "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

// 目前不考虑格式不对的情况，比如 nodeLevel=4, lastLevel=2 这种情况
func (t *Tree) AddNode(n, last *Node) {
	if n.Level == 1 || last == nil {
		t.Children = append(t.Children, n)
		return
	}
	v := last.Level - n.Level
	switch v {
	case -1:
		n.FullTitle = last.FullTitle + " " + n.RawTitle
		last.Children = append(last.Children, n)
	case 0:
		p := t.FindNodeParent(last)
		n.FullTitle = p.FullTitle + " " + n.RawTitle
		p.Children = append(p.Children, n)
	default:
		var p *Node
		for i := 0; i <= v; i++ {
			p = t.FindNodeParent(last)
			last = p
		}
		n.FullTitle = p.FullTitle + " " + n.RawTitle
		p.Children = append(p.Children, n)
	}
}

func (t *Tree) FindNodeParent(n *Node) *Node {
	for _, p := range t.Children {
		if node, find := containsNode(p, n); find {
			return node
		}
	}
	return nil
}

func containsNode(parent, child *Node) (*Node, bool) {
	for _, c := range parent.Children {
		if c == child {
			return parent, true
		}
		if node, find := containsNode(c, child); find {
			return node, true
		}
	}
	return nil, false
}

func (n *Node) PreOrder(option string) []string {
	if n == nil {
		return nil
	}
	var res []string
	switch option {
	case "rawTitle":
		res = []string{strings.Repeat("-", n.Level) + n.RawTitle}
	case "fullTitle":
		res = []string{n.FullTitle}
	}

	for _, child := range n.Children {
		res = append(res, child.PreOrder(option)...)
	}

	return res
}

func (t *Tree) String() string {
	if t == nil || len(t.Children) == 0 {
		return ""
	}
	res := make([]string, 0)
	for _, n := range t.Children {
		res = append(res, n.PreOrder("rawTitle")...)
	}
	return strings.Join(res, "\n")
}

func (t *Tree) FindNodeByFullTitile(title string) *Node {
	for _, p := range t.Children {
		if node, find := findNodeByFullTitle(p, title); find {
			return node
		}
	}
	return nil
}
func findNodeByFullTitle(node *Node, title string) (*Node, bool) {
	if node.FullTitle == title {
		return node, true
	}
	for _, c := range node.Children {
		if node, find := findNodeByFullTitle(c, title); find {
			return node, true
		}
	}
	return nil, false
}

func (t *Tree) GetFullTitles() []string {
	if t == nil || len(t.Children) == 0 {
		return nil
	}
	res := make([]string, 0)
	for _, n := range t.Children {
		res = append(res, n.PreOrder("fullTitle")...)
	}
	return res
}

func (n *Node) IsLeaf() bool {
	return len(n.Children) == 0
}
