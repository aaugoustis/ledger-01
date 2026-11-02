package uy

import (
	"os"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/importer"
)

func TestItauClassify(t *testing.T) {
	for _, c := range []struct {
		note     string
		role     importer.Role
		merchant string
	}{
		{"COMPRA  ALMACEN ROSA ", importer.RolePurchase, "almacen rosa"},
		{"PAGO FACTURAAGUAS SUR 000123", importer.RolePurchase, "aguas sur 000123"},
		{"Pago factura Luz Norte 004567", importer.RolePurchase, "luz norte 004567"},
		{"REDIVA 19210ALMACEN ROSA", importer.RoleDiscount, "almacen rosa"},
		{"REDIVA 17934PARRILLA CENTRO", importer.RoleDiscount, "parrilla centro"},
		{"Rediva 19.210 Mercado Sol", importer.RoleDiscount, "mercado sol"},
		{"DEVOLUCION PAY*TRANS*BUS", importer.RoleRefund, "pay*trans*bus"},
		{"Anulación  Kiosco Azul", importer.RoleRefund, "kiosco azul"},
		{"TRANSFERENCIA RECIBIDA", importer.RolePlain, "transferencia recibida"},
	} {
		role, merchant, _ := Itau.Classify(c.note)
		if role != c.role || merchant != c.merchant {
			t.Errorf("Classify(%q) = %v %q", c.note, role, merchant)
		}
	}
}

// Made-up statements: a double charge with its refund, and purchases with their REDIVA.
func TestItauPairs(t *testing.T) {
	rows := rowsOf(t, "refund.csv")
	got := importer.PairAdjustments(rows, Itau, func(int) bool { return false })
	if len(got.Refunded) != 1 || got.Refunded[2] != 1 || len(got.Merged) != 1 || got.Merged[4] != 3 {
		t.Fatalf("pairs = %+v", got)
	}
}

func TestDetectItau(t *testing.T) {
	banks := []importer.Bank{importer.Generic, Itau}
	for _, f := range []string{"rediva.csv", "refund.csv"} {
		if b := importer.DetectBank(rowsOf(t, f), banks); b.ID != Itau.ID {
			t.Errorf("%s: detected %s", f, b.ID)
		}
	}
	if b := importer.DetectBank(rowsOf(t, "northbank.csv"), banks); b.ID != importer.Generic.ID {
		t.Errorf("northbank.csv: detected %s", b.ID)
	}
}

func rowsOf(t *testing.T, name string) []importer.Row {
	t.Helper()
	b, err := os.ReadFile("../../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	g, _, err := importer.Parse(name, b)
	if err != nil {
		t.Fatal(err)
	}
	return importer.Apply(g, importer.GuessMapping(g), 2)
}

// Two meals at one place on one day, each with its REDIVA right after it.
func TestItauSameDayDiscounts(t *testing.T) {
	day, _ := time.Parse(time.DateOnly, "2026-10-02")
	rows := []importer.Row{
		{Date: day, Amount: -895, Note: "COMPRA PARRILLA CENTRO"},
		{Date: day, Amount: 166, Note: "REDIVA 17934PARRILLA CENTRO"},
		{Date: day, Amount: -2250, Note: "COMPRA PARRILLA CENTRO"},
		{Date: day, Amount: 66, Note: "REDIVA 17934PARRILLA CENTRO"},
	}
	got := importer.PairAdjustments(rows, Itau, func(int) bool { return false })
	if got.Merged[1] != 0 || got.Merged[3] != 2 {
		t.Fatalf("merged = %v", got.Merged)
	}
}

// Bill payments get their REDIVA too, listed just before the payment.
func TestItauBillPaymentDiscount(t *testing.T) {
	day, _ := time.Parse(time.DateOnly, "2026-10-21")
	rows := []importer.Row{
		{Date: day, Amount: 6609, Note: "REDIVA 19210AGUAS SUR 000123"},
		{Date: day, Amount: -435700, Note: "PAGO FACTURAAGUAS SUR 000123"},
		{Date: day, Amount: 4260, Note: "REDIVA 19210LUZ NORTE 004567"},
		{Date: day, Amount: -259900, Note: "PAGO FACTURALUZ NORTE 004567"},
	}
	got := importer.PairAdjustments(rows, Itau, func(int) bool { return false })
	if len(got.Merged) != 2 || got.Merged[0] != 1 || got.Merged[2] != 3 {
		t.Fatalf("merged = %v", got.Merged)
	}
}
