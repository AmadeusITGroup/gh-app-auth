/// <reference types="astro/client" />

declare module 'virtual:starlight/project-context' {
  const project: { trailingSlash: 'always' | 'never' | 'ignore' };
  export default project;
}

declare module 'virtual:starlight/pagefind-config' {
  export const pagefindUserConfig: Record<string, unknown>;
}
