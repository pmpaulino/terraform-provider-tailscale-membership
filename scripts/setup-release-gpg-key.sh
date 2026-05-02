#!/usr/bin/env bash
#
# setup-release-gpg-key.sh
#
# One-shot helper that creates a dedicated GPG key for signing
# pmpaulino/tailscale-membership releases (FR-015 from
# specs/002-standalone-membership-provider/spec.md), prints the
# fingerprint, and outputs the secret/passphrase blobs you need
# to paste into:
#
#   - GitHub repo Settings -> Secrets and variables -> Actions
#       GPG_PRIVATE_KEY  (the armored private key)
#       PASSPHRASE       (the passphrase you choose below)
#
# It also prints the armored public key so you can drop it into the
# in-repo /KEYS file produced by task T075.
#
# Run this on your workstation (not in CI). It is interactive only at
# the passphrase prompt; everything else is fully scripted.
#
# Requires: gpg >= 2.2
#
# NOTE: RSA 4096 is used (not ed25519) because the Terraform Registry only
# accepts RSA and DSA public keys for release signature verification.

set -euo pipefail

if ! command -v gpg >/dev/null 2>&1; then
  echo "ERROR: 'gpg' binary not found on PATH." >&2
  echo "" >&2
  echo "On macOS:    brew install gnupg" >&2
  echo "On Debian:   sudo apt-get install -y gnupg" >&2
  echo "On Fedora:   sudo dnf install -y gnupg2" >&2
  echo "" >&2
  echo "After installing, re-run this script. Requires gpg >= 2.2." >&2
  exit 1
fi

REAL_NAME="${RELEASE_KEY_NAME:-pmpaulino tailscale-membership release signing key}"
REAL_EMAIL="${RELEASE_KEY_EMAIL:-}"

if [[ -z "$REAL_EMAIL" ]]; then
  read -rp "Email to associate with the signing key (e.g. you@example.com): " REAL_EMAIL
fi

echo
echo ">>> You will now be prompted for a PASSPHRASE."
echo ">>> Pick a strong one and store it in your password manager;"
echo ">>> you will paste the SAME value into the GitHub Actions secret PASSPHRASE."
echo

read -rsp "Passphrase: " PASSPHRASE
echo
read -rsp "Passphrase (confirm): " PASSPHRASE_CONFIRM
echo

if [[ "$PASSPHRASE" != "$PASSPHRASE_CONFIRM" ]]; then
  echo "ERROR: passphrases do not match" >&2
  exit 1
fi

if [[ ${#PASSPHRASE} -lt 12 ]]; then
  echo "ERROR: passphrase must be at least 12 characters" >&2
  exit 1
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

BATCH_FILE="$WORKDIR/batch.txt"
cat > "$BATCH_FILE" <<EOF
%echo Generating release signing key for $REAL_NAME <$REAL_EMAIL>
Key-Type: RSA
Key-Length: 4096
Key-Usage: sign
Subkey-Type: RSA
Subkey-Length: 4096
Subkey-Usage: encrypt
Name-Real: $REAL_NAME
Name-Email: $REAL_EMAIL
Expire-Date: 2y
Passphrase: $PASSPHRASE
%commit
%echo Done
EOF

echo
echo ">>> Generating RSA 4096 key pair (this may take 10–30 seconds)..."
gpg --batch --pinentry-mode loopback --generate-key "$BATCH_FILE"

# Resolve the fingerprint of the key we just created (matched on email).
FINGERPRINT="$(gpg --with-colons --list-keys "$REAL_EMAIL" \
  | awk -F: '/^fpr:/ { print $10; exit }')"

if [[ -z "$FINGERPRINT" ]]; then
  echo "ERROR: failed to read fingerprint of newly generated key" >&2
  exit 1
fi

KEY_ID_LONG="$(gpg --with-colons --list-keys "$FINGERPRINT" \
  | awk -F: '/^pub:/ { print $5; exit }')"

OUT_DIR="${RELEASE_KEY_OUT_DIR:-$HOME/.tailscale-membership-release-key}"
mkdir -p "$OUT_DIR"
chmod 700 "$OUT_DIR"

PRIVATE_KEY_FILE="$OUT_DIR/private-key.asc"
PUBLIC_KEY_FILE="$OUT_DIR/public-key.asc"
PASSPHRASE_FILE="$OUT_DIR/passphrase.txt"

gpg --batch --pinentry-mode loopback --passphrase "$PASSPHRASE" \
  --armor --export-secret-keys "$FINGERPRINT" > "$PRIVATE_KEY_FILE"
gpg --armor --export "$FINGERPRINT" > "$PUBLIC_KEY_FILE"
printf '%s\n' "$PASSPHRASE" > "$PASSPHRASE_FILE"

chmod 600 "$PRIVATE_KEY_FILE" "$PUBLIC_KEY_FILE" "$PASSPHRASE_FILE"

echo
echo "============================================================"
echo "  Release signing key created."
echo "============================================================"
echo
echo "  Fingerprint : $FINGERPRINT"
echo "  Long key ID : $KEY_ID_LONG"
echo "  Email       : $REAL_EMAIL"
echo
echo "  Files written to: $OUT_DIR"
echo "    - private-key.asc  (paste into GitHub Secret GPG_PRIVATE_KEY)"
echo "    - public-key.asc   (use as the body of the in-repo /KEYS file in task T075)"
echo "    - passphrase.txt   (paste into GitHub Secret PASSPHRASE)"
echo
echo "Next steps:"
echo "  1. Update the in-repo /KEYS file with the contents of $PUBLIC_KEY_FILE"
echo "  2. Update the fingerprint in README.md 'Verifying releases' section"
echo "  3. Open https://github.com/pmpaulino/terraform-provider-tailscale-membership/settings/secrets/actions"
echo "     - Update secret GPG_PRIVATE_KEY  -> contents of $PRIVATE_KEY_FILE"
echo "     - Update secret PASSPHRASE       -> contents of $PASSPHRASE_FILE"
echo "     - Update secret GPG_FINGERPRINT  -> $FINGERPRINT"
echo "  4. Commit and push the updated KEYS + README, then retag v1.0.0 to rebuild signed artifacts"
echo "  5. Add the public key to the Terraform Registry:"
echo "     https://registry.terraform.io/settings/gpg-keys"
echo "     (paste contents of $PUBLIC_KEY_FILE)"
echo
echo "Then DELETE the local copies you no longer need:"
echo "    rm $PASSPHRASE_FILE $PRIVATE_KEY_FILE   # keep public-key.asc until KEYS is updated"
echo
