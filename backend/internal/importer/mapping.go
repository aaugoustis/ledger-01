package importer

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Sign says which amounts are money going out.
type Sign int

const (
	// NegativeIsExpense: "-12.30" is an expense (most bank accounts).
	NegativeIsExpense Sign = iota + 1
	// PositiveIsExpense: "12.30" is an expense (most card statements).
	PositiveIsExpense
)

// Mapping says which columns hold what. Column indexes are 0-based; -1 means not mapped. Amounts
// come from Amount, or from Debit (money out) and/or Credit (money in).
// The JSON form is stored as a wallet's saved mapping.
type Mapping struct {
	// Rows above the table, skipped (a bank's logo, account details, …).
	Skip       int    `json:"skip_rows"`
	HasHeader  bool   `json:"has_header"`
	Date       int    `json:"date"`
	DateFormat string `json:"date_format"`
	Amount     int    `json:"amount"`
	Debit      int    `json:"debit"`
	Credit     int    `json:"credit"`
	Sign       Sign   `json:"sign"`
	// '.' or ','; the other one is taken as a thousands separator.
	Decimal  byte `json:"decimal"`
	Note     int  `json:"note"`
	Category int  `json:"category"`
	// How the bank words purchases, discounts and refunds (a banks ID, e.g. "uy-itau"). Empty
	// means detect it from the file.
	Bank string `json:"bank"`
}

// FieldError is a mapping problem with a specific field, named as in the API.
type FieldError struct{ Field, Msg string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Msg }

// Header returns the row naming the columns, or nil when the mapping has none.
func (m Mapping) Header(g Grid) []string {
	if !m.HasHeader || m.Skip < 0 || m.Skip >= len(g) {
		return nil
	}
	return g[m.Skip]
}

// Validate checks the mapping against a file with cols columns and rows rows.
func (m Mapping) Validate(cols, rows int) error {
	if m.Skip < 0 || (rows > 0 && m.Skip >= rows) {
		return &FieldError{"skip_rows", "the table has to start within the file"}
	}
	col := func(field string, c int, required bool) error {
		if c == -1 && !required {
			return nil
		}
		if c < 0 || c >= cols {
			return &FieldError{field, "choose a column"}
		}
		return nil
	}
	if err := col("date_column", m.Date, true); err != nil {
		return err
	}
	if !validDateFormat(m.DateFormat) {
		return &FieldError{"date_format", "choose a date format"}
	}
	if m.Amount == -1 && m.Debit == -1 && m.Credit == -1 {
		return &FieldError{"amount_column", "choose the amount column, or the debit and credit columns"}
	}
	for _, c := range []struct {
		field string
		col   int
	}{{"amount_column", m.Amount}, {"debit_column", m.Debit}, {"credit_column", m.Credit}, {"note_column", m.Note}, {"category_column", m.Category}} {
		if err := col(c.field, c.col, false); err != nil {
			return err
		}
	}
	if m.Amount != -1 && (m.Debit != -1 || m.Credit != -1) {
		return &FieldError{"amount_column", "use one amount column or debit and credit columns, not both"}
	}
	if m.Amount != -1 && m.Sign != NegativeIsExpense && m.Sign != PositiveIsExpense {
		return &FieldError{"sign", "choose which amounts are expenses"}
	}
	if m.Decimal != '.' && m.Decimal != ',' {
		return &FieldError{"decimal_separator", "choose the decimal separator"}
	}
	return nil
}

// Row is one data row after mapping. Amount is signed in the wallet's minor units: negative is
// money out (an expense). Err says why the row can't be imported.
type Row struct {
	Index    int // 0-based among data rows (after the skipped rows and the header)
	Cells    []string
	Date     time.Time
	Amount   int64
	Note     string
	Category string
	Err      string
}

// Apply maps every data row of g. minorUnits is the wallet currency's decimal places.
func Apply(g Grid, m Mapping, minorUnits int) []Row {
	start := max(m.Skip, 0)
	if m.HasHeader {
		start++
	}
	out := make([]Row, 0, max(0, len(g)-start))
	for i := start; i < len(g); i++ {
		r := Row{Index: i - start, Cells: g[i], Note: g.Cell(i, m.Note), Category: g.Cell(i, m.Category)}
		if m.Note == -1 {
			r.Note = ""
		}
		if m.Category == -1 {
			r.Category = ""
		}
		r.Note = strings.Join(strings.Fields(r.Note), " ")
		var err error
		if r.Date, err = ParseDate(g.Cell(i, m.Date), m.DateFormat); err != nil {
			r.Err = err.Error()
		} else if r.Amount, err = rowAmount(g, i, m, minorUnits); err != nil {
			r.Err = err.Error()
		}
		out = append(out, r)
	}
	return out
}

