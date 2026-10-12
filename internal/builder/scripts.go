package builder

import _ "embed"

// OpenZFSKeys are the public keys that sign OpenZFS releases (fingerprints in OpenZFSSigningKeys). Shipped
// with the builder so source verification does not depend on a keyserver.
//
//go:embed openzfs-keys.asc
var OpenZFSKeys string
