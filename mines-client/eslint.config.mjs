import lightwing from '@lightwing/eslint-config'

export default lightwing(
  {
    ignores: [
      'dist',
      'node_modules',
      '*.svelte',
      '*.snap',
      '*.d.ts',
      'coverage',
      'js_test',
      'local-data',
      'release',
      'api_docs',
      'scripts',
    ],
  },
  {
    rules: {
      'node/prefer-global/process': 'off',
      'no-console': 'off',
      'style/max-statements-per-line': 'off',
    },
  },
  {
    files: ['README.md'],
    rules: {
      'markdown/fenced-code-language': 'off',
    },
  },
)
