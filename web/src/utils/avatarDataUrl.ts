/**
 * Stored avatars are raw base64: legacy default is JPEG; minime output is PNG.
 */
export function userAvatarToDataUrl(avatar: string | undefined): string | undefined {
  if (!avatar?.trim()) return undefined;
  if (avatar.startsWith('data:')) return avatar;
  if (avatar.startsWith('/9j/')) return `data:image/jpeg;base64,${avatar}`;
  return `data:image/png;base64,${avatar}`;
}
