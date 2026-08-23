package accounting

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// ReceiptHash computes the Beleg-Hash: one digest over the ordered file list of a
// Beleg.
//
// It exists so the journal keeps covering exactly one Belegfeld while that field
// now stands for the whole file list instead of a single file. A ZUGFeRD invoice
// swapped for a different XML, an attachment removed, two files reordered — each
// changes this value and therefore the entry hash that carries it.
//
// The covered fields are role, original file name and file digest, as laid down
// in the concept. Deliberately *not* covered is ReceiptFile.Derived: it records
// provenance, while role and content are what the chain has to pin. Also not
// covered is StoredPath — moving the data directory must not break anything,
// which is the same reason the journal chain leaves the path out.
func ReceiptHash(r *domain.Receipt) string {
	var w canonicalWriter

	files := make([]domain.ReceiptFile, len(r.Files))
	copy(files, r.Files)
	sort.SliceStable(files, func(i, j int) bool { return files[i].Position < files[j].Position })

	// The count is written first so a truncated list cannot hash like a shorter
	// one that was always that length.
	w.putInt("files", int64(len(files)))
	for _, f := range files {
		w.put("role", string(f.Role))
		w.put("name", f.FileName)
		w.put("sha256", f.SHA256)
	}

	sum := sha256.Sum256(w.bytes())
	return hex.EncodeToString(sum[:])
}
