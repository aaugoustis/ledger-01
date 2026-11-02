package apitest

import (
	"net/http"
	"testing"
)

// HouseholdScenario invites the partner into the owner's household, checks the member can't
// administer it, then removes them and checks their session loses access. It first undoes an
// earlier run, so both drivers can run it against the same database.
func HouseholdScenario(t *testing.T, do Driver) Result {
	t.Helper()
	var res Result
	login := func(email string) string {
		r := do(t, http.MethodPost, "/auth/dev/login", map[string]string{"email": email}, "")
		session := SessionCookie(r)
		if session == "" {
			t.Fatalf("dev login %s = %d", email, r.Status)
		}
		return session
	}
	householdOf := func(session string) map[string]any {
		r := do(t, http.MethodPost, rpcPath("UserService", "GetMe"), "{}", session)
		if r.Status != http.StatusOK {
			t.Fatalf("GetMe = %d %s", r.Status, r.Body)
		}
		return r.JSON(t)["household"].(map[string]any)
	}

	owner := login("owner@example.com")
	r := do(t, http.MethodPost, rpcPath("HouseholdService", "ListMembers"), "{}", owner)
	for _, m := range list(r.JSON(t)["members"]) {
		if m["role"] != "owner" {
			do(t, http.MethodPost, rpcPath("HouseholdService", "RemoveMember"), map[string]any{"userId": m["userId"]}, owner)
		}
	}

	r = do(t, http.MethodPost, rpcPath("HouseholdService", "InviteMember"), map[string]any{"email": "partner@example.com"}, owner)
	if r.Status != http.StatusOK {
		t.Fatalf("invite = %d %s", r.Status, r.Body)
	}
	res.add("invite %d allowed=%v", r.Status, r.JSON(t)["invite"].(map[string]any)["allowed"])
	r = do(t, http.MethodPost, rpcPath("HouseholdService", "InviteMember"), map[string]any{"email": "stranger@example.com"}, owner)
	res.add("invite-unlisted %d allowed=%v", r.Status, r.JSON(t)["invite"].(map[string]any)["allowed"])
	r = do(t, http.MethodPost, rpcPath("HouseholdService", "CancelInvite"), map[string]any{"email": "stranger@example.com"}, owner)
	res.add("cancel %d", r.Status)

	partner := login("partner@example.com")
	ph, oh := householdOf(partner), householdOf(owner)
	res.add("joined same=%v role=%v", ph["id"] == oh["id"], ph["role"])

	r = do(t, http.MethodPost, rpcPath("HouseholdService", "InviteMember"), map[string]any{"email": "x@example.com"}, partner)
	res.add("member-invite %d", r.Status)
	r = do(t, http.MethodPost, rpcPath("HouseholdService", "ListMembers"), "{}", partner)
	res.add("members %d", len(list(r.JSON(t)["members"])))

	var partnerID any
	for _, m := range list(r.JSON(t)["members"]) {
		if m["role"] == "member" {
			partnerID = m["userId"]
		}
	}
	r = do(t, http.MethodPost, rpcPath("HouseholdService", "RemoveMember"), map[string]any{"userId": partnerID}, owner)
	res.add("remove %d", r.Status)
	// The partner's session drops the owner's household: they're signed out, or back in a
	// household of their own if an earlier scenario gave them one.
	r = do(t, http.MethodPost, rpcPath("UserService", "GetMe"), "{}", partner)
	lost := r.Status == http.StatusUnauthorized
	if r.Status == http.StatusOK {
		lost = r.JSON(t)["household"].(map[string]any)["id"] != oh["id"]
	}
	res.add("removed lost-access=%v", lost)
	return res
}

func list(v any) []map[string]any {
	items, _ := v.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, it.(map[string]any))
	}
	return out
}
