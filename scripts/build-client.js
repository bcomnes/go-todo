import { rm, readdir } from 'node:fs/promises'
import { dirname, join, relative, sep } from 'node:path'
import { build } from 'esbuild'

const routesDirectory = 'pkg/routes'
const outputDirectory = 'pkg/web/dist'

/**
 * @param {string} directory
 * @returns {Promise<string[]>}
 */
async function findPageClients (directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const files = await Promise.all(entries.map(async (entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) return findPageClients(path)
    return entry.isFile() && entry.name === 'page.client.ts' ? [path] : []
  }))
  return files.flat()
}

const pageClients = await findPageClients(routesDirectory)
const entryPoints = { global: 'pkg/web/global.client.ts' }
for (const path of pageClients.sort()) {
  const page = relative(routesDirectory, dirname(path)).split(sep).join('/')
  entryPoints[`pages/${page}`] = path
}

await Promise.all([
  rm(join(outputDirectory, 'global.js'), { force: true }),
  rm(join(outputDirectory, 'pages'), { force: true, recursive: true }),
  rm(join(outputDirectory, 'chunks'), { force: true, recursive: true }),
])

await build({
  entryPoints,
  outdir: outputDirectory,
  entryNames: '[dir]/[name]',
  chunkNames: 'chunks/[name]-[hash]',
  bundle: true,
  splitting: true,
  format: 'esm',
  minify: true,
})
