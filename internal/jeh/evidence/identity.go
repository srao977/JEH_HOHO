// File: internal/jeh/evidence/identity.go (identity.go) // Date: 2026-09-17 // Version/Status: 1.0 / Graduated frozen JEH behavior // Product/Component: JEH-HOHO / Frozen JEH Engine // Purpose: Preserve the file's existing DSE_JEH implementation or regression. // Origin: DSE_JEH_Lab/internal/jeh/evidence/identity.go, Slice 0 fingerprint // Invariants: Governed JEH/Capital/runtime semantics unchanged // Non-responsibilities: No Fin transport and no public product contract.

package evidence

import (
	"crypto/sha256"
	"encoding/hex"
)

func ID(parts ...string) string {
	hash := sha256.New()
	for index, part := range parts {
		if index > 0 {
			_, _ = hash.Write([]byte{0})
		}
		_, _ = hash.Write([]byte(part))
	}
	return hex.EncodeToString(hash.Sum(nil))
}
