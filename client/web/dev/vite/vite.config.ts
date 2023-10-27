import path from 'path'

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

import { ENVIRONMENT_CONFIG } from '../utils/environment-config'

import { manifestPlugin } from './manifestPlugin'

const clientWebRoot = path.join(__dirname, '../..')

export default defineConfig({
    plugins: [react(), manifestPlugin({ fileName: 'vite-manifest.json' })],
    build: {
        rollupOptions: {
            input: ENVIRONMENT_CONFIG.CODY_APP
                ? ['src/enterprise/app/main.tsx']
                : ['src/enterprise/main.tsx', 'src/enterprise/embed/embedMain.tsx'],
        },
        sourcemap: true,

        // modulepreload is supported widely enough now (https://caniuse.com/link-rel-modulepreload)
        // and is only relevant for local dev.
        modulePreload: { polyfill: false },

        emptyOutDir: false, // client/web/dist has static assets checked in
    },
    root: clientWebRoot,
    publicDir: 'dist',
    assetsInclude: ['**/*.yaml'],
    define: {
        ...Object.fromEntries(
            Object.entries({ ...ENVIRONMENT_CONFIG, SOURCEGRAPH_API_URL: undefined }).map(([key, value]) => [
                `process.env.${key}`,
                JSON.stringify(value === undefined ? null : value),
            ])
        ),
    },
    optimizeDeps: {
        exclude: [
            // Without addings this Vite throws an error
            'linguist-languages',
        ],
    },
    resolve: {
        alias: {
            path: require.resolve('path-browserify'),
        },
        mainFields: ['browser', 'module', 'main'],
    },
    css: {
        preprocessorOptions: {
            scss: {
                includePaths: [
                    // Our scss files and scss files in client/* often import global styles via @import 'wildcard/src/...'
                    // Adding '..' as load path causes scss to look for these imports in the client folder.
                    // (without it scss @import paths are always relative to the importing file)
                    path.join(clientWebRoot, '..'),
                ],
            },
        },
        modules: {
            localsConvention: 'camelCaseOnly',
        },
    },
})
