/**
 * Regex to detect env var keys that typically contain sensitive data.
 * Matches: PASSWORD, PASSWD, SECRET, TOKEN, CREDENTIAL, AUTH, *_KEY, PRIVATE_KEY, API_KEY, etc.
 */
const SENSITIVE_KEY_REGEX =
  /(password|passwd|secret|token|credential|auth|private[_\-]?key|api[_\-]?key|_key$)/i;

/**
 * Returns true if the env var key suggests it contains sensitive data.
 */
export function isSensitiveEnvKey(key: string): boolean {
  return SENSITIVE_KEY_REGEX.test(key);
}

/**
 * Redacts the value if the key appears to contain sensitive data.
 */
export function redactEnvValue(key: string, value: string): string {
  if (isSensitiveEnvKey(key) && value.length > 0) {
    return '***redacted***';
  }
  return value;
}
