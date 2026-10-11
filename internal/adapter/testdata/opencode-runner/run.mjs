import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
const [plugin, payloadPath, directory] = process.argv.slice(2);
const { server } = await import(pathToFileURL(plugin).href);
const hooks = await server({ directory, worktree: directory });
const payload = JSON.parse(readFileSync(payloadPath, 'utf8'));
let blocked = false, message = '';
try {
  await hooks['tool.execute.before'](payload.input, payload.output);
} catch (error) {
  blocked = true;
  message = error.message;
}
console.log(JSON.stringify({ blocked, message }));
