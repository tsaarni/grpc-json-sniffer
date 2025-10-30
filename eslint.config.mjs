import globals from 'globals';
import pluginJs from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';

export default defineConfig([
  globalIgnores(['public/*.min.js']),
  {
    languageOptions: {
      globals: globals.browser,
    },
    rules: {
      eqeqeq: 'error',
      curly: 'error',
      'no-unused-vars': 'error',
      strict: ['error', 'global'],
      quotes: ['error', 'single'],
      semi: ['error', 'always'],
    },
  },
  pluginJs.configs.recommended,
]);
