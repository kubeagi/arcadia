package documentloaders

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// QACSV represents a QA CSV document loader.
type QACSV struct {
	r              io.Reader
	fileName       string
	questionColumn string
	answerColumn   string
}

var _ documentloaders.Loader = QACSV{}

// NewQACSV creates a new qa csv loader with an io.Reader and optional column names for filtering.
func NewQACSV(r io.Reader, fileName string, questionColumn string, answerColumn string) QACSV {
	if questionColumn == "" {
		questionColumn = "q"
	}
	if answerColumn == "" {
		answerColumn = "a"
	}
	return QACSV{
		r:              r,
		fileName:       fileName,
		questionColumn: questionColumn,
		answerColumn:   answerColumn,
	}
}

// Load reads from the io.Reader and returns a single document with the data.
func (c QACSV) Load(_ context.Context) ([]schema.Document, error) {
	var header []string
	var docs []schema.Document
	var rown int

	rd := csv.NewReader(c.r)
	for {
		row, err := rd.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(header) == 0 {
			header = append(header, row...)
			continue
		}

		doc := schema.Document{}
		for i, value := range row {
			if c.questionColumn != "" && header[i] != c.questionColumn && header[i] != c.answerColumn {
				continue
			}
			value = strings.TrimSpace(value)
			if header[i] == c.questionColumn {
				doc.PageContent = fmt.Sprintf("%s: %s", header[i], value)
			}
			if header[i] == c.answerColumn {
				doc.Metadata = map[string]any{
					c.answerColumn: value,
					"fileName":     c.fileName,
					"lineNumber":   rown,
				}
			}
		}
		rown++
		docs = append(docs, doc)
	}

	return docs, nil
}

// LoadAndSplit reads text data from the io.Reader and splits it into multiple
// documents using a text splitter.
func (c QACSV) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	docs, err := c.Load(ctx)
	if err != nil {
		return nil, err
	}

	return textsplitter.SplitDocuments(splitter, docs)
}
