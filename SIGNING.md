# Code Signing for tidy

Code signing cryptographically signs your executables so Windows SmartScreen recognizes them as trusted. Unsigned binaries downloaded from the internet trigger the "Windows protected your PC" warning, requiring users to click "More info" and "Run anyway." Signed binaries with sufficient reputation skip this friction entirely.

## EV vs Standard certificates

| Feature | EV Code Signing | Standard Code Signing |
|---------|----------------|----------------------|
| **Cost** | ~$400-600/year | ~$150-250/year |
| **SmartScreen reputation** | Instant | Builds over weeks of downloads |
| **Validation** | Organization validation required | Simpler identity verification |
| **Hardware token** | Yes (USB eToken shipped) | No |
| **Issuance time** | 1-4 weeks | Minutes to hours |

**Recommendation for tidy:** Go EV. As a solo developer shipping to the public, instant SmartScreen bypass is the goal. Standard certs require reputation buildup, which means early users still see warnings.

## Recommended vendors

- **Sectigo** (formerly Comodo) — Cheapest EV option, good for solo developers. ~$400/year for EV Code Signing.
- **SSL.com** — Mid-range pricing, solid documentation. ~$450/year for EV.
- **DigiCert** — Enterprise-grade, expensive. Overkill for a single-developer project.

**Warning:** Avoid cheap "code signing" certificates from unknown vendors. Many are just standard certificates rebranded with misleading marketing. Stick with established CAs.

## Buying the cert

### Sectigo EV Code Signing (recommended)

1. Visit sectigo.com and navigate to Code Signing Certificates > EV Code Signing.
2. Select the EV Code Signing certificate and add to cart.
3. During checkout, you'll need to register as an organization. As a solo developer, you can register as a business entity or DBA (Doing Business As). You'll need:
   - Business name (your name or DBA)
   - Business address
   - Phone number
   - DUNS number (free from Dun & Bradstreet, takes ~2 weeks to obtain)
4. Complete the purchase.
5. Sectigo initiates organization validation. This involves:
   - Verifying your business registration
   - Phone verification (they call the number on file)
   - Address verification
   - This process takes 1-4 weeks depending on how quickly you respond to verification requests.
6. Once validated, Sectigo generates your certificate and ships a USB hardware token (SafeNet eToken) to your address.
7. When the token arrives, install the SafeNet Authentication Client software (Sectigo provides download links).
8. Insert the USB token. The certificate and private key are stored on the token, not on your filesystem.

## Exporting cert + key to PEM for osslsigncode

`osslsigncode` requires PEM-format files, not the Windows PFX/P12 format most vendors ship.

### If your vendor provides a .pfx file

```bash
# Extract private key (you'll be prompted for the PFX password)
openssl pkcs12 -in cert.pfx -nocerts -out key.pem -nodes

# Extract certificate
openssl pkcs12 -in cert.pfx -clcerts -nokeys -out cert.pem
```

### If you have an EV certificate with hardware token

**Problem:** `osslsigncode` cannot access private keys stored on hardware tokens (PKCS#11 devices). The private key never leaves the token.

**Solution 1: Use signtool on Windows**

If you have access to a Windows machine (physical or VM), use Microsoft's `signtool.exe` with the SafeNet Authentication Client. This is the native Windows workflow.

**Solution 2: Use jsign (Java-based, supports PKCS#11)**

`jsign` is a cross-platform code signing tool that supports hardware tokens via PKCS#11.

Install jsign:

```bash
# Download from https://github.com/ebourg/jsign/releases
# Extract and add to PATH
```

Sign with jsign:

```bash
jsign --storetype PKCS11 \
      --storepass "" \
      --name "tidy" \
      --url "https://github.com/YousefMohiey/tidy" \
      --tsaurl "http://timestamp.sectigo.com" \
      --alg SHA-256 \
      dist/tidy-Windows-amd64.exe
```

The `--storepass ""` prompts for your token PIN. jsign signs the file in-place (no `-signed.exe` suffix).

## Signing the binaries

Once you have `cert.pem` and `key.pem` in PEM format:

```bash
./sign.sh cert.pem key.pem dist/tidy-Windows-amd64.exe dist/tidy-Setup-x.x.x.exe
```

The script produces `dist/tidy-Windows-amd64.exe-signed` and `dist/tidy-Setup-x.x.x.exe-signed`.

Verify the signature:

```bash
osslsigncode verify dist/tidy-Windows-amd64.exe-signed
```

Expected output: `Signature verification: ok`

Replace unsigned files with signed versions:

```bash
mv dist/tidy-Windows-amd64.exe-signed dist/tidy-Windows-amd64.exe
mv dist/tidy-Setup-x.x.x.exe-signed dist/tidy-Setup-x.x.x.exe
```

## Uploading to GitHub Releases

Upload the signed binaries as release assets on GitHub. When users download them, SmartScreen recognizes the valid signature and reputation, and the warning is suppressed.

For automated releases, integrate `sign.sh` into your CI/CD pipeline. Store `cert.pem` and `key.pem` as encrypted secrets in your CI system (never commit them to the repository).

## Troubleshooting

### Windows still shows warning after signing

**EV certificates:** Should be instant. If you still see warnings, verify the signature with `osslsigncode verify` and ensure the timestamp is valid.

**Standard certificates:** Reputation takes time. Windows builds trust based on download volume over weeks. Early users will see warnings until sufficient reputation accumulates.

### osslsigncode: timestamp failed

The timestamp server may be temporarily unavailable. Try an alternate RFC3161 timestamp URL:

```bash
# In sign.sh, change the -ts flag to:
-ts "http://timestamp.digicert.com"
```

Both Sectigo and DigiCert provide free public timestamp servers.

### openssl pkcs12 fails with 'Mac verify error'

You entered the wrong password for the .pfx file. The password is case-sensitive. If you've forgotten it, you'll need to re-export the certificate from your vendor's portal.

### osslsigncode: signing failed

Verify your `cert.pem` and `key.pem` are valid:

```bash
openssl x509 -in cert.pem -text -noout
openssl rsa -in key.pem -check
```

If the certificate shows "Not Before" in the future, the certificate isn't yet valid. Wait until the validity period starts.
