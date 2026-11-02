package ledger

import (
	"context"
	"math/big"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/money"
	"github.com/google/uuid"
)

type Transfer struct {
	ID           uuid.UUID
	FromWalletID uuid.UUID
	ToWalletID   uuid.UUID
	FromAmount   int64
	ToAmount     int64
	Rate         string
	OccurredOn   time.Time
	Note         string
	LabelIDs     []uuid.UUID
}

type TransferInput struct {
	FromWalletID uuid.UUID
	ToWalletID   uuid.UUID
	FromAmount   int64
	ToAmount     int64
	// Target units per 1 source unit. Optional when both wallets share a currency.
	Rate       string
	OccurredOn time.Time
	Note       string
	LabelIDs   []uuid.UUID

	// Same as on TransactionInput.
	Repeat Recurrence
	Scope  Scope
	Today  time.Time
}

// prepareTransfer validates wallets, amounts and the rate. Returns the normalised rate.
func (s *Service) prepareTransfer(ctx context.Context, q *dbq.Queries, a Actor, in *TransferInput) error {
	if in.FromWalletID == in.ToWalletID {
		return invalid("to_wallet_id", "must differ from the source wallet")
	}
	if in.OccurredOn.IsZero() {
		return invalid("occurred_on", "is required")
	}
	if err := checkAmount("from_amount_minor", in.FromAmount, false); err != nil {
		return err
	}
	if err := checkAmount("to_amount_minor", in.ToAmount, false); err != nil {
		return err
	}
	note, err := checkNote(in.Note)
	if err != nil {
		return err
	}
	in.Note = note

	ws, err := q.GetWalletsByIDs(ctx, dbq.GetWalletsByIDsParams{HouseholdID: a.HouseholdID, Ids: []uuid.UUID{in.FromWalletID, in.ToWalletID}})
	if err != nil {
		return err
	}
	var from, to *dbq.GetWalletsByIDsRow
	for i := range ws {
		switch ws[i].ID {
		case in.FromWalletID:
			from = &ws[i]
		case in.ToWalletID:
			to = &ws[i]
		}
	}
	if from == nil {
		return invalid("from_wallet_id", "wallet doesn't exist")
	}
	if to == nil {
		return invalid("to_wallet_id", "wallet doesn't exist")
	}
	if from.ArchivedAt != nil || to.ArchivedAt != nil {
		return invalid("wallet", "can't transfer with an archived wallet")
	}

	if from.Currency == to.Currency {
		if in.FromAmount != in.ToAmount {
			return invalid("to_amount_minor", "must equal the sent amount for same-currency transfers")
		}
		in.Rate = money.FormatRate(mustRate("1"))
	} else {
		rate, err := money.ParseRate(in.Rate)
		if err != nil {
			return invalid("rate", "%s", err.Error())
		}
		if err := money.CheckTransfer(in.FromAmount, in.ToAmount, rate, int(from.MinorUnits), int(to.MinorUnits)); err != nil {
			return invalid("rate", "%s", err.Error())
		}
		in.Rate = money.FormatRate(rate)
	}

	in.LabelIDs, err = checkLabels(ctx, q, a, in.LabelIDs)
	return err
}

func mustRate(s string) *big.Rat {
	r, err := money.ParseRate(s)
	if err != nil {
		panic(err)
	}
	return r
}

// insertTransfer writes a validated transfer and its two legs, returning the transfer's id.
func insertTransfer(ctx context.Context, q *dbq.Queries, householdID uuid.UUID, createdBy *uuid.UUID, in TransferInput,
	rl recurringLink,
) (uuid.UUID, error) {
	id := newID()
	if err := q.CreateTransfer(ctx, dbq.CreateTransferParams{
		ID: id, HouseholdID: householdID, FromWalletID: in.FromWalletID, ToWalletID: in.ToWalletID,
		FromAmountMinor: in.FromAmount, ToAmountMinor: in.ToAmount, Rate: in.Rate, OccurredOn: in.OccurredOn,
		Note: in.Note, CreatedBy: createdBy,
	}); err != nil {
		return uuid.Nil, dbErr(err, "transfer")
	}
	for _, leg := range []struct {
		kind   TransactionKind
		wallet uuid.UUID
		amount int64
	}{
		{KindTransferOut, in.FromWalletID, in.FromAmount},
		{KindTransferIn, in.ToWalletID, in.ToAmount},
	} {
		t, err := q.CreateTransaction(ctx, dbq.CreateTransactionParams{
			ID: newID(), HouseholdID: householdID, WalletID: leg.wallet, Kind: leg.kind, TransferID: &id,
			AmountMinor: leg.amount, OccurredOn: in.OccurredOn, Note: in.Note, CreatedBy: createdBy,
			RecurringRuleID: rl.ruleID, OccurrenceDate: rl.on,
		})
		if err != nil {
			return uuid.Nil, dbErr(err, "transfer leg")
		}
		if len(in.LabelIDs) > 0 {
			if err := q.AddTransactionLabels(ctx, dbq.AddTransactionLabelsParams{TransactionID: t.ID, LabelIds: in.LabelIDs}); err != nil {
				return uuid.Nil, err
			}
		}
	}
	return id, nil
}

