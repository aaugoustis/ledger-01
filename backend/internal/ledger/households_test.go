package ledger_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/auth"
	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/testdb"
	"github.com/google/uuid"
)

func newEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@example.com", prefix, uuid.NewString())
}

// signIn is a Google sign-in for an email: it accepts pending invites like the real one.
func signIn(t *testing.T, svc *ledger.Service, email, name string) ledger.Actor {
	t.Helper()
	a, err := svc.EnsureUser(context.Background(), ledger.Profile{Subject: "test:" + email, Email: email, Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func mustInvite(t *testing.T, svc *ledger.Service, owner ledger.Actor, email string) {
	t.Helper()
	if _, err := svc.InviteMember(context.Background(), owner, email); err != nil {
		t.Fatal(err)
	}
}

func walletNames(t *testing.T, svc *ledger.Service, a ledger.Actor) []string {
	t.Helper()
	ws, err := svc.ListWallets(context.Background(), a, true, true, day)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, w := range ws {
		out = append(out, w.Name)
	}
	return out
}

func TestInviteNewUserJoinsHousehold(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	owner := signIn(t, svc, newEmail("owner"), "Ada Owner")
	mustWallet(t, svc, owner, "Shared", "USD", 10_00)
	email := newEmail("partner")
	mustInvite(t, svc, owner, email)

	// Inviting twice, or inviting a member, is a conflict.
	var conf *ledger.ConflictError
	if _, err := svc.InviteMember(ctx, owner, email); !errors.As(err, &conf) {
		t.Fatalf("second invite: err = %v, want conflict", err)
	}
	if _, err := svc.InviteMember(ctx, owner, "not an email"); err == nil {
		t.Fatal("invalid email accepted")
	}

	partner := signIn(t, svc, email, "Pat Partner")
	if partner.HouseholdID != owner.HouseholdID || partner.Role != ledger.RoleMember {
		t.Fatalf("partner = %+v, want member of %s", partner, owner.HouseholdID)
	}
	if got := walletNames(t, svc, partner); fmt.Sprint(got) != "[Shared]" {
		t.Fatalf("partner sees wallets %v", got)
	}
	hs, err := svc.Households(ctx, partner)
	if err != nil || len(hs) != 1 {
		t.Fatalf("a first-time invitee got their own household too: %+v, %v", hs, err)
	}
	ms, err := svc.Members(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms.Members) != 2 || ms.Members[0].Role != ledger.RoleOwner || len(ms.Invites) != 0 {
		t.Fatalf("members = %+v", ms)
	}
	if _, err := svc.InviteMember(ctx, owner, email); !errors.As(err, &conf) {
		t.Fatalf("inviting a member: err = %v, want conflict", err)
	}
}

func TestInviteExistingUserKeepsOwnHousehold(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	owner := signIn(t, svc, newEmail("owner"), "Ada Owner")
	mustWallet(t, svc, owner, "Shared", "USD", 0)
	email := newEmail("existing")
	own := signIn(t, svc, email, "Eli Existing")
	mustWallet(t, svc, own, "Mine", "USD", 0)

	mustInvite(t, svc, owner, email)
	joined := signIn(t, svc, email, "Eli Existing")
	if joined.HouseholdID != owner.HouseholdID {
		t.Fatalf("after accepting, active household = %s, want %s", joined.HouseholdID, owner.HouseholdID)
	}
	if got := walletNames(t, svc, joined); fmt.Sprint(got) != "[Shared]" {
		t.Fatalf("joined household shows %v", got)
	}
	hs, err := svc.Households(ctx, joined)
	if err != nil || len(hs) != 2 {
		t.Fatalf("households = %+v, %v", hs, err)
	}

	if err := svc.SwitchHousehold(ctx, joined, own.HouseholdID); err != nil {
		t.Fatal(err)
	}
	back := signIn(t, svc, email, "Eli Existing")
	if back.HouseholdID != own.HouseholdID || back.Role != ledger.RoleOwner {
		t.Fatalf("after switching, actor = %+v", back)
	}
	if got := walletNames(t, svc, back); fmt.Sprint(got) != "[Mine]" {
		t.Fatalf("own household shows %v", got)
	}
	// Not a member of a stranger's household.
	stranger := newActor(t, svc)
	if err := svc.SwitchHousehold(ctx, back, stranger.HouseholdID); !errors.Is(err, ledger.ErrNotFound) {
		t.Fatalf("switch to a stranger's household: err = %v", err)
	}
}

func TestMembersCantAdminister(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	owner := signIn(t, svc, newEmail("owner"), "Ada Owner")
	email := newEmail("member")
	mustInvite(t, svc, owner, email)
	member := signIn(t, svc, email, "Mo Member")
	mustInvite(t, svc, owner, newEmail("pending"))
	name, cur := "Renamed", "EUR"

	backup := export(t, svc, owner)
	for what, err := range map[string]error{
		"invite":           func() error { _, err := svc.InviteMember(ctx, member, newEmail("x")); return err }(),
		"cancel invite":    svc.CancelInvite(ctx, member, "pending@example.com"),
		"remove member":    svc.RemoveMember(ctx, member, owner.UserID),
		"update household": func() error { _, err := svc.UpdateHousehold(ctx, member, &name, &cur); return err }(),
		"restore":          func() error { _, err := svc.RestoreBackup(ctx, member, backup); return err }(),
	} {
		if !errors.Is(err, ledger.ErrForbidden) {
			t.Errorf("%s: err = %v, want ErrForbidden", what, err)
		}
	}

	// Members still edit the ledger.
	mustWallet(t, svc, member, "Groceries cash", "USD", 0)

	var conf *ledger.ConflictError
	if err := svc.LeaveHousehold(ctx, owner); !errors.As(err, &conf) {
		t.Fatalf("owner leaving: err = %v, want conflict", err)
	}
	if err := svc.RemoveMember(ctx, owner, owner.UserID); !errors.As(err, &conf) {
		t.Fatalf("owner removing themself: err = %v, want conflict", err)
	}
}

func TestRemovedMemberLosesAccess(t *testing.T) {
	pool := testdb.Pool(t)
	svc := ledger.New(pool)
	sessions := auth.NewSessions(dbq.New(pool), time.Hour)
	ctx := context.Background()
	owner := signIn(t, svc, newEmail("owner"), "Ada Owner")
	w := mustWallet(t, svc, owner, "Shared", "USD", 0)
	food := categoryNamed(t, svc, owner, ledger.CategoryExpense, "Food & Drink")
	email := newEmail("member")
	mustInvite(t, svc, owner, email)
	member := signIn(t, svc, email, "Mo Member")
	if _, err := svc.CreateTransaction(ctx, member, ledger.TransactionInput{WalletID: w.ID, CategoryID: food.ID,
		Amount: 9_00, OccurredOn: day, Note: "Snacks"}); err != nil {
		t.Fatal(err)
	}

	token, _, err := sessions.Create(ctx, member.UserID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if a, err := sessions.Resolve(ctx, token); err != nil || a.HouseholdID != owner.HouseholdID || a.Role != ledger.RoleMember {
		t.Fatalf("session before removal = %+v, %v", a, err)
	}
	if err := svc.RemoveMember(ctx, owner, member.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Resolve(ctx, token); !errors.Is(err, auth.ErrNoSession) {
		t.Fatalf("session after removal: err = %v, want ErrNoSession", err)
	}

	// Their rows stay, attributed to them, and they show as a former member.
	ts, _, err := svc.ListTransactions(ctx, owner, ledger.Filter{WalletID: w.ID, From: day, To: day, Today: day}, 0, "")
	if err != nil || len(ts) != 1 || ts[0].CreatedBy == nil || *ts[0].CreatedBy != member.UserID {
		t.Fatalf("transactions after removal = %+v, %v", ts, err)
	}
	ms, err := svc.Members(ctx, owner)
	if err != nil || len(ms.Members) != 1 || len(ms.Former) != 1 || ms.Former[0].UserID != member.UserID {
		t.Fatalf("members after removal = %+v, %v", ms, err)
	}

	// Signing in again gives them a household of their own.
	again := signIn(t, svc, email, "Mo Member")
	if again.HouseholdID == owner.HouseholdID || again.Role != ledger.RoleOwner {
		t.Fatalf("after removal, sign-in = %+v", again)
	}
}

func TestLeaveFallsBackToOwnHousehold(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	owner := signIn(t, svc, newEmail("owner"), "Ada Owner")
	email := newEmail("member")
	own := signIn(t, svc, email, "Mo Member")
	mustInvite(t, svc, owner, email)
	member := signIn(t, svc, email, "Mo Member")
	if err := svc.LeaveHousehold(ctx, member); err != nil {
		t.Fatal(err)
	}
	if back := signIn(t, svc, email, "Mo Member"); back.HouseholdID != own.HouseholdID {
		t.Fatalf("after leaving, household = %s, want %s", back.HouseholdID, own.HouseholdID)
	}
}

func TestBaseCurrencyBelongsToHousehold(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	owner := signIn(t, svc, newEmail("owner"), "Ada Owner")
	email := newEmail("member")
	mustInvite(t, svc, owner, email)
	member := signIn(t, svc, email, "Mo Member")

	cur := "eur"
	h, err := svc.UpdateHousehold(ctx, owner, nil, &cur)
	if err != nil || h.BaseCurrency != "EUR" {
		t.Fatalf("UpdateHousehold = %+v, %v", h, err)
	}
	if _, mh, err := svc.Me(ctx, member); err != nil || mh.BaseCurrency != "EUR" || mh.Role != ledger.RoleMember {
		t.Fatalf("member's household = %+v, %v", mh, err)
	}
	bad := "XXX"
	if _, err := svc.UpdateHousehold(ctx, owner, nil, &bad); err == nil {
		t.Fatal("unknown currency accepted")
	}
}

func TestFilterByMember(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	owner := signIn(t, svc, newEmail("owner"), "Ada Owner")
	email := newEmail("member")
	mustInvite(t, svc, owner, email)
	member := signIn(t, svc, email, "Mo Member")
	w := mustWallet(t, svc, owner, "Shared", "USD", 0)
	food := categoryNamed(t, svc, owner, ledger.CategoryExpense, "Food & Drink")
	for _, x := range []struct {
		a      ledger.Actor
		amount int64
		on     time.Time
	}{
		{owner, 10_00, day}, {member, 3_00, day}, {member, 4_00, day},
		{member, 5_00, day.AddDate(0, 1, 0)}, // upcoming
	} {
		if _, err := svc.CreateTransaction(ctx, x.a, ledger.TransactionInput{WalletID: w.ID, CategoryID: food.ID,
			Amount: x.amount, OccurredOn: x.on}); err != nil {
			t.Fatal(err)
		}
	}

	f := october()
	f.WalletID, f.Today, f.MemberIDs = w.ID, day, []uuid.UUID{member.UserID}
	ts, _, err := svc.ListTransactions(ctx, owner, f, 0, "")
	if err != nil || len(ts) != 2 {
		t.Fatalf("member's transactions = %d, %v; want 2", len(ts), err)
	}
	sum, err := svc.PeriodSummary(ctx, owner, f)
	if err != nil || sum.Expenses != 7_00 || sum.TransactionCount != 2 {
		t.Fatalf("member's summary = %+v, %v", sum, err)
	}
	found, _, err := svc.SearchTransactions(ctx, owner, ledger.Search{Filter: f}, 0, "")
	if err != nil || len(found) != 2 {
		t.Fatalf("search by member = %d, %v", len(found), err)
	}
	up, err := svc.ListUpcoming(ctx, owner, nil, []uuid.UUID{owner.UserID}, day)
	if err != nil || len(up) != 0 {
		t.Fatalf("owner's upcoming = %d, %v; want 0", len(up), err)
	}
	up, err = svc.ListUpcoming(ctx, owner, nil, []uuid.UUID{member.UserID}, day)
	if err != nil || len(up) != 1 {
		t.Fatalf("member's upcoming = %d, %v; want 1", len(up), err)
	}
}

func TestRestoreMatchesCreatorsByEmail(t *testing.T) {
	svc := ledger.New(testdb.Pool(t))
	ctx := context.Background()
	from := signIn(t, svc, newEmail("from"), "Fay From")
	sharedEmail := newEmail("shared")
	mustInvite(t, svc, from, sharedEmail)
	shared := signIn(t, svc, sharedEmail, "Sam Shared")
	w := mustWallet(t, svc, from, "Joint", "USD", 0)
	food := categoryNamed(t, svc, from, ledger.CategoryExpense, "Food & Drink")
	for _, a := range []ledger.Actor{from, shared} {
		if _, err := svc.CreateTransaction(ctx, a, ledger.TransactionInput{WalletID: w.ID, CategoryID: food.ID,
			Amount: 1_00, OccurredOn: day}); err != nil {
			t.Fatal(err)
		}
	}
	b := export(t, svc, from)
	if b.Version != ledger.BackupVersion || len(b.Members) != 2 {
		t.Fatalf("backup header = version %d, %d members", b.Version, len(b.Members))
	}

	// Another household where only the shared person is a member.
	to := signIn(t, svc, newEmail("to"), "Toni To")
	mustInvite(t, svc, to, sharedEmail)
	signIn(t, svc, sharedEmail, "Sam Shared")
	if _, err := svc.RestoreBackup(ctx, to, b); err != nil {
		t.Fatal(err)
	}
	ws, _ := svc.ListWallets(ctx, to, true, true, day)
	ts, _, err := svc.ListTransactions(ctx, to, ledger.Filter{WalletID: ws[0].ID, From: day, To: day, Today: day}, 0, "")
	if err != nil || len(ts) != 2 {
		t.Fatalf("restored transactions = %d, %v", len(ts), err)
	}
	creators := map[uuid.UUID]int{}
	for _, tx := range ts {
		creators[*tx.CreatedBy]++
	}
	// The shared member keeps their row; the creator who isn't a member here becomes the restorer.
	if creators[shared.UserID] != 1 || creators[to.UserID] != 1 {
		t.Fatalf("creators = %v (shared %s, restorer %s)", creators, shared.UserID, to.UserID)
	}
	// Restoring never changes membership.
	ms, _ := svc.Members(ctx, to)
	if len(ms.Members) != 2 {
		t.Fatalf("members after restore = %+v", ms.Members)
	}
}
