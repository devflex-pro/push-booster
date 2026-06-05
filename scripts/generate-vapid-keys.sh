#!/usr/bin/env bash
set -euo pipefail

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

private_pem="$tmpdir/private.pem"
private_der="$tmpdir/private.der"
public_der="$tmpdir/public.der"

openssl ecparam -name prime256v1 -genkey -noout -out "$private_pem"
openssl pkcs8 -topk8 -nocrypt -in "$private_pem" -outform DER -out "$private_der"
openssl ec -in "$private_pem" -pubout -outform DER -out "$public_der" >/dev/null 2>&1

private_key="$(
  openssl ec -in "$private_pem" -text -noout 2>/dev/null |
    awk '
      /priv:/ { in_priv=1; next }
      /pub:/ { in_priv=0 }
      in_priv { gsub(/[:[:space:]]/, ""); printf "%s", $0 }
    ' |
    xxd -r -p |
    base64 -w0 |
    tr "+/" "-_" |
    tr -d "="
)"

public_key="$(
  tail -c 65 "$public_der" |
    base64 -w0 |
    tr "+/" "-_" |
    tr -d "="
)"

cat <<EOF
VAPID_PUBLIC_KEY=$public_key
VAPID_PRIVATE_KEY=$private_key
EOF
