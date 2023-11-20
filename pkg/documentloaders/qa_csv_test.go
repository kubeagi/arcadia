package documentloaders

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVLoader(t *testing.T) {
	t.Parallel()
	fileName := "./testdata/qa.csv"
	file, err := os.Open(fileName)
	assert.NoError(t, err)

	loader := NewQACSV(file, fileName, "q", "a")

	docs, err := loader.Load(context.Background())
	require.NoError(t, err)
	require.Len(t, docs, 25)

	expected1PageContent := "q: 什么是员工考勤管理制度？"
	assert.Equal(t, docs[0].PageContent, expected1PageContent)

	expected1Metadata := map[string]any{
		"a":          "该制度旨在严格工作纪律、提高工作效率，规范公司考勤管理，为公司考勤管理提供明确依据。",
		"fileName":   fileName,
		"lineNumber": 0,
	}
	assert.Equal(t, docs[0].Metadata, expected1Metadata)

	expected2PageContent := "q: 该制度适用于哪些员工？"
	assert.Equal(t, docs[1].PageContent, expected2PageContent)

	expected2Metadata := map[string]any{
		"a":          "适用于公司全体正式员工及实习生。",
		"fileName":   fileName,
		"lineNumber": 1,
	}
	assert.Equal(t, docs[1].Metadata, expected2Metadata)
}
