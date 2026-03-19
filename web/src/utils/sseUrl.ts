/**
 * EventSource cannot send Authorization headers. Append JWT as access_token for protected SSE routes.
 */
import { peekToken } from '../security/tokenVault';

export function withAccessToken(url: string): string {
  const token = peekToken();
  if (!token) {
    return url;
  }
  const sep = url.includes('?') ? '&' : '?';
  return `${url}${sep}access_token=${encodeURIComponent(token)}`;
}
