export function isExternalHref(href: string): boolean;
export function splitHref(href: string): {
  rawPath: string;
  decodedPath: string;
  query: string;
  hash: string;
  anchor: string;
};
