package importer

import (
	"regexp"
	"strconv"
	"strings"
)

// Header words that identify a column, in English, Spanish and Portuguese. Matched against the
// lower-cased header without accents; the first kind whose word appears wins.
var headerWords = []struct {
	kind  string
	words []string
}{
	{"skip", []string{"balance", "saldo"}},
	{"category", []string{"category", "categoria"}},
	{"debit", []string{"debit", "debito", "debe", "cargo", "withdrawal", "money out", "salida", "egreso"}},
	{"credit", []string{"credit", "credito", "haber", "abono", "deposit", "money in", "entrada", "ingreso"}},
	{"date", []string{"date", "fecha", "data", "posted"}},
	{"amount", []string{"amount", "importe", "monto", "valor", "value", "cantidad", "sum"}},
	{"note", []string{"description", "descripcion", "concepto", "detail", "memo", "narrative", "payee", "merchant", "movimiento", "referencia", "reference", "historico"}},
}

var unaccent = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "ã", "a", "õ", "o", "ç", "c", "â", "a", "ê", "e", "ô", "o")

func headerKind(h string) string {
	h = unaccent.Replace(strings.ToLower(strings.TrimSpace(h)))
	for _, hw := range headerWords {
		for _, w := range hw.words {
			if strings.Contains(h, w) {
				return hw.kind
			}
		}
	}
	return ""
}

// GuessMapping proposes a mapping for a file: where the table starts (banks often put a logo and
// account details above it), columns from header names when there's a header row, otherwise from
// the values; the date format and decimal separator from the values. Anything it can't tell is
// left unmapped (-1).
func GuessMapping(file Grid) Mapping {
	m := Mapping{Date: -1, Amount: -1, Debit: -1, Credit: -1, Note: -1, Category: -1, Sign: NegativeIsExpense, Decimal: '.'}
	if len(file) == 0 {
		return m
	}
	m.Skip, m.HasHeader = guessTableStart(file)
	g := file[m.Skip:]
	if m.HasHeader {
		for c, h := range g[0] {
			slot := map[string]*int{"date": &m.Date, "amount": &m.Amount, "debit": &m.Debit, "credit": &m.Credit, "note": &m.Note, "category": &m.Category}[headerKind(h)]
			if slot != nil && *slot == -1 {
				*slot = c
			}
		}
	}
	data := g
	if m.HasHeader {
		data = g[1:]
	}
	if m.Date == -1 {
		m.Date = bestColumn(data, looksLikeDate, -1)
	}
	if m.Amount == -1 && m.Debit == -1 && m.Credit == -1 {
		m.Amount = bestColumn(data, looksLikeAmount, m.Date)
	}
	if m.Date != -1 {
		m.DateFormat = guessDateFormat(column(data, m.Date))
	}
	if m.DateFormat == "" {
		m.DateFormat = DateFormats[0]
	}
	var amounts []string
	for _, c := range []int{m.Amount, m.Debit, m.Credit} {
		if c != -1 {
			amounts = append(amounts, column(data, c)...)
		}
	}
	m.Decimal = guessDecimal(amounts)
	if m.Debit != -1 || m.Credit != -1 {
		m.Amount = -1
	}
	return m
}

