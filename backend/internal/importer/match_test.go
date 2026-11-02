package importer

import (
	"testing"
	"time"
)

func on(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestFingerprints(t *testing.T) {
	rows := []Row{
		{Date: on("2026-10-01"), Amount: -350, Note: "CAFE LUNA"},
		{Date: on("2026-10-01"), Amount: -350, Note: "cafe  luna"}, // the same row again: a second coffee
		{Err: "bad date"},
		{Date: on("2026-10-02"), Amount: -350, Note: "CAFE LUNA"},
	}
	fp := Fingerprints(rows)
	if len(fp[0]) != 16 || fp[0] == fp[1] || fp[0] == fp[3] || fp[2] != "" {
		t.Fatalf("fingerprints = %q", fp)
	}
	// Stable: the same row in another file (same ordinal) has the same fingerprint.
	if again := Fingerprints(rows[:1]); again[0] != fp[0] {
		t.Fatalf("not stable: %s vs %s", again[0], fp[0])
	}
}

// A made-up bank, so these tests show the core knows no real bank's wording.
var northbank = Bank{
	ID: "zz-northbank", Name: "Northbank", Country: "ZZ",
	Patterns: []Pattern{
		NewPattern(RolePurchase, `^card\s+(?P<merchant>.+?)\s*$`, 0),
		NewPattern(RoleDiscount, `^cashback\s+(?P<merchant>.+?)\s*$`, 0),
		NewPattern(RoleRefund, `^refund\s+(?P<merchant>.+?)\s*$`, 0),
	},
}

func TestBankClassify(t *testing.T) {
	for _, c := range []struct {
		note     string
		role     Role
		merchant string
	}{
		{"CARD  Corner Grocer ", RolePurchase, "corner grocer"},
		{"Cashback Corner Grocer", RoleDiscount, "corner grocer"},
		{"REFUND City Cabs", RoleRefund, "city cabs"},
		{"Salary  Acme", RolePlain, "salary acme"},
	} {
		role, merchant, _ := northbank.Classify(c.note)
		if role != c.role || merchant != c.merchant {
			t.Errorf("Classify(%q) = %v %q", c.note, role, merchant)
		}
	}
	if _, _, w, _ := northbank.Adjustment("refund city cabs"); w != DefaultRefundWindow {
		t.Fatalf("refund window = %v", w)
	}
	if northbank.MaxWindow() != DefaultRefundWindow || Generic.MaxWindow() != 0 {
		t.Fatal("MaxWindow")
	}
	// With a purchase pattern, plain rows aren't purchases; without one, they are.
	if _, ok := northbank.Purchase("Corner Grocer"); ok {
		t.Fatal("plain row taken as a purchase")
	}
	noPrefix := Bank{ID: "zz-plain", Patterns: []Pattern{NewPattern(RoleRefund, `^refund\s+(?P<merchant>.+)`, 0)}}
	if m, ok := noPrefix.Purchase("Corner Grocer 0042"); !ok || m != "corner grocer 0042" {
		t.Fatalf("Purchase without a pattern = %q %v", m, ok)
	}
	if !SameMerchant("corner grocer", "corner grocers") || SameMerchant("cor", "corner") || SameMerchant("corner", "city") {
		t.Fatal("SameMerchant")
	}
}

func TestNewPatternNeedsAMerchant(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("no panic")
		}
	}()
	NewPattern(RoleRefund, `^refund .+`, 0)
}

func TestPairAdjustments(t *testing.T) {
	rows := []Row{
		{Date: on("2026-10-01"), Amount: -50000, Note: "CARD CORNER GROCER"},
		{Date: on("2026-10-01"), Amount: -12000, Note: "CARD NORTH PHARMACY"},
		{Date: on("2026-10-01"), Amount: 1639, Note: "CASHBACK CORNER GROCER"},
		{Date: on("2026-10-04"), Amount: -30000, Note: "CARD CORNER GROCER"},
		{Date: on("2026-10-05"), Amount: 983, Note: "CASHBACK CORNER GROCER"},    // the closer purchase: row 3
		{Date: on("2026-10-20"), Amount: 500, Note: "CASHBACK NORTH PHARMACY"},   // too late for a discount
		{Date: on("2026-10-06"), Amount: 99999, Note: "CASHBACK NORTH PHARMACY"}, // bigger than the purchase
		// Charged twice, refunded once: the refund cancels the second charge.
		{Date: on("2026-10-02"), Amount: -740, Note: "CARD CITY CABS"},
		{Date: on("2026-10-02"), Amount: -740, Note: "CARD CITY CABS"},
		{Date: on("2026-10-02"), Amount: 740, Note: "REFUND CITY CABS"},
		// A partial refund lowers its purchase like a discount.
		{Date: on("2026-10-03"), Amount: -8000, Note: "CARD MOON SHOP"},
		{Date: on("2026-10-15"), Amount: 2000, Note: "REFUND MOON SHOP"},
	}
	none := func(int) bool { return false }
	got := PairAdjustments(rows, northbank, none)
	if len(got.Merged) != 3 || got.Merged[2] != 0 || got.Merged[4] != 3 || got.Merged[11] != 10 {
		t.Fatalf("merged = %v", got.Merged)
	}
	if len(got.Refunded) != 1 || got.Refunded[9] != 8 {
		t.Fatalf("refunded = %v", got.Refunded)
	}
	// Without the bank's wording nothing is recognised.
	if g := PairAdjustments(rows, Generic, none); len(g.Merged)+len(g.Refunded) != 0 {
		t.Fatalf("generic paired %v", g)
	}
}

func TestDetectBank(t *testing.T) {
	other := Bank{ID: "zz-other", Patterns: []Pattern{NewPattern(RolePurchase, `^pos\s+(?P<merchant>.+)`, 0)}}
	rows := []Row{
		{Note: "CARD CORNER GROCER"}, {Note: "CASHBACK CORNER GROCER"}, {Note: "Salary"}, {Note: "POS CITY CABS"},
	}
	if b := DetectBank(rows, []Bank{other, northbank}); b.ID != "zz-northbank" {
		t.Fatalf("detected %s", b.ID)
	}
	// One match isn't enough.
	if b := DetectBank(rows[2:], []Bank{other, northbank}); b.ID != Generic.ID {
		t.Fatalf("detected %s", b.ID)
	}
}

// Two purchases from one merchant on one day, each listed with its own discount: every discount
// goes to the purchase next to it, whichever way the statement is sorted.
func TestPairAdjustmentsSameDayByPosition(t *testing.T) {
	day := on("2026-10-02")
	oldestFirst := []Row{
		{Date: day, Amount: -895, Note: "CARD CORNER CAFE"},
		{Date: day, Amount: 166, Note: "CASHBACK CORNER CAFE"},
		{Date: day, Amount: -4000, Note: "CARD MOON SHOP"},
		{Date: day, Amount: -2250, Note: "CARD CORNER CAFE"},
		{Date: day, Amount: 66, Note: "CASHBACK CORNER CAFE"},
	}
	got := PairAdjustments(oldestFirst, northbank, func(int) bool { return false })
	if len(got.Merged) != 2 || got.Merged[1] != 0 || got.Merged[4] != 3 {
		t.Fatalf("oldest first: %v", got.Merged)
	}

	newestFirst := []Row{oldestFirst[4], oldestFirst[3], oldestFirst[2], oldestFirst[1], oldestFirst[0]}
	got = PairAdjustments(newestFirst, northbank, func(int) bool { return false })
	if len(got.Merged) != 2 || got.Merged[0] != 1 || got.Merged[3] != 4 {
		t.Fatalf("newest first: %v", got.Merged)
	}
}