func rowAmount(g Grid, i int, m Mapping, minorUnits int) (int64, error) {
	if m.Amount != -1 {
		v, err := ParseAmount(g.Cell(i, m.Amount), m.Decimal, minorUnits)
		if err != nil {
			return 0, err
		}
		if v == 0 {
			return 0, errors.New("the amount is zero")
		}
		if m.Sign == PositiveIsExpense {
			v = -v
		}
		return v, nil
	}
	// Debit and credit columns: either may be empty; amounts count by column, whatever their sign.
	var total int64
	for _, c := range []struct {
		col  int
		sign int64
	}{{m.Debit, -1}, {m.Credit, 1}} {
		if c.col == -1 || g.Cell(i, c.col) == "" {
			continue
		}
		v, err := ParseAmount(g.Cell(i, c.col), m.Decimal, minorUnits)
		if err != nil {
			return 0, err
		}
		total += c.sign * abs(v)
	}
	if total == 0 {
		return 0, errors.New("there is no amount")
	}
	return total, nil
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// maxAmount mirrors the ledger's cap on a single amount (in minor units).
const maxAmount = 1_000_000_000_000_000

// ParseAmount reads an amount such as "-1.234,56", "(12.30)", "12.30-", "$ 45" or "1 000,00"
// into signed minor units. decimal is the decimal separator ('.' or ','); the other one, spaces
// and apostrophes are thousands separators. Currency symbols and codes are ignored.
func ParseAmount(s string, decimal byte, minorUnits int) (int64, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return 0, errors.New("there is no amount")
	}
	neg := false
	if strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")") {
		neg, raw = true, raw[1:len(raw)-1]
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == rune(decimal):
			b.WriteByte('.')
		case r == '-' || r == '−':
			neg = !neg
		case r == '+', r == '.', r == ',', r == '\'', unicode.IsSpace(r), unicode.IsLetter(r), unicode.Is(unicode.Sc, r):
			// thousands separators, signs, currency symbols and codes
		default:
			return 0, fmt.Errorf("%q isn't an amount", s)
		}
	}
	num := b.String()
	if num == "" || strings.Count(num, ".") > 1 || strings.HasPrefix(num, ".") && len(num) == 1 {
		return 0, fmt.Errorf("%q isn't an amount", s)
	}
	r, ok := new(big.Rat).SetString(num)
	if !ok {
		return 0, fmt.Errorf("%q isn't an amount", s)
	}
	r.Mul(r, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(minorUnits)), nil)))
	if !r.IsInt() {
		return 0, fmt.Errorf("%q has more than %d decimals", s, minorUnits)
	}
	if r.Num().CmpAbs(big.NewInt(maxAmount)) > 0 {
		return 0, fmt.Errorf("%q is too large", s)
	}
	v := r.Num().Int64()
	if neg {
		v = -v
	}
	return v, nil
}

// DateFormats lists the supported date formats, in the order the UI offers them.
var DateFormats = []string{
	"yyyy-mm-dd", "dd/mm/yyyy", "mm/dd/yyyy", "dd-mm-yyyy", "dd.mm.yyyy", "yyyy/mm/dd", "dd/mm/yy", "mm/dd/yy",
}

func validDateFormat(f string) bool {
	for _, d := range DateFormats {
		if d == f {
			return true
		}
	}
	return false
}

var (
	datePartsRe = regexp.MustCompile(`^(\d{1,4})[-/.](\d{1,2})[-/.](\d{1,4})$`)
	serialRe    = regexp.MustCompile(`^\d{5}(\.\d+)?$`)
)

// ParseDate reads a date in one of DateFormats. Separators may be any of - / . and day and
// month may have one digit; a time after the date is ignored. A spreadsheet date serial (days
// since 1899-12-30, as XLSX stores dates) is accepted in any format.
func ParseDate(s, format string) (time.Time, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return time.Time{}, errors.New("there is no date")
	}
	if serialRe.MatchString(raw) {
		n, _ := strconv.ParseFloat(raw, 64)
		if n >= 20_000 && n <= 80_000 { // 1954 to 2119
			return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(n)), nil
		}
	}
	date, _, _ := strings.Cut(strings.Replace(raw, "T", " ", 1), " ")
	m := datePartsRe.FindStringSubmatch(date)
	if m == nil {
		return time.Time{}, fmt.Errorf("%q isn't a date", s)
	}
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[2])
	c, _ := strconv.Atoi(m[3])
	var y, mo, d int
	switch format {
	case "yyyy-mm-dd", "yyyy/mm/dd":
		y, mo, d = a, b, c
	case "dd/mm/yyyy", "dd-mm-yyyy", "dd.mm.yyyy", "dd/mm/yy":
		d, mo, y = a, b, c
	case "mm/dd/yyyy", "mm/dd/yy":
		mo, d, y = a, b, c
	default:
		return time.Time{}, fmt.Errorf("unknown date format %q", format)
	}
	if y < 100 && strings.HasSuffix(format, "/yy") {
		y += 2000
	}
	if y < 1900 || y > 2200 {
		return time.Time{}, fmt.Errorf("%q isn't a %s date", s, format)
	}
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	if t.Year() != y || int(t.Month()) != mo || t.Day() != d {
		return time.Time{}, fmt.Errorf("%q isn't a %s date", s, format)
	}
	return t, nil
}
