package quality

import _ "embed"

// snapshotJSON embarque la carte de qualité committée.
//
// Elle est EMBARQUÉE et non lue depuis le disque parce qu'un utilisateur qui
// lance `pepin control explain` n'a pas le dépôt sous la main : les scénarios de
// véracité, les tenants de référence et le registre de dette vivent dans le
// dépôt, pas dans l'exécutable. L'instantané est le seul pont entre les deux, et
// il est GÉNÉRÉ (internal/docgen) puis committé, donc TestGeneratedDocsAreUpToDate
// casse la CI dès qu'il dérive du dépôt.
//
//go:embed snapshot.json
var snapshotJSON []byte

// Embedded rend la carte compilée dans ce binaire.
func Embedded() (Snapshot, error) { return Decode(snapshotJSON) }
