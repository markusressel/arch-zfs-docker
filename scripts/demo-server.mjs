#!/usr/bin/env node
// Serves the demo build of the web UI (web/dist-demo, `just build-demo`): the UI with made-up data of a healthy
// repository, without a server behind it (web/src/demo). GitHub Pages publishes the same build next to the docs;
// the screenshots are taken from it (scripts/screenshots.mjs).
//
// Usage: node scripts/demo-server.mjs [port]   (default 8099), then open http://localhost:8099/

import fs from 'node:fs'
import http from 'node:http'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const DIST = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'web', 'dist-demo')
const TYPES = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.svg': 'image/svg+xml', '.woff2': 'font/woff2' }

/** Starts the server; resolves to the http.Server once it listens. Port 0 picks a free port. */
export async function startDemoServer(port = 8099, host = '127.0.0.1') {
  if (!fs.existsSync(path.join(DIST, 'index.html'))) {
    throw new Error(`no demo build in ${path.relative(process.cwd(), DIST)}; run \`just build-demo\` first`)
  }
  const server = http.createServer((req, res) => {
    const rel = new URL(req.url, 'http://demo').pathname.replace(/^\/+/, '') || 'index.html'
    const file = path.join(DIST, path.normalize(rel))
    if (!file.startsWith(DIST) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
      res.writeHead(404).end()
      return
    }
    res.writeHead(200, { 'content-type': TYPES[path.extname(file)] ?? 'application/octet-stream' })
    fs.createReadStream(file).pipe(res)
  })
  await new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(port, host, resolve)
  })
  return server
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const server = await startDemoServer(Number(process.argv[2] ?? 8099)).catch((err) => {
    console.error(`error: ${err.message}`)
    process.exit(1)
  })
  console.log(`Demo UI on http://localhost:${server.address().port}/`)
}
