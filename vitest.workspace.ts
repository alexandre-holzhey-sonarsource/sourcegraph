import { readFileSync, existsSync } from 'fs'
import path from 'path'

import glob from 'glob'
import { load } from 'js-yaml'
import { defineWorkspace } from 'vitest/config'

interface PnpmWorkspaceFile {
    packages: string[]
}
const workspaceFile = load(readFileSync(path.join(__dirname, 'pnpm-workspace.yaml'), 'utf8')) as PnpmWorkspaceFile
const projectRoots = filterProjectRootsByArgv(
    workspaceFile.packages.flatMap(p => glob.sync(`${p}/`, { cwd: __dirname })).map(p => p.replace(/\/$/, ''))
).filter(dir => existsSync(path.join(dir, 'vitest.config.ts')))

// If a project root is given as a command-line argument (such as in `pnpm exec vitest client/web`),
// then only load that project root, to speed up init time (from ~3.2s to ~1.9s).
function filterProjectRootsByArgv(projectRoots: string[]): string[] {
    if (!process.argv[1].endsWith('vitest.mjs')) {
        // Only filter when running `vitest`.
        return projectRoots
    }

    const args = process.argv.slice(2)
    const mentionedRoots = projectRoots.filter(root => args.includes(root))
    return mentionedRoots.length > 0 ? mentionedRoots : projectRoots
}

export default defineWorkspace(projectRoots)
