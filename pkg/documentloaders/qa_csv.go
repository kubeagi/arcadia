package documentloaders

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

const (
	QuestionCol     = "q"
	AnswerCol       = "a"
	FileNameCol     = "file_name"
	PageNumberCol   = "page_number"
	ChunkContentCol = "chunk_content"
	LineNumber      = "line_number"
	QAFileName      = "qafile_name"
)

// QACSV represents a QA CSV document loader.
type QACSV struct {
	r                  io.Reader
	fileName           string
	questionColumn     string
	answerColumn       string
	fileNameColumn     string
	pageNumberColumn   string
	chunkContentColumn string
}

var _ documentloaders.Loader = QACSV{}

// Option is a function type that can be used to modify the client.
type Option func(p *QACSV)

func WithQuestionColumn(s string) Option {
	return func(p *QACSV) {
		p.questionColumn = s
	}
}
func WithAnswerColumn(s string) Option {
	return func(p *QACSV) {
		p.answerColumn = s
	}
}
func WithFileNameColumn(s string) Option {
	return func(p *QACSV) {
		p.fileNameColumn = s
	}
}
func WithPageNumberColumn(s string) Option {
	return func(p *QACSV) {
		p.pageNumberColumn = s
	}
}
func WithChunkContentColumn(s string) Option {
	return func(p *QACSV) {
		p.chunkContentColumn = s
	}
}

// NewQACSV creates a new qa csv loader with an io.Reader and optional column names for filtering.
func NewQACSV(r io.Reader, fileName string, opts ...Option) QACSV {
	q := QACSV{
		r:                  r,
		fileName:           fileName,
		questionColumn:     QuestionCol,
		answerColumn:       AnswerCol,
		fileNameColumn:     FileNameCol,
		pageNumberColumn:   PageNumberCol,
		chunkContentColumn: ChunkContentCol,
	}
	for _, opt := range opts {
		opt(&q)
	}
	return q
}

// Load reads from the io.Reader and returns a single document with the data.
func (c QACSV) Load(_ context.Context) ([]schema.Document, error) {
	var header []string
	var docs []schema.Document
	var rown int
	cols := []string{c.questionColumn, c.answerColumn, c.fileNameColumn, c.pageNumberColumn, c.chunkContentColumn}

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
		doc.Metadata = make(map[string]any, len(cols)-1)
		for i, value := range row {
			value = strings.TrimSpace(value)
			switch header[i] {
			case c.questionColumn:
				doc.PageContent = fmt.Sprintf("%s: %s", header[i], value)
			case c.answerColumn:
				doc.Metadata[c.answerColumn] = value
				doc.Metadata[QAFileName] = c.fileName
				doc.Metadata[LineNumber] = strconv.Itoa(rown)
			case c.pageNumberColumn:
				doc.Metadata[PageNumberCol] = value
			case c.fileNameColumn:
				doc.Metadata[FileNameCol] = value
			case c.chunkContentColumn:
				doc.Metadata[ChunkContentCol] = value
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
	return c.Load(ctx)
}
