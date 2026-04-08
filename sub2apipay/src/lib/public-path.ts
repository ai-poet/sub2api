function normalizeLeadingSlash(value: string): string {
  return value.startsWith('/') ? value : `/${value}`;
}

export function normalizeBasePath(value: string | null | undefined): string {
  const trimmed = (value || '').trim();
  if (!trimmed || trimmed === '/') return '';

  const normalized = normalizeLeadingSlash(trimmed).replace(/\/+$/, '');
  return normalized === '/' ? '' : normalized;
}

export function inferPublicBasePathFromPathname(pathname: string | null | undefined): string {
  const normalizedPath = normalizeLeadingSlash((pathname || '').trim() || '/');
  if (normalizedPath === '/pay' || normalizedPath.startsWith('/pay/')) {
    return '/pay';
  }
  return '';
}

export function getPublicBasePath(): string {
  if (typeof document === 'undefined') return '';

  const explicitBasePath = normalizeBasePath(document.documentElement.dataset.basePath);
  if (explicitBasePath) return explicitBasePath;

  if (typeof window !== 'undefined') {
    return inferPublicBasePathFromPathname(window.location.pathname || '');
  }

  return '';
}

export function withPublicBasePath(path: string, basePath = getPublicBasePath()): string {
  if (!path) return path;

  const normalizedPath = normalizeLeadingSlash(path);
  if (!basePath) return normalizedPath;
  if (normalizedPath === basePath || normalizedPath.startsWith(`${basePath}/`)) {
    return normalizedPath;
  }

  return `${basePath}${normalizedPath}`;
}

export function buildAppApiPath(path: string, basePath = getPublicBasePath()): string {
  return withPublicBasePath(path, basePath);
}
