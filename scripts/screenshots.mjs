#!/usr/bin/env node
// Takes the README screenshots (docs/screenshots/*.png) from a running instance of the web UI.
//
// Usage: node scripts/screenshots.mjs [ui-url]   (default http://localhost:5173/ui/, i.e. `just dev`)
// Needs Firefox (headless screenshots) and ImageMagick (trimming the empty space below the content).
//
// Firefox takes its screenshot on the page's load event, which fires before the UI has fetched its data.
// The UI is therefore loaded in an iframe of a small wrapper page whose load event is held back by an
// image that answers only after SETTLE_MS.

import { execFileSync, spawn, spawnSync } from 'node:child_process'
import fs from 'node:fs'
import http from 'node:http'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const SETTLE_MS = 5000
const WIDTH = 1280
const BACKGROUND = '#0d1117'

// The settings page is left out: it shows the server's absolute repository path.
const PAGES = [
  { name: 'dashboard', route: '/', height: 1600 },
  { name: 'builds', route: '/builds', height: 1600 },
  { name: 'analytics', route: '/analytics', height: 2600 },
]

const uiUrl = new URL(process.argv[2] ?? 'http://localhost:5173/ui/')
const outDir = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'docs', 'screenshots')

function requireTool(cmd) {
  if (spawnSync(cmd, ['--version'], { stdio: 'ignore' }).error) {
    console.error(`error: ${cmd} is required`)
    process.exit(1)
  }
}

async function main() {
  requireTool('firefox')
  requireTool('magick')

  try {
    const res = await fetch(new URL('/api/settings', uiUrl))
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
  } catch (err) {
    console.error(`error: no server answering at ${uiUrl.origin} (${err.message}); start one with \`just dev\``)
    process.exit(1)
  }

  const wrapper = http.createServer((req, res) => {
    const u = new URL(req.url, 'http://wrapper')
    if (u.pathname === '/settle') {
      setTimeout(() => res.writeHead(204).end(), SETTLE_MS)
      return
    }
    const src = new URL(`#${u.searchParams.get('route')}`, uiUrl)
    res.writeHead(200, { 'content-type': 'text/html' }).end(
      `<body style="margin:0;background:${BACKGROUND}">` +
        `<iframe src="${src}" style="border:0;width:100vw;height:100vh;display:block"></iframe>` +
        `<img src="/settle" alt="" style="position:absolute;width:1px;height:1px"></body>`,
    )
  })
  await new Promise((resolve) => wrapper.listen(0, '127.0.0.1', resolve))
  const wrapperUrl = `http://127.0.0.1:${wrapper.address().port}/`

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'arch-zfs-screenshots-'))
  fs.mkdirSync(outDir, { recursive: true })
  try {
    for (const page of PAGES) {
      const raw = path.join(tmp, `${page.name}.png`)
      const profile = fs.mkdtempSync(path.join(tmp, 'profile-'))
      console.log(`==> ${page.name}`)
      // Not spawnSync: the wrapper server runs in this process and must keep answering while Firefox runs.
      await new Promise((resolve, reject) => {
        const ff = spawn('firefox', [
          '--headless',
          '--no-remote',
          '--profile', profile,
          `--window-size=${WIDTH},${page.height}`,
          '--screenshot', raw,
          `${wrapperUrl}?route=${encodeURIComponent(page.route)}`,
        ], { stdio: 'ignore' })
        ff.on('error', reject)
        ff.on('exit', (code) => (fs.existsSync(raw) ? resolve() : reject(new Error(`firefox exited with ${code} and no screenshot`))))
      })
      // Cut the empty background below the content and leave an even margin around it.
      execFileSync('magick', [raw, '-fuzz', '2%', '-trim', '-bordercolor', BACKGROUND, '-border', '40', '+repage', '-strip', path.join(outDir, `${page.name}.png`)])
    }
  } finally {
    wrapper.closeAllConnections()
    wrapper.close()
    fs.rmSync(tmp, { recursive: true, force: true })
  }
  console.log(`Screenshots written to ${path.relative(process.cwd(), outDir)}/`)
}

await main()
