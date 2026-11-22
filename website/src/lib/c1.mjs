export const nodes = {
  actors: { name: 'Developer or CI job', kind: 'actor' },
  tools: { name: 'Git and GitHub CLI', kind: 'external' },
  auth: { name: 'gh-app-auth', kind: 'system' },
  github: { name: 'GitHub (github.com or Enterprise Server)', short: 'GitHub.com / GHES', kind: 'external' },
  others: { name: 'Other HTTPS Git hosts', kind: 'external' },
};

export const edges = [
  {
    id: 'run',
    from: 'actors',
    to: 'tools',
    label: 'Runs git and gh commands.',
    short: ['runs git', 'and gh'],
    type: 'local',
  },
  {
    id: 'helper',
    from: 'tools',
    to: 'auth',
    label: 'Git runs the credential helper per request; gh runs the extension.',
    protocol: 'local process, stdin and stdout',
    short: ['local helper', 'or extension', 'stdin, stdout'],
    type: 'local',
  },
  {
    id: 'exchange',
    from: 'auth',
    to: 'github',
    label: 'Finds the installation and exchanges a signed JWT for a short-lived installation token. App credentials only.',
    protocol: 'HTTPS REST API',
    short: ['signed JWT for an', 'installation token', 'App only, HTTPS'],
    type: 'network',
  },
  {
    id: 'git',
    from: 'tools',
    to: 'github',
    label: 'Git operations and API calls, authenticated with the credential that gh-app-auth returned or injected.',
    protocol: 'HTTPS',
    short: ['Git and gh calls with the credential (HTTPS)'],
    type: 'network',
  },
  {
    id: 'pat',
    from: 'tools',
    to: 'others',
    label: 'Git operations authenticated with a stored personal access token. No GitHub API is involved.',
    protocol: 'HTTPS, PAT only',
    short: ['Git with a stored PAT (HTTPS, PAT only)'],
    type: 'network',
  },
];

export const relationshipRows = edges.map((edge) => ({
  from: nodes[edge.from].name,
  to: nodes[edge.to].name,
  label: edge.label,
  protocol: edge.protocol,
}));
