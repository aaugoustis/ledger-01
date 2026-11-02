// Package uy holds Uruguayan banks.
//
// Uruguay refunds part of the VAT on card purchases (laws 19.210 and 17.934, among others):
// statements list the refund as its own row after the purchase. Each bank words it its own way.
package uy

import "github.com/franco-roura/ledger-01/backend/internal/importer"

// Itau is Itaú Uruguay: "COMPRA <merchant>" for card purchases and "PAGO FACTURA<merchant>" for
// bill payments (no space: "PAGO FACTURAUTE 124186"), "REDIVA <law><merchant>" for the VAT
// refund (e.g. "REDIVA 19210ALMACEN ROSA"; on bill payments it's listed just before the
// payment), "DEVOLUCION <merchant>" or "ANULACION <merchant>" for refunds. Merchants are
// truncated to the same width in all of them.
var Itau = importer.Bank{
	ID:      "uy-itau",
	Name:    "Itaú",
	Country: "UY",
	Patterns: []importer.Pattern{
		importer.NewPattern(importer.RolePurchase, `^\s*compra\s+(?P<merchant>.+?)\s*$`, 0),
		importer.NewPattern(importer.RolePurchase, `^\s*pago\s+factura\s*(?P<merchant>.+?)\s*$`, 0),
		importer.NewPattern(importer.RoleDiscount, `^\s*rediva\s*\d{2}\.?\d{3}\s*(?P<merchant>.+?)\s*$`, importer.DefaultDiscountWindow),
		importer.NewPattern(importer.RoleRefund, `^\s*(?:devoluci[oó]n|anulaci[oó]n)\s+(?P<merchant>.+?)\s*$`, importer.DefaultRefundWindow),
	},
}
