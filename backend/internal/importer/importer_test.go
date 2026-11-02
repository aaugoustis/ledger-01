package importer

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func summary(rows []Row) string {
	var out []string
	for _, r := range rows {
		if r.Err != "" {
			out = append(out, "ERR("+r.Err+")")
			continue
		}
		out = append(out, r.Date.Format(time.DateOnly)+" "+formatMinor(r.Amount)+" "+r.Note)
	}
	return strings.Join(out, " | ")
}

func formatMinor(v int64) string {
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	return sign + itoa(v/100) + "." + pad2(v%100)
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for ; v > 0; v /= 10 {
		b = append([]byte{byte('0' + v%10)}, b...)
	}
	return string(b)
}

func pad2(v int64) string {
	if v < 10 {
		return "0" + itoa(v)
	}
	return itoa(v)
}

func TestNorthbankCSV(t *testing.T) {
	g, info, err := Parse("statement.csv", fixture(t, "northbank.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "csv" || info.Encoding != "UTF-8" || info.Delimiter != "," || len(g) != 7 || g.Columns() != 4 {
		t.Fatalf("info %+v rows %d cols %d", info, len(g), g.Columns())
	}
	if g[0][0] != "Date" {
		t.Fatalf("BOM not stripped: %q", g[0][0])
	}
	m := GuessMapping(g)
	want := Mapping{HasHeader: true, Date: 0, DateFormat: "yyyy-mm-dd", Amount: 2, Debit: -1, Credit: -1, Sign: NegativeIsExpense, Decimal: '.', Note: 1, Category: -1}
	if m != want {
		t.Fatalf("guess = %+v\nwant    %+v", m, want)
	}
	if err := m.Validate(g.Columns(), len(g)); err != nil {
		t.Fatal(err)
	}
	got := summary(Apply(g, m, 2))
	if want := "2026-09-01 1500.00 Opening deposit | 2026-09-03 -45.20 Corner Grocer | 2026-09-03 -45.20 Corner Grocer | " +
		"2026-09-05 -20.00 City Transit Card | 2026-09-10 2400.00 Acme Payroll | 2026-09-12 -1200.00 Riverside Rent"; got != want {
		t.Fatalf("rows:\n got  %s\n want %s", got, want)
	}
}

func TestBancoSurLatin1DebitCredit(t *testing.T) {
	g, info, err := Parse("movimientos.csv", fixture(t, "bancosur.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Encoding != "Windows-1252" || info.Delimiter != ";" {
		t.Fatalf("info %+v", info)
	}
	if g[0][2] != "Débito" || g[1][1] != "Depósito inicial" {
		t.Fatalf("decoding: %q %q", g[0][2], g[1][1])
	}
	m := GuessMapping(g)
	want := Mapping{HasHeader: true, Date: 0, DateFormat: "dd/mm/yyyy", Amount: -1, Debit: 2, Credit: 3, Sign: NegativeIsExpense, Decimal: ',', Note: 1, Category: -1}
	if m != want {
		t.Fatalf("guess = %+v\nwant    %+v", m, want)
	}
	got := summary(Apply(g, m, 2))
	if want := "2026-09-01 10000.00 Depósito inicial | 2026-09-03 -1250.50 Almacén Central | 2026-09-04 -320.00 Café Plaza | " +
		"2026-09-15 45000.00 Sueldo Empresa Ficticia"; got != want {
		t.Fatalf("rows:\n got  %s\n want %s", got, want)
	}
}

func TestCardStatementTabs(t *testing.T) {
	g, info, err := Parse("card.txt", fixture(t, "card.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Delimiter != "\t" {
		t.Fatalf("delimiter %q", info.Delimiter)
	}
	m := GuessMapping(g)
	if m.Date != 0 || m.DateFormat != "mm/dd/yyyy" || m.Note != 1 || m.Category != 2 || m.Amount != 3 {
		t.Fatalf("guess = %+v", m)
	}
	// Card statements list charges as positive amounts.
	m.Sign = PositiveIsExpense
	rows := Apply(g, m, 2)
	if got := summary(rows); got != "2026-09-02 -8.50 Sunrise Bakery | 2026-09-07 -52.10 Metro Fuel | 2026-09-09 -61.75 Corner Grocer | 2026-09-14 122.35 Payment - thank you" {
		t.Fatalf("rows: %s", got)
	}
	if rows[0].Category != "Food & Drink" || rows[3].Category != "" {
		t.Fatalf("categories %q %q", rows[0].Category, rows[3].Category)
	}
}

func TestXLSX(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for i, row := range [][]any{
		{"Fecha", "Descripción", "Importe"},
		{time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), "Librería Norte", -18.5},
		{46296, "Transferencia recibida", 250},
		{"2026-10-02", "Farmacia Sol", "-7,25"},
	} {
		for j, v := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	g, info, err := Parse("export.xlsx", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "xlsx" || len(g) != 4 {
		t.Fatalf("info %+v rows %v", info, g)
	}
	m := Mapping{HasHeader: true, Date: 0, DateFormat: "mm/dd/yy", Amount: 2, Debit: -1, Credit: -1, Sign: NegativeIsExpense, Decimal: '.', Note: 1, Category: -1}
	rows := Apply(g, m, 2)
	// Date cells and serials both read as dates, and a text amount with a comma is an error
	// under a dot decimal separator.
	if rows[0].Err != "" || rows[0].Date.Format(time.DateOnly) != "2026-09-01" || rows[0].Amount != -1850 {
		t.Fatalf("row 0 %+v (cells %q)", rows[0], g[1])
	}
	if rows[1].Err != "" || rows[1].Date.Format(time.DateOnly) != "2026-10-01" || rows[1].Amount != 25000 {
		t.Fatalf("row 1 %+v", rows[1])
	}
	if rows[2].Err == "" {
		t.Fatalf("row 2 should fail: %+v", rows[2])
	}
}

func TestParseRejects(t *testing.T) {
	for name, c := range map[string]struct {
		file    string
		content []byte
	}{
		"empty":           {"a.csv", nil},
		"legacy xls":      {"old.xls", []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}},
		"xls by name":     {"old.xls", []byte("a,b\n1,2\n")},
		"damaged xlsx":    {"x.xlsx", []byte("PK\x03\x04not really a zip")},
		"too large":       {"big.csv", bytes.Repeat([]byte("a"), MaxFileBytes+1)},
		"too many rows":   {"rows.csv", bytes.Repeat([]byte("2026-01-01,1\n"), MaxRows+1)},
		"only blank rows": {"blank.csv", []byte(" , \n\n , \n")},
	} {
		if _, _, err := Parse(c.file, c.content); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, _, err := Parse("old.xls", []byte{0xD0, 0xCF, 0x11, 0xE0}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("xls error = %v", err)
	}
}

func TestParseAmount(t *testing.T) {
	for _, c := range []struct {
		in    string
		dec   byte
		units int
		want  int64
		ok    bool
	}{
		{"12.30", '.', 2, 1230, true},
		{"-1,234.56", '.', 2, -123456, true},
		{"1.234,56", ',', 2, 123456, true},
		{"1 000,00", ',', 2, 100000, true},
		{"1'000.50", '.', 2, 100050, true},
		{"(12.30)", '.', 2, -1230, true},
		{"12.30-", '.', 2, -1230, true},
		{"−5.00", '.', 2, -500, true},
		{"$ 45", '.', 2, 4500, true},
		{"USD 10.00", '.', 2, 1000, true},
		{"€12,5", ',', 2, 1250, true},
		{"1,200", '.', 0, 1200, true},
		{"12.345", '.', 2, 0, false},
		{"12.30.4", '.', 2, 0, false},
		{"abc", '.', 2, 0, false},
		{"", '.', 2, 0, false},
		{"12#30", '.', 2, 0, false},
		{"99999999999999999999", '.', 2, 0, false},
	} {
		got, err := ParseAmount(c.in, c.dec, c.units)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("ParseAmount(%q, %q, %d) = %d, %v; want %d ok=%v", c.in, c.dec, c.units, got, err, c.want, c.ok)
		}
	}
}

func TestParseDate(t *testing.T) {
	for _, c := range []struct {
		in, format, want string
	}{
		{"2026-10-03", "yyyy-mm-dd", "2026-10-03"},
		{"2026-10-03T14:22:00", "yyyy-mm-dd", "2026-10-03"},
		{"2026/1/3", "yyyy/mm/dd", "2026-01-03"},
		{"03/10/2026", "dd/mm/yyyy", "2026-10-03"},
		{"3.10.2026", "dd.mm.yyyy", "2026-10-03"},
		{"10/03/2026", "mm/dd/yyyy", "2026-10-03"},
		{"03/10/26", "dd/mm/yy", "2026-10-03"},
		{"10/3/26 0:00", "mm/dd/yy", "2026-10-03"},
		{"46296", "dd/mm/yyyy", "2026-10-01"},
		{"31/02/2026", "dd/mm/yyyy", ""},
		{"13/13/2026", "mm/dd/yyyy", ""},
		{"yesterday", "yyyy-mm-dd", ""},
		{"", "yyyy-mm-dd", ""},
		{"2026-10-03", "unknown", ""},
	} {
		got, err := ParseDate(c.in, c.format)
		if c.want == "" {
			if err == nil {
				t.Errorf("ParseDate(%q, %q) = %s; want an error", c.in, c.format, got.Format(time.DateOnly))
			}
			continue
		}
		if err != nil || got.Format(time.DateOnly) != c.want {
			t.Errorf("ParseDate(%q, %q) = %s, %v; want %s", c.in, c.format, got.Format(time.DateOnly), err, c.want)
		}
	}
}

func TestValidateMapping(t *testing.T) {
	ok := Mapping{Date: 0, DateFormat: "yyyy-mm-dd", Amount: 1, Debit: -1, Credit: -1, Sign: NegativeIsExpense, Decimal: '.', Note: -1, Category: -1}
	if err := ok.Validate(3, 0); err != nil {
		t.Fatal(err)
	}
	for field, m := range map[string]Mapping{
		"date_column":       {Date: -1, DateFormat: "yyyy-mm-dd", Amount: 1, Debit: -1, Credit: -1, Sign: 1, Decimal: '.', Note: -1, Category: -1},
		"date_format":       {Date: 0, DateFormat: "dd mon yyyy", Amount: 1, Debit: -1, Credit: -1, Sign: 1, Decimal: '.', Note: -1, Category: -1},
		"amount_column":     {Date: 0, DateFormat: "yyyy-mm-dd", Amount: -1, Debit: -1, Credit: -1, Sign: 1, Decimal: '.', Note: -1, Category: -1},
		"note_column":       {Date: 0, DateFormat: "yyyy-mm-dd", Amount: 1, Debit: -1, Credit: -1, Sign: 1, Decimal: '.', Note: 7, Category: -1},
		"sign":              {Date: 0, DateFormat: "yyyy-mm-dd", Amount: 1, Debit: -1, Credit: -1, Decimal: '.', Note: -1, Category: -1},
		"decimal_separator": {Date: 0, DateFormat: "yyyy-mm-dd", Amount: 1, Debit: -1, Credit: -1, Sign: 1, Decimal: ' ', Note: -1, Category: -1},
	} {
		var fe *FieldError
		if err := m.Validate(3, 0); !errors.As(err, &fe) || fe.Field != field {
			t.Errorf("%s: err = %v", field, err)
		}
	}
	both := ok
	both.Debit = 2
	var fe *FieldError
	if err := both.Validate(3, 0); !errors.As(err, &fe) || fe.Field != "amount_column" {
		t.Errorf("amount and debit: err = %v", err)
	}
}

// Uploads are untrusted: parsing, guessing and applying a mapping must never panic.
func FuzzParse(f *testing.F) {
	f.Add("a.csv", []byte("Date,Description,Amount\n2026-01-01,Shop,-1.00\n"))
	f.Add("b.csv", []byte("Fecha;Importe\n01/02/2026;\"1.234,5\"\n"))
	f.Add("c.xlsx", []byte("PK\x03\x04\x14\x00"))
	f.Add("d.txt", []byte("\xff\xfeD\x00a\x00t\x00e\x00"))
	f.Fuzz(func(t *testing.T, name string, content []byte) {
		g, _, err := Parse(name, content)
		if err != nil {
			return
		}
		m := GuessMapping(g)
		_ = m.Validate(g.Columns(), len(g))
		_ = Apply(g, m, 2)
	})
}

// A made-up statement with the bank's name, the holder and account details above the table.
func TestDecoratedStatementSkipsTheTop(t *testing.T) {
	g, _, err := Parse("estado.csv", fixture(t, "decorated.csv"))
	if err != nil {
		t.Fatal(err)
	}
	m := GuessMapping(g)
	if m.Skip != 5 || !m.HasHeader { // blank rows are dropped when parsing
		t.Fatalf("table start = skip %d header %v, want 5 true", m.Skip, m.HasHeader)
	}
	if m.Date != 0 || m.DateFormat != "dd/mm/yyyy" || m.Debit != 3 || m.Credit != 4 || m.Note != 1 || m.Decimal != ',' {
		t.Fatalf("guess = %+v", m)
	}
	if h := m.Header(g); len(h) == 0 || h[0] != "Fecha" {
		t.Fatalf("header = %v", h)
	}
	rows := Apply(g, m, 2)
	if len(rows) != 3 || rows[0].Index != 0 || rows[0].Err != "" || rows[0].Amount != -4520 || rows[1].Amount != 150000 {
		t.Fatalf("rows = %+v", rows)
	}
	if err := m.Validate(g.Columns(), len(g)); err != nil {
		t.Fatal(err)
	}
	bad := m
	bad.Skip = len(g)
	var fe *FieldError
	if err := bad.Validate(g.Columns(), len(g)); !errors.As(err, &fe) || fe.Field != "skip_rows" {
		t.Fatalf("skip past the end: %v", err)
	}
}

// Without a header row, the table starts at the first rows holding a date and an amount; XLSX
// date serials count as dates.
func TestTableStartWithoutHeader(t *testing.T) {
	g := Grid{
		{"Northbank card statement"},
		{"Card ending 4242", ""},
		{"46268", "Corner Grocer", "45.20"},
		{"46270", "City Transit Card", "20.00"},
	}
	m := GuessMapping(g)
	if m.Skip != 2 || m.HasHeader || m.Date != 0 || m.Amount != 2 {
		t.Fatalf("guess = %+v", m)
	}
}
