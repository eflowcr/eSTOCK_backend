package assets

import _ "embed"

// LogoEPRAC holds the ePRAC logo PNG, embedded at compile time.
//
//go:embed logo_ePRAC.png
var LogoEPRAC []byte
