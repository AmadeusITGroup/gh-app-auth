function makeFocusable(node) {
  if (node.type === 'element' && node.tagName === 'pre') {
    node.properties = { ...node.properties, tabIndex: 0 };
  }
  if (Array.isArray(node.children)) {
    for (const child of node.children) makeFocusable(child);
  }
}

export default {
  name: 'Focusable code blocks',
  hooks: {
    postprocessRenderedBlock({ renderData }) {
      makeFocusable(renderData.blockAst);
    },
  },
};
