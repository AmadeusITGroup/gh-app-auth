export interface C1Node {
  name: string;
  short?: string;
  kind: 'actor' | 'external' | 'system';
}
export interface C1Edge {
  id: string;
  from: string;
  to: string;
  label: string;
  protocol?: string;
  short: string[];
  type: 'local' | 'network';
}
export const nodes: Record<string, C1Node>;
export const edges: C1Edge[];
export const relationshipRows: { from: string; to: string; label: string; protocol?: string }[];
