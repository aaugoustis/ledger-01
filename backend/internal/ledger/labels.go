package ledger

import (
	"context"

	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/google/uuid"
)

type Label struct {
	ID    uuid.UUID
	Name  string
	Color string
}

func labelFrom(l dbq.Label) Label { return Label{ID: l.ID, Name: l.Name, Color: l.Color} }

func (s *Service) ListLabels(ctx context.Context, a Actor) ([]Label, error) {
	rows, err := s.q.ListLabels(ctx, a.HouseholdID)
	if err != nil {
		return nil, err
	}
	out := make([]Label, len(rows))
	for i, r := range rows {
		out[i] = labelFrom(r)
	}
	return out, nil
}

func (s *Service) CreateLabel(ctx context.Context, a Actor, name, color string) (Label, error) {
	name, err := cleanName("name", name, 30)
	if err != nil {
		return Label{}, err
	}
	if err := checkIconColor(nil, &color); err != nil {
		return Label{}, err
	}
	l, err := s.q.CreateLabel(ctx, dbq.CreateLabelParams{ID: newID(), HouseholdID: a.HouseholdID, Name: name, Color: color})
	if err != nil {
		return Label{}, dbErr(err, "label")
	}
	return labelFrom(l), nil
}

func (s *Service) UpdateLabel(ctx context.Context, a Actor, id uuid.UUID, name, color *string) (Label, error) {
	if name != nil {
		n, err := cleanName("name", *name, 30)
		if err != nil {
			return Label{}, err
		}
		name = &n
	}
	if err := checkIconColor(nil, color); err != nil {
		return Label{}, err
	}
	l, err := s.q.UpdateLabel(ctx, dbq.UpdateLabelParams{ID: id, HouseholdID: a.HouseholdID, Name: name, Color: color})
	if err != nil {
		return Label{}, dbErr(err, "label")
	}
	return labelFrom(l), nil
}

func (s *Service) DeleteLabel(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.q.DeleteLabel(ctx, dbq.DeleteLabelParams{ID: id, HouseholdID: a.HouseholdID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// checkLabels verifies every label id belongs to the household and returns them de-duplicated.
func checkLabels(ctx context.Context, q *dbq.Queries, a Actor, ids []uuid.UUID) ([]uuid.UUID, error) {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return ids, nil
	}
	if len(ids) > 20 {
		return nil, invalid("label_ids", "at most 20 labels per transaction")
	}
	n, err := q.CountLabelsByIDs(ctx, dbq.CountLabelsByIDsParams{HouseholdID: a.HouseholdID, Ids: ids})
	if err != nil {
		return nil, err
	}
	if int(n) != len(ids) {
		return nil, invalid("label_ids", "some labels don't exist")
	}
	return ids, nil
}
