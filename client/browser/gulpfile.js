// @ts-check

const { spawn } = require('child_process')

function build() {
  return spawn('pnpm', ['run', 'build'], {
    stdio: 'inherit',
    shell: true,
    env: { ...process.env, NODE_OPTIONS: '--max_old_space_size=8192' },
  })
}

function watch() {
  return spawn('pnpm', ['run', 'dev'], {
    stdio: 'inherit',
    shell: true,
    env: { ...process.env, NODE_OPTIONS: '--max_old_space_size=8192' },
  })
}

module.exports = { build, watch }
