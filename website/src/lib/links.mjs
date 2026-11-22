const schemePattern = /^[a-z][a-z0-9+.-]*:/i;
const hrefPattern = /^([^?#]*)(\?[^#]*)?(#.*)?$/;

export function isExternalHref(href) {
  return !href || href.startsWith('#') || href.startsWith('//') || schemePattern.test(href);
}

export function splitHref(href) {
  const match = href.match(hrefPattern);
  if (!match) throw new Error(`malformed link "${href}"`);
  const [, rawPath, query = '', hash = ''] = match;
  let decodedPath;
  let anchor;
  try {
    decodedPath = decodeURI(rawPath);
    anchor = hash.length > 1 ? decodeURIComponent(hash.slice(1)) : '';
  } catch {
    throw new Error(`malformed link "${href}"`);
  }
  return { rawPath, decodedPath, query, hash, anchor };
}
