/**
 * Validates whether a URL is an allowed external redirect target.
 * Uses proper URL parsing to prevent userinfo-based open redirect bypasses.
 */
export const isAllowedExternalRedirect = (value: string): boolean => {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' && url.hostname === 'localhost';
  } catch {
    return false;
  }
};
