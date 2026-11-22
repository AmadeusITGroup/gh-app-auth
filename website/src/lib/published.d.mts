export const sectionDirs: string[];
export const publishedPatterns: string[];
export function listPublished(docsRoot: string): string[];
export function routeFor(relativePath: string): string;
export function isDraft(file: string): boolean;
