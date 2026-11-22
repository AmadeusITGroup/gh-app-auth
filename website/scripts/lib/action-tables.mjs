const IDENTIFIER = /^`([A-Za-z_][A-Za-z0-9_-]*)`$/;

const splitRow = (line) =>
  line
    .trim()
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split(/(?<!\\)\|/)
    .map((cell) => cell.trim());

const tableRows = (text) => text.split('\n').filter((line) => line.trim().startsWith('|')).map(splitRow);

const isSeparator = (cells) => cells.every((cell) => /^:?-+:?$/.test(cell));

const identifiers = (rows) =>
  new Set(rows.filter((cells) => !isSeparator(cells)).map((cells) => IDENTIFIER.exec(cells[0])?.[1]).filter(Boolean));

const unwrap = (cell) => {
  const match = /^`(.*)`$/.exec(cell);
  return match ? match[1] : cell;
};

function compare(name, kind, actual, documented) {
  const problems = [];
  for (const key of actual) if (!documented.has(key)) problems.push(`${name}: ${kind} "${key}" is not documented`);
  for (const key of documented) {
    if (!actual.includes(key)) problems.push(`${name}: documented ${kind} "${key}" does not exist in action.yml`);
  }
  return problems;
}

function checkDefaults(name, inputs, rows) {
  const problems = [];
  const withDefaults = Object.entries(inputs).filter(([, definition]) => 'default' in definition && String(definition.default) !== '');
  if (withDefaults.length === 0) return [];
  const defaultIndex = (rows[0] ?? []).indexOf('Default');
  if (defaultIndex === -1) return [`${name}: inputs table has no Default column`];
  for (const [input, definition] of withDefaults) {
    const expected = String(definition.default);
    const row = rows.find((cells) => IDENTIFIER.exec(cells[0])?.[1] === input);
    if (!row || row.length <= defaultIndex || unwrap(row[defaultIndex]) !== expected) {
      problems.push(`${name}: default for "${input}" should be ${expected}`);
    }
  }
  return problems;
}

export function checkActionTables(name, metadata, inputsText, outputsText) {
  const inputs = metadata.inputs ?? {};
  const inputRows = tableRows(inputsText);
  return [
    ...compare(name, 'input', Object.keys(inputs), identifiers(inputRows)),
    ...compare(name, 'output', Object.keys(metadata.outputs ?? {}), identifiers(tableRows(outputsText))),
    ...checkDefaults(name, inputs, inputRows),
  ];
}
