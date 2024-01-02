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
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	dashscopeembeddings "github.com/kubeagi/arcadia/pkg/embeddings/dashscope"
	"github.com/kubeagi/arcadia/pkg/llms/dashscope"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/chroma"
	"k8s.io/klog/v2"
)

const (
	DontKnow = "这个问题我不知道哦😯"
)

type Bot struct {
	DB   chroma.Store
	Tree *Tree
	AI   *dashscope.DashScope
}

func NewDashScope(apiKey string, chromaURL, namespace string) (*Bot, error) {
	embedder := dashscopeembeddings.NewDashScopeEmbedder(apiKey)
	chromadb, err := chroma.New(
		chroma.WithChromaURL(chromaURL),
		chroma.WithEmbedder(embedder),
		chroma.WithNameSpace(namespace),
	)
	if err != nil {
		return nil, err
	}
	dashes := dashscope.NewDashScope(apiKey, false)
	return &Bot{
		DB:   chromadb,
		AI:   dashes,
		Tree: NewTree(""),
	}, nil
}

func (d *Bot) EmbeddingFileTitle(ctx context.Context, fileName string) (err error) {
	if err = d.Tree.ParseFile(fileName); err != nil {
		return err
	}
	var documents []schema.Document
	for _, t := range d.Tree.GetFullTitles() {
		documents = append(documents, schema.Document{PageContent: t})
	}
	//fmt.Println(d.Tree.String())
	_, err = d.DB.AddDocuments(ctx, documents)
	return err
}

func (d *Bot) Query(ctx context.Context, text string, chatHistory []string, lastNode *Node) (res string, foundNode *Node, err error) {
	var titles []string
	if lastNode != nil {
		titles = lastNode.PreOrder("fullTitle")
	} else {
		titles = d.Tree.GetFullTitles()
	}
	res = DontKnow
	var allInput bytes.Buffer
	for i := 0; ; i++ {
		if i*2 >= len(chatHistory) {
			break
		}
		userInput := chatHistory[i*2]
		aiResp := chatHistory[i*2+1]
		if userInput == "" || aiResp == DontKnow {
			continue
		}
		allInput.WriteString(userInput)
		allInput.WriteByte(' ')
	}
	allInput.WriteString(text)
	// TODO 先不考虑不匹配
	titlesWithIndex := make([]string, len(titles))
	for i, t := range titles {
		titlesWithIndex[i] = fmt.Sprintf("%d.  %s", i, strings.TrimSpace(t))
	}
	prompt := fmt.Sprintf("请根据以下问题和标题，找出问题最符合的标题序号，然后只输出该序号，不要包含额外的文字或标点符号。\n\n问题: %s\n\n标题: \n%s", allInput.String(), strings.Join(titlesWithIndex, "\n"))
	//fmt.Printf("prompt:%s\n", prompt)
	resp, err := d.Chat(ctx, prompt, nil)
	if err != nil {
		return res, nil, err
	}
	var wantTitle string
	//fmt.Printf("resp:%s\n", resp)
	index, err := strconv.Atoi(resp)
	if err != nil || index < 0 || index >= len(titles) {
		wantTitleDoc, err := d.DB.SimilaritySearch(ctx, allInput.String(), 1)
		if err != nil || len(wantTitleDoc) == 0 {
			return res, nil, err
		}
		wantTitle = wantTitleDoc[0].PageContent
	} else {
		wantTitle = titles[index]
	}

	//wantTitleDoc, err := d.DB.SimilaritySearch(ctx, allInput.String(), 1)
	//if err != nil || len(wantTitleDoc) == 0 {
	//	return res, err
	//}
	//wantTitle := wantTitleDoc[0].PageContent
	//fmt.Printf("get title:%s\n", wantTitle)

	foundNode = d.Tree.FindNodeByFullTitile(wantTitle)
	if foundNode.IsLeaf() {
		prompt := fmt.Sprintf("我将提供一些内容并提出一个问题，您应该根据我提供的内容来回答。请使用您的知识和理解来回答下列问题，如果不知道，请回复'不知道':\n问题：%s\n---\n内容：\n%s%s%s", allInput.String(), foundNode.FullTitle, foundNode.Question, foundNode.Text)
		//fmt.Printf("prompt:%s\n", prompt)
		resp, err = d.Chat(ctx, prompt, nil)
		//fmt.Printf("resp:%s\n", resp)
		return resp, foundNode, err
	} else {
		return foundNode.Question, foundNode, nil
	}
}

func (d *Bot) Chat(ctx context.Context, prompt string, history []string) (res string, err error) {
	params := dashscope.DefaultModelParams()
	params.Input.Messages = make([]dashscope.Message, 0)
	for i, h := range history {
		if h == "" {
			continue
		}
		role := dashscope.User
		if i%2 == 1 {
			role = dashscope.Assistant
		}
		params.Input.Messages = append(params.Input.Messages, dashscope.Message{Role: role, Content: h})
	}
	params.Input.Messages = append(params.Input.Messages, dashscope.Message{Role: dashscope.User, Content: prompt})
	klog.V(4).Info("message: %s\n", params.Input.Messages)
	resp, err := d.AI.Call(params.Marshal())
	if err != nil {
		return "", err
	}
	res = resp.String()
	klog.V(4).Info("resp: %s\n", res)
	return res, nil
}
