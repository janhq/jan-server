/**
 * Hook to resolve media URLs for display.
 * Direct URLs are used as-is.
 */
export function useResolvedMediaUrl(url: string | undefined) {
  return {
    displayUrl: url,
    isLoading: false,
  };
}
