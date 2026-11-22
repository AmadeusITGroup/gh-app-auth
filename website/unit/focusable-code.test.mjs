import assert from 'node:assert/strict';
import test from 'node:test';
import focusableCode from '../src/plugins/focusable-code.mjs';

const run = (blockAst) => focusableCode.hooks.postprocessRenderedBlock({ renderData: { blockAst } });
const element = (tagName, properties = {}, children = []) => ({ type: 'element', tagName, properties, children });
const text = (value) => ({ type: 'text', value });

test('plugin is named', () => {
  assert.equal(focusableCode.name, 'Focusable code blocks');
});

test('a root pre gets tabIndex 0 and keeps its attributes and children', () => {
  const code = element('code', {}, [text('gh app-auth setup')]);
  const root = element('pre', { class: 'frame', dataLanguage: 'bash' }, [code]);
  run(root);
  assert.deepEqual(root.properties, { class: 'frame', dataLanguage: 'bash', tabIndex: 0 });
  assert.deepEqual(root.children, [code]);
});

test('a pre inside wrapper elements is found and text nodes are walked safely', () => {
  const pre = element('pre', {}, [text('x')]);
  const root = element('div', { class: 'expressive-code' }, [text('before'), element('figure', {}, [pre]), text('after')]);
  run(root);
  assert.equal(pre.properties.tabIndex, 0);
  assert.equal(root.properties.tabIndex, undefined);
  assert.equal(root.children[1].properties.tabIndex, undefined);
});

test('non-pre roots and siblings are left unchanged', () => {
  const sibling = element('span', { class: 'copy' });
  const root = element('div', {}, [sibling]);
  run(root);
  assert.deepEqual(root.properties, {});
  assert.deepEqual(sibling.properties, { class: 'copy' });
});

test('an existing tabIndex of -1 becomes 0', () => {
  const pre = element('pre', { tabIndex: -1 });
  run(pre);
  assert.equal(pre.properties.tabIndex, 0);
});

test('every pre in a tree is made focusable', () => {
  const first = element('pre');
  const second = element('pre');
  run(element('div', {}, [first, element('section', {}, [second])]));
  assert.equal(first.properties.tabIndex, 0);
  assert.equal(second.properties.tabIndex, 0);
});