func column(g Grid, c int) []string {
	out := make([]string, 0, len(g))
	for i := range g {
		if v := g.Cell(i, c); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// tableScan is how far down the file the table is looked for.
const tableScan = 30

// guessTableStart finds the first row of the table: a header row that names a date column and is
// followed by a row with a date, else the first of two rows that both hold a date and an amount
// (with the text row just above it as the header). Without either, the table starts at the top.
func guessTableStart(g Grid) (skip int, header bool) {
	limit := min(len(g), tableScan)
	for r := 0; r+1 < len(g) && r < limit; r++ {
		if isHeaderRow(g[r]) && rowHasDate(g[r+1]) {
			return r, true
		}
	}
	for r := 0; r < limit; r++ {
		if isDataRow(g[r]) && (r+1 == len(g) || isDataRow(g[r+1])) {
			if r > 0 && isTextRow(g[r-1]) {
				return r - 1, true
			}
			return r, false
		}
	}
	return 0, looksLikeHeader(g)
}

// isTextRow: at least two filled cells, none of them a date or an amount.
func isTextRow(row []string) bool {
	filled := 0
	for _, c := range row {
		if strings.TrimSpace(c) == "" {
			continue
		}
		if looksLikeDate(c) || looksLikeAmount(c) {
			return false
		}
		filled++
	}
	return filled >= 2
}

// isHeaderRow: a text row with a cell that names a date column ("Fecha", "Date", …).
func isHeaderRow(row []string) bool {
	if !isTextRow(row) {
		return false
	}
	for _, c := range row {
		if headerKind(c) == "date" {
			return true
		}
	}
	return false
}

func rowHasDate(row []string) bool {
	for _, c := range row {
		if looksLikeDate(c) {
			return true
		}
	}
	return false
}

// isDataRow: a date in one cell and an amount in another.
func isDataRow(row []string) bool {
	for i, c := range row {
		if !looksLikeDate(c) {
			continue
		}
		for j, d := range row {
			if j != i && !looksLikeDate(d) && looksLikeAmount(d) {
				return true
			}
		}
	}
	return false
}

// looksLikeDate: a date in one of DateFormats, or a spreadsheet date serial between 1982 and 2064.
func looksLikeDate(s string) bool {
	s = strings.TrimSpace(s)
	if serialRe.MatchString(s) {
		v, err := strconv.ParseFloat(s, 64)
		return err == nil && v >= 30000 && v < 60000
	}
	return guessDateFormat([]string{s}) != ""
}

// looksLikeHeader: the first row has text cells and no date or amount in them, while the second
// row has some.
func looksLikeHeader(g Grid) bool {
	values := func(row []string) int {
		n := 0
		for _, c := range row {
			if guessDateFormat([]string{c}) != "" || looksLikeAmount(c) {
				n++
			}
		}
		return n
	}
	if len(g) < 2 {
		return false
	}
	return values(g[0]) == 0 && values(g[1]) > 0
}

// bestColumn returns the column (other than skip) where ok holds for the most values (at least
// half of the rows), or -1.
func bestColumn(g Grid, ok func(string) bool, skip int) int {
	best, bestN := -1, 0
	for c := range g.Columns() {
		if c == skip {
			continue
		}
		n := 0
		for _, v := range column(g, c) {
			if ok(v) {
				n++
			}
		}
		if n > bestN && n*2 >= len(g) {
			best, bestN = c, n
		}
	}
	return best
}

var amountRe = regexp.MustCompile(`^[(+\-\x{2212}]?\s*[^\d\s]{0,3}\s*\d[\d.,'\s]*\)?-?(\s*[A-Z]{3})?$`)

func looksLikeAmount(s string) bool {
	return amountRe.MatchString(strings.TrimSpace(s)) && strings.ContainsAny(s, "0123456789")
}

// guessDateFormat returns the format that reads the most of values, preferring day-first
// formats when the first part goes above 12, or "" when none reads any.
func guessDateFormat(values []string) string {
	best, bestN := "", 0
	for _, f := range DateFormats {
		n := 0
		for _, v := range values[:min(len(values), 200)] {
			if serialRe.MatchString(strings.TrimSpace(v)) {
				continue
			}
			if _, err := ParseDate(v, f); err == nil {
				n++
			}
		}
		if n > bestN {
			best, bestN = f, n
		}
	}
	return best
}

var (
	commaDecimalRe = regexp.MustCompile(`,\d{1,2}\)?-?$|\.\d{3},`)
	dotDecimalRe   = regexp.MustCompile(`\.\d{1,2}\)?-?$|,\d{3}\.`)
)

// guessDecimal votes on the decimal separator from amounts such as "1.234,56" or "12.30".
func guessDecimal(values []string) byte {
	comma, dot := 0, 0
	for _, v := range values[:min(len(values), 200)] {
		v = strings.TrimSpace(strings.TrimRight(v, " ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
		if commaDecimalRe.MatchString(v) {
			comma++
		}
		if dotDecimalRe.MatchString(v) {
			dot++
		}
	}
	if comma > dot {
		return ','
	}
	return '.'
}
