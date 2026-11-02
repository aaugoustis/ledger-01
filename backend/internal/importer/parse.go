// Package importer reads bank statement files (CSV and XLSX) into a grid of text cells and turns
// rows into amounts, dates and notes with a column mapping. It is pure: no database, no network,
// everything in memory and capped, so it is safe to run on untrusted uploads inside a Lambda.
package importer

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

const (
	// MaxFileBytes caps uploads. On AWS a request reaches the Lambda inside a 6 MB invocation
	// payload, base64-encoded in JSON, so ~4.4 MB of file is the most one request can carry.
	MaxFileBytes = 4 << 20
	// MaxRows caps the rows of one file (header included).
	MaxRows = 10_000
	// MaxColumns caps the columns kept per row.
	MaxColumns = 50
	// maxCell caps the characters kept per cell.
	maxCell = 500
)

// Grid is a file's rows as trimmed text cells. Rows can have different lengths.
type Grid [][]string

// Columns is the widest row's length.
func (g Grid) Columns() int {
	n := 0
	for _, r := range g {
		n = max(n, len(r))
	}
	return n
}

// Cell returns row r's column c, or "" when the row is shorter.
func (g Grid) Cell(r, c int) string {
	if r < 0 || r >= len(g) || c < 0 || c >= len(g[r]) {
		return ""
	}
	return g[r][c]
}

// FileInfo is what parsing detected; Encoding and Delimiter are empty for XLSX.
type FileInfo struct {
	Format    string // "csv" or "xlsx"
	Encoding  string // "UTF-8", "UTF-16" or "Windows-1252"
	Delimiter string // ",", ";", "\t" or "|"
}

// ErrUnsupported is returned for files that aren't CSV or XLSX (legacy .xls included).
var ErrUnsupported = errors.New("unsupported file")

// Parse reads a CSV or XLSX file. The format comes from the content (an XLSX is a zip), falling
// back to the file name.
func Parse(name string, content []byte) (Grid, FileInfo, error) {
	if len(content) == 0 {
		return nil, FileInfo{}, errors.New("the file is empty")
	}
	if len(content) > MaxFileBytes {
		return nil, FileInfo{}, fmt.Errorf("the file is larger than %d MB", MaxFileBytes>>20)
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case bytes.HasPrefix(content, []byte("PK\x03\x04")):
		g, err := parseXLSX(content)
		return g, FileInfo{Format: "xlsx"}, err
	case bytes.HasPrefix(content, []byte{0xD0, 0xCF, 0x11, 0xE0}) || ext == ".xls":
		return nil, FileInfo{}, fmt.Errorf("%w: save .xls files as .xlsx or .csv first", ErrUnsupported)
	case ext == ".xlsx":
		return nil, FileInfo{}, fmt.Errorf("%w: this .xlsx file is damaged", ErrUnsupported)
	}
	g, info, err := parseCSV(content)
	info.Format = "csv"
	return g, info, err
}

// decode turns CSV bytes into UTF-8 text: UTF-8 (with or without BOM), UTF-16 with a BOM, and
// otherwise Windows-1252 (a superset of Latin-1's printable characters).
func decode(b []byte) (string, string, error) {
	var enc encoding.Encoding
	name := "UTF-8"
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return string(b[3:]), name, nil
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}), bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		enc, name = unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM), "UTF-16"
	case utf8.Valid(b):
		return string(b), name, nil
	default:
		enc, name = charmap.Windows1252, "Windows-1252"
	}
	out, err := enc.NewDecoder().Bytes(b)
	if err != nil {
		return "", "", fmt.Errorf("the file's text encoding isn't supported: %w", err)
	}
	return string(out), name, nil
}

var delimiters = []rune{',', ';', '\t', '|'}

// detectDelimiter picks the delimiter that splits the first lines into the most columns, as
// consistently as possible.
func detectDelimiter(text string) rune {
	sample := text
	if i := nthIndex(text, '\n', 20); i >= 0 {
		sample = text[:i]
	}
	best, bestScore := ',', -1
	for _, d := range delimiters {
		r := csv.NewReader(strings.NewReader(sample))
		r.Comma, r.FieldsPerRecord, r.LazyQuotes = d, -1, true
		counts := map[int]int{}
		for {
			rec, err := r.Read()
			if err != nil {
				break
			}
			if len(rec) > 1 {
				counts[len(rec)]++
			}
		}
		// Score: lines sharing the most common width, weighted by that width.
		for width, n := range counts {
			if score := n*100 + width; score > bestScore {
				best, bestScore = d, score
			}
		}
	}
	return best
}

func nthIndex(s string, c byte, n int) int {
	for i := range len(s) {
		if s[i] == c {
			if n--; n == 0 {
				return i
			}
		}
	}
	return -1
}

func parseCSV(b []byte) (Grid, FileInfo, error) {
	text, enc, err := decode(b)
	if err != nil {
		return nil, FileInfo{}, err
	}
	d := detectDelimiter(text)
	r := csv.NewReader(strings.NewReader(text))
	r.Comma, r.FieldsPerRecord, r.LazyQuotes, r.ReuseRecord = d, -1, true, false
	var g Grid
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, FileInfo{}, fmt.Errorf("the file isn't valid CSV: %w", err)
		}
		if g, err = appendRow(g, rec); err != nil {
			return nil, FileInfo{}, err
		}
	}
	if len(g) == 0 {
		return nil, FileInfo{}, errors.New("the file has no rows")
	}
	return g, FileInfo{Encoding: enc, Delimiter: string(d)}, nil
}

func parseXLSX(b []byte) (Grid, error) {
	// Untrusted zips: cap how much may be unpacked.
	f, err := excelize.OpenReader(bytes.NewReader(b), excelize.Options{
		UnzipSizeLimit: 64 << 20, UnzipXMLSizeLimit: 32 << 20,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: the .xlsx file can't be read", ErrUnsupported)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("the workbook has no sheets")
	}
	rows, err := f.Rows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("the first sheet can't be read: %w", err)
	}
	defer rows.Close()
	var g Grid
	for rows.Next() {
		// Raw values, not Excel's display text: a date's display format may drop the day
		// ("Sep-26"), while its raw serial converts exactly; numbers come as plain decimals.
		rec, err := rows.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, fmt.Errorf("the first sheet can't be read: %w", err)
		}
		if g, err = appendRow(g, rec); err != nil {
			return nil, err
		}
	}
	if len(g) == 0 {
		return nil, errors.New("the first sheet is empty")
	}
	return g, nil
}

// appendRow trims and caps a record and appends it, skipping rows with no text.
func appendRow(g Grid, rec []string) (Grid, error) {
	if len(rec) > MaxColumns {
		rec = rec[:MaxColumns]
	}
	row := make([]string, len(rec))
	empty := true
	for i, c := range rec {
		c = strings.TrimSpace(strings.ReplaceAll(c, " ", " "))
		if utf8.RuneCountInString(c) > maxCell {
			c = string([]rune(c)[:maxCell])
		}
		row[i] = c
		empty = empty && c == ""
	}
	if empty {
		return g, nil
	}
	// Drop trailing empty cells so a stray delimiter doesn't add a column.
	for len(row) > 0 && row[len(row)-1] == "" {
		row = row[:len(row)-1]
	}
	if len(g) >= MaxRows {
		return nil, fmt.Errorf("the file has more than %d rows; split it into smaller files", MaxRows)
	}
	return append(g, row), nil
}
