import assert from 'node:assert/strict';
import fs from 'node:fs';
import test from 'node:test';
import YAML from 'yaml';

const workflow = YAML.parse(fs.readFileSync(new URL('../../.github/workflows/docs.yml', import.meta.url), 'utf8'));
const normalize = (value) => String(value).replace(/\s+/g, ' ').trim();
const publishCondition =
  "github.ref == 'refs/heads/main' && ((github.event_name == 'push' && github.repository == 'AmadeusITGroup/gh-app-auth') || (github.event_name == 'workflow_dispatch' && inputs.publish))";

test('validation-only manual runs use a separate concurrency group from publication', () => {
  assert.equal(
    workflow.concurrency.group,
    "documentation-${{ github.ref }}-${{ github.event_name == 'workflow_dispatch' && !inputs.publish && 'validation-only' || 'publish' }}",
  );
  assert.equal(workflow.concurrency['cancel-in-progress'], "${{ github.ref != 'refs/heads/main' }}");
});

test('permissions stay read-only globally and Pages-scoped for publication', () => {
  assert.deepEqual(workflow.permissions, { contents: 'read' });
  const publish = Object.values(workflow.jobs).find((job) => job.permissions && job.permissions.pages);
  assert.deepEqual(publish.permissions, { pages: 'write', 'id-token': 'write' });
});

test('the Pages artifact upload and the publish job share the exact publication condition', () => {
  const build = Object.values(workflow.jobs).find((job) => job.steps?.some((step) => /upload-pages-artifact/.test(step.uses ?? '')));
  const upload = build.steps.find((step) => /upload-pages-artifact/.test(step.uses ?? ''));
  const publish = Object.values(workflow.jobs).find((job) => job.permissions && job.permissions.pages);
  assert.equal(normalize(upload.if), publishCondition);
  assert.equal(normalize(publish.if), publishCondition);
});
