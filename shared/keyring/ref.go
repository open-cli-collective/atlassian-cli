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

	// KeyAPIToken is the single shared API token bundle key. There is
	// one key per logical credential (Secret-Handling Standard §1.11.10);
	// jtk and cfl resolve the same api_token. There are no per-tool
	// override keys.
	KeyAPIToken = "api_token" //nolint:gosec // G101: a bundle key name, not a credential

	// ToolCFL / ToolJTK identify the calling tool for the unchanged
	// per-tool env-var selection only (§ envVarsFor); the keyring key is
	// shared regardless of tool.
	ToolCFL = "cfl"
	ToolJTK = "jtk"
)

// allowedKeys is the §1.5.2 allowlist and the §1.11.11 conforming bundle
// key set: exactly the one shared token key.
var allowedKeys = []string{KeyAPIToken}