// CreateTransfer moves money between two wallets. With in.Repeat it becomes the first occurrence
// of a recurring rule (every occurrence has its own two legs).
func (s *Service) CreateTransfer(ctx context.Context, a Actor, in TransferInput) (Transfer, error) {
	var id uuid.UUID
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		if err := s.prepareTransfer(ctx, q, a, &in); err != nil {
			return err
		}
		if err := in.Repeat.normalize(in.OccurredOn); err != nil {
			return err
		}
		var err error
		if id, err = insertTransfer(ctx, q, a.HouseholdID, &a.UserID, in, recurringLink{}); err != nil {
			return err
		}
		if in.Repeat.Repeats() {
			return s.createRule(ctx, q, a, transferTemplate(in), in.Repeat, in.OccurredOn, orToday(in.Today),
				occurrence{transferID: &id})
		}
		return nil
	})
	if err != nil {
		return Transfer{}, err
	}
	return transferFrom(id, in), nil
}

// UpdateTransfer replaces a transfer. Recurring transfers follow the same rules as
// UpdateTransaction.
func (s *Service) UpdateTransfer(ctx context.Context, a Actor, id uuid.UUID, in TransferInput) (Transfer, error) {
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		if err := s.prepareTransfer(ctx, q, a, &in); err != nil {
			return err
		}
		cur, err := q.GetTransferOccurrence(ctx, dbq.GetTransferOccurrenceParams{TransferID: &id, HouseholdID: a.HouseholdID})
		if err != nil {
			return dbErr(err, "transfer")
		}
		recurring := cur.RecurringRuleID != nil && cur.OccurrenceDate != nil
		series := recurring && in.Scope == ScopeThisAndFuture
		if !series && recurring {
			in.Repeat = Recurrence{}
		}
		if err := in.Repeat.normalize(in.OccurredOn); err != nil {
			return err
		}
		n, err := q.UpdateTransfer(ctx, dbq.UpdateTransferParams{
			ID: id, HouseholdID: a.HouseholdID, FromWalletID: in.FromWalletID, ToWalletID: in.ToWalletID,
			FromAmountMinor: in.FromAmount, ToAmountMinor: in.ToAmount, Rate: in.Rate, OccurredOn: in.OccurredOn, Note: in.Note,
		})
		if err != nil {
			return dbErr(err, "transfer")
		}
		if n == 0 {
			return ErrNotFound
		}
		var legIDs []uuid.UUID
		for _, leg := range []struct {
			kind   TransactionKind
			wallet uuid.UUID
			amount int64
		}{
			{KindTransferOut, in.FromWalletID, in.FromAmount},
			{KindTransferIn, in.ToWalletID, in.ToAmount},
		} {
			legID, err := q.UpdateTransferLeg(ctx, dbq.UpdateTransferLegParams{
				TransferID: &id, Kind: leg.kind, WalletID: leg.wallet, AmountMinor: leg.amount, OccurredOn: in.OccurredOn, Note: in.Note,
			})
			if err != nil {
				return dbErr(err, "transfer leg")
			}
			legIDs = append(legIDs, legID)
		}
		if err := q.ClearTransactionLabels(ctx, legIDs); err != nil {
			return err
		}
		for _, legID := range legIDs {
			if len(in.LabelIDs) > 0 {
				if err := q.AddTransactionLabels(ctx, dbq.AddTransactionLabelsParams{TransactionID: legID, LabelIds: in.LabelIDs}); err != nil {
					return err
				}
			}
		}
		today, tpl, chosen := orToday(in.Today), transferTemplate(in), occurrence{transferID: &id}
		switch {
		case series:
			return s.editSeries(ctx, q, a, *cur.RecurringRuleID, *cur.OccurrenceDate, in.OccurredOn, tpl, in.Repeat, today, chosen)
		case !recurring && in.Repeat.Repeats():
			return s.createRule(ctx, q, a, tpl, in.Repeat, in.OccurredOn, today, chosen)
		}
		return nil
	})
	if err != nil {
		return Transfer{}, err
	}
	return transferFrom(id, in), nil
}

func transferFrom(id uuid.UUID, in TransferInput) Transfer {
	labels := in.LabelIDs
	if labels == nil {
		labels = []uuid.UUID{}
	}
	return Transfer{
		ID: id, FromWalletID: in.FromWalletID, ToWalletID: in.ToWalletID, FromAmount: in.FromAmount, ToAmount: in.ToAmount,
		Rate: in.Rate, OccurredOn: in.OccurredOn, Note: in.Note, LabelIDs: labels,
	}
}
