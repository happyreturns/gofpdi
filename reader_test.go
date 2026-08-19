package gofpdi

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestReadXrefMultiByteLastFieldSize(t *testing.T) {
	t.Run("W 1 2 2 decodes compressed object index from passport-style xref stream", func(t *testing.T) {
		// Pre-fix code read only the first byte of the 2-byte index field (0x00), not 0x0001.
		record := []byte{0x02, 0x00, 0x1b, 0x00, 0x01}
		reader := readXRefStreamFromSyntheticPDF(t, 2, 1, [3]int{1, 2, 2}, record)

		assert.Equal(t, [2]int{27, 1}, reader.xrefStream[2])
	})

	t.Run("W 1 4 2 decodes in-use object generation", func(t *testing.T) {
		record := []byte{0x01, 0x00, 0x00, 0x00, 0x64, 0x00, 0x01}
		reader := readXRefStreamFromSyntheticPDF(t, 1, 1, [3]int{1, 4, 2}, record)

		assert.Equal(t, 100, reader.xref[1][1])
	})

	t.Run("W 1 4 2 decodes compressed object index", func(t *testing.T) {
		record := []byte{0x02, 0x00, 0x00, 0x00, 0x1b, 0x00, 0x03}
		reader := readXRefStreamFromSyntheticPDF(t, 2, 1, [3]int{1, 4, 2}, record)

		assert.Equal(t, [2]int{27, 3}, reader.xrefStream[2])
	})
}

func readXRefStreamFromSyntheticPDF(t *testing.T, indexStart, indexCount int, w [3]int, streamPayload []byte) *PdfReader {
	t.Helper()

	pdfBytes, xrefPos := buildSyntheticXRefStreamPDF(t, indexStart, indexCount, w, streamPayload)

	reader := &PdfReader{
		f:          bytes.NewReader(pdfBytes),
		nBytes:     int64(len(pdfBytes)),
		xref:       make(map[int]map[int]int),
		xrefStream: make(map[int][2]int),
		xrefPos:    xrefPos,
	}

	require.NoError(t, reader.readXref())
	return reader
}

func buildSyntheticXRefStreamPDF(t *testing.T, indexStart, indexCount int, w [3]int, streamPayload []byte) ([]byte, int) {
	t.Helper()

	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	_, err := zw.Write(streamPayload)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.7\n")
	xrefPos := pdf.Len()
	fmt.Fprintf(
		&pdf,
		"1 0 obj\n<< /Type /XRef /Size %d /Index [ %d %d ] /W [ %d %d %d ] /Root 1 0 R /Length %d /Filter /FlateDecode >>\nstream\n",
		indexStart+indexCount,
		indexStart,
		indexCount,
		w[0],
		w[1],
		w[2],
		compressed.Len(),
	)
	pdf.Write(compressed.Bytes())
	fmt.Fprintf(&pdf, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", xrefPos)

	return pdf.Bytes(), xrefPos
}
