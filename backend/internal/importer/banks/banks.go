// Package banks holds how each bank words its statements, so the matching in package importer
// stays bank-agnostic. One file per bank under its country; see README.md to add yours.
package banks

import (
	"github.com/franco-roura/ledger-01/backend/internal/importer"
	"github.com/franco-roura/ledger-01/backend/internal/importer/banks/uy"
)

// All lists every known bank, Generic first.
func All() []importer.Bank {
	return []importer.Bank{
		importer.Generic,
		uy.Itau,
	}
}

// ByID returns a bank by its ID; unknown or empty IDs give Generic.
func ByID(id string) importer.Bank {
	for _, b := range All() {
		if b.ID == id {
			return b
		}
	}
	return importer.Generic
}
