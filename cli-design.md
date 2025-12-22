# VaultSandbox CLI (`vsb`) Design Draft

## Overview
The `vsb` CLI acts as a developer companion for testing email flows. It leverages the Go SDK to handle complex end-to-end encryption transparently. The core principle is **Zero-Knowledge**: the server never sees plaintext; the CLI handles all key generation and decryption locally.

## Configuration & State
The CLI needs to persist state (inboxes, keys) and configuration.
*   **Location:** `~/.config/vsb/` (or OS equivalent via `os.UserConfigDir()`)
*   **Files:**
    *   `config.yaml`: API keys, base URL, default output preferences.
    *   `keystore.json`: Stores inbox metadata and **private keys**.
        *   *Security Note:* This file should have restrictive permissions (`0600`).
        *   *Structure:*
            ```json
            {
              "inboxes": [
                {
                  "email": "user@example.com",
                  "id": "inbox-hash",
                  "createdAt": "2024-01-01T00:00:00Z",
                  "expiresAt": "2024-01-08T00:00:00Z",
                  "keys": {
                    "kem_private": "base64...", 
                    "kem_public": "base64...",
                    "server_sig_pk": "base64..." // Pinned server key for verification
                  }
                }
              ],
              "active_inbox": "user@example.com"
            }
            ```

## Command Flow: Creating a New Inbox
This is the entry point for most users.

**Command:** `vsb inbox create [label]`

### Detailed Flow
1.  **Key Generation (Local):**
    *   CLI initializes the ML-KEM-768 algorithm.
    *   Generates a fresh public/private keypair.
    *   *User Feedback:* "Generating quantum-safe keys..." (Spinner)

2.  **Server Registration:**
    *   CLI sends `POST /api/inboxes` with the **Public Key**.
    *   *Payload:* `{"clientKemPk": "..."}`
    *   *User Feedback:* "Registering with VaultSandbox..."

3.  **Response Handling:**
    *   Server returns: `emailAddress`, `inboxHash`, `serverSigPk`, `expiresAt`.
    *   CLI verifies the response structure.

4.  **Persistence:**
    *   CLI saves the `emailAddress`, `inboxHash`, `serverSigPk`, and the locally generated `privateKey` to `keystore.json`.
    *   Sets this new inbox as the `active_inbox`.

5.  **Output (The "Wow" Moment):**
    *   Prints the new email address in a large, copyable format (using Lip Gloss).
    *   *Example Output:*
        ```text
        ✨ Inbox Ready!
        
        📬 Address:  8f92a.test@vaultsandbox.com
        🔑 Security: ML-KEM-768 (Quantum-Safe)
        ⏱️ Expires:  24 hours
        
        Run 'vsb watch' to see emails arrive live.
        ```

## Command: `vsb watch` (The Live Feed)
This command consumes the stored keys to provide a decrypted view.

1.  **Load State:** Reads `active_inbox` and its corresponding private key from `keystore.json`.
2.  **Connect:**
    *   Establishes an SSE connection to `/api/events`.
    *   *Fallback:* Starts polling if SSE fails.
3.  **Real-time Decryption Loop:**
    *   **Event Received:** Encrypted payload arrives.
    *   **Verify:** CLI verifies the `ML-DSA-65` signature using the pinned `serverSigPk`.
    *   **Decrypt:** CLI uses the stored `kem_private` key to decapsulate and decrypt the AES-GCM payload.
    *   **Render:** TUI updates instantly with the decrypted Subject, From, and Security Verdicts.

## Command: `vsb wait-for` (CI/CD)
Designed for scripting.

1.  **Arguments:** `--email <address>` (optional, defaults to active), `--subject-regex "..."`.
2.  **Load Keys:** Finds the private key for the target email.
3.  **Polling/Listening:** Blocks until a matching email arrives.
4.  **Decrypt & Match:** Decrypts headers of incoming messages to check against filter criteria.
5.  **Exit:** Returns `0` on match, `1` on timeout.

## Command: `vsb audit` (Deep-Dive)
Proves the "Production Fidelity" of the email flow.

**Command:** `vsb audit <email-id>` (or `--latest`)

1.  **Fetch:** Retrieves the specific email (encrypted).
2.  **Decrypt:** Uses local keys to unlock the payload.
3.  **Analysis:**
    *   **Transport Security:** Extracts TLS version and Cipher suite from metadata (e.g., `TLS 1.3 / X25519`).
    *   **Authentication:** Validates and displays details for SPF, DKIM (selector, key size), and DMARC.
    *   **MIME Structure:** Parses the raw source to display a tree view of the content (Headers -> Body -> Parts/Attachments).
4.  **Output:** A structured, colorful report proving the email was sent with production-grade security standards.

## Command: `vsb open` (Developer UX)
1.  **Fetch:** Gets the latest email for the active inbox.
2.  **Decrypt:** Full decryption (including body).
3.  **Parse:** Scans HTML/Text body for links (http/https).
4.  **Action:** Opens the *first* link found in the system default browser.

---

## Technical Stack Recommendation
*   **CLI Framework:** `cobra` (standard, robust).
*   **TUI:** `bubbletea` (for the `watch` dashboard).
*   **Styling:** `lipgloss` (for pretty boxes and colors).
*   **Config:** `viper` (to handle config file parsing).
*   **Storage:** `zalando/go-keyring` (Optional: if we want to store private keys in the OS keychain instead of a file, though a file is easier for CI).
