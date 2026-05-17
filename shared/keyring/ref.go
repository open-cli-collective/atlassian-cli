// Package keyring is atlassian-cli's credential adapter: a thin wrapper
// over cli-common's credstore (OS keyring — macOS Keychain, Linux Secret
// Service, Windows Credential Manager, or an opt-in encrypted-file
// backend). It is the single chokepoint through which both cfl and jtk
// store and read the Atlassian API token (the access secret). Non-secret
// configuration (url, email, auth_method, cloud_id, per-tool defaults)
// stays in the plaintext shared config file owned by package credstore.
//
// This package is pure library: it must not import cobra. The
// set-credential logic lives here as a plain function; each tool registers
// its own thin cobra wrapper.
package keyring

// The Atlassian token bundle lives under one fixed, shared ref so cfl and
// jtk collapse onto the same keyring entry (a token migrated by whichever
// binary runs first is seen as already-migrated by the other). The ref is
// a compile-time constant — there is no credential_ref config field.
const (
	// Service is the credstore service segment (also the keyring "service"
	// and the prefix of the passphrase env var,
	// ATLASSIAN_CLI_KEYRING_PASSPHRASE).
	Service = "atlassian-cli"
	// Profile is the credstore profile segment.
	Profile = "default"
	// Ref is the canonical "service/profile" string. Still ParseRef'd, never
	// string-split by hand.
	Ref = Service + "/" + Profile

	// KeyAPIToken is the shared/default API token bundle key.
	KeyAPIToken = "api_token" //nolint:gosec // G101: a bundle key name, not a credential
	// KeyCFLAPIToken / KeyJTKAPIToken are the per-tool override keys.
	KeyCFLAPIToken = "cfl_api_token" //nolint:gosec // G101: a bundle key name, not a credential
	KeyJTKAPIToken = "jtk_api_token" //nolint:gosec // G101: a bundle key name, not a credential

	// ToolCFL / ToolJTK identify the calling tool for per-tool resolution.
	ToolCFL = "cfl"
	ToolJTK = "jtk"
)

// allowedKeys is the §1.5.2 allowlist: exactly the three token keys.
var allowedKeys = []string{KeyAPIToken, KeyCFLAPIToken, KeyJTKAPIToken}

// KeyFor returns the per-tool override key for a tool, or "" for an
// unknown tool (callers fall back to KeyAPIToken).
func KeyFor(tool string) string {
	switch tool {
	case ToolCFL:
		return KeyCFLAPIToken
	case ToolJTK:
		return KeyJTKAPIToken
	default:
		return ""
	}
}
