package gofpdi

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPdfReaderFromStream(t *testing.T) {
	t.Run("bad_pdf", func(t *testing.T) {
		input, err := ioutil.ReadFile("test_files/bad_pdf.pdf")
		if err != nil {
			t.Fatalf("failed to read test file %s: %v", "bad_pdf.pdf", err)
		}
		rs := io.ReadSeeker(bytes.NewReader(input))
		reader, err := NewPdfReaderFromStream("bad_pdf.pdf", rs)
		assert.Nil(t, reader)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "Failed to find startxref token")
	})
}

func TestResolveObjectNilSpec(t *testing.T) {
	reader := &PdfReader{}
	result, err := reader.resolveObject(nil)
	assert.Nil(t, result)
	assert.EqualError(t, err, "cannot resolve nil object")
}

func TestReadPagesMissingPagesEntry(t *testing.T) {
	reader := &PdfReader{
		catalog: &PdfValue{
			Value: &PdfValue{
				Dictionary: map[string]*PdfValue{},
			},
		},
	}

	err := reader.readPages()
	assert.EqualError(t, err, "catalog missing /Pages entry")
}
