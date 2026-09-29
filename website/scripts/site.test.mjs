import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { cp, mkdtemp, readFile, readdir, rm, stat, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { basename, join } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const siteRoot = fileURLToPath(new URL('../', import.meta.url));
const baseURL = new URL('https://modelpack.github.io/modctl/');

async function files(directory, prefix = '') {
  const result = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const name = `${prefix}${entry.name}`;
    if (entry.isDirectory()) result.push(...await files(join(directory, entry.name), `${name}/`));
    else result.push(name);
  }
  return result.sort();
}

function attributes(tag) {
  return Object.fromEntries([...tag.matchAll(/\s([\w:-]+)\s*=\s*(['"])(.*?)\2/gs)]
    .map(([, name, , value]) => [name.toLowerCase(), value.replaceAll('&amp;', '&')]));
}

function decodeHTML(text) {
  const named = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'" };
  return text.replace(/&(amp|lt|gt|quot|apos|#\d+|#x[\da-f]+);/gi, (_, entity) =>
    entity.startsWith('#')
      ? String.fromCodePoint(entity[1].toLowerCase() === 'x' ? parseInt(entity.slice(2), 16) : Number(entity.slice(1)))
      : named[entity.toLowerCase()]);
}

function pageURL(file) {
  return new URL(file.replace(/index\.html$/, ''), baseURL).href;
}

function hugo(args) {
  const result = spawnSync(process.env.HUGO_BINARY || 'hugo', args,
    { cwd: siteRoot, encoding: 'utf8', timeout: 30000 });
  assert.ifError(result.error);
  assert.equal(result.status, 0, `Hugo failed:\n${result.stdout}\n${result.stderr}`);
}

function contentShape(value, path) {
  if (Array.isArray(value)) {
    assert.ok(value.length > 0, `${path}: empty content list`);
    return value.map((entry, index) => contentShape(entry, `${path}[${index}]`));
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).sort(([a], [b]) => a.localeCompare(b))
      .map(([key, entry]) => [key, contentShape(entry, `${path}.${key}`)]));
  }
  assert.notEqual(value, null, `${path}: null content`);
  if (typeof value === 'string') assert.ok(value.trim(), `${path}: empty content`);
  return typeof value;
}

test('every language has matching, nonempty UI translations and content files', async () => {
  const languages = (await readdir(join(siteRoot, 'i18n')))
    .filter((name) => name.endsWith('.json')).map((name) => name.slice(0, -5)).sort();
  assert.ok(languages.includes('en') && languages.includes('zh'));
  const contentLanguages = (await readdir(join(siteRoot, 'content'), { withFileTypes: true }))
    .filter((entry) => entry.isDirectory()).map((entry) => entry.name).sort();
  assert.deepEqual(contentLanguages, languages, 'each language needs both i18n and content');

  const english = JSON.parse(await readFile(join(siteRoot, 'i18n/en.json'), 'utf8'));
  const keys = Object.keys(english).sort();
  const contentFiles = await files(join(siteRoot, 'content/en'));
  assert.ok(keys.length > 0, 'UI translations must not be empty');
  assert.ok(contentFiles.length > 0, 'English content must not be empty');
  for (const language of languages) {
    const messages = JSON.parse(await readFile(join(siteRoot, 'i18n', `${language}.json`), 'utf8'));
    assert.deepEqual(Object.keys(messages).sort(), keys, `${language}: UI translation keys differ`);
    for (const [key, message] of Object.entries(messages)) {
      assert.ok(key.trim(), `${language}: empty translation key`);
      assert.deepEqual(Object.keys(message ?? {}), ['other'], `${language}.${key}: expected { other: string }`);
      assert.equal(typeof message.other, 'string', `${language}.${key}: translation must be text`);
      assert.ok(message.other.trim(), `${language}.${key}: translation must not be empty`);
    }
    const directory = join(siteRoot, 'content', language);
    assert.deepEqual(await files(directory), contentFiles, `${language}: missing corresponding content files`);
    for (const file of contentFiles.filter((name) => name.endsWith('.md'))) {
      assert.ok((await readFile(join(directory, file), 'utf8')).trim(), `${language}/${file}: empty content`);
    }
  }
});

test('translated frontmatter has the same structure and complete homepage sections', async (t) => {
  const temporary = await mkdtemp(join(process.env.JCODE_SCRATCH_DIR || tmpdir(), 'modctl-content-'));
  t.after(() => rm(temporary, { recursive: true, force: true }));
  hugo(['convert', 'toJSON', '--output', temporary]);
  const languages = (await readdir(join(siteRoot, 'content'), { withFileTypes: true }))
    .filter((entry) => entry.isDirectory()).map((entry) => entry.name);
  for (const file of (await files(join(siteRoot, 'content/en'))).filter((name) => name.endsWith('.md'))) {
    let english;
    for (const language of ['en', ...languages.filter((name) => name !== 'en')]) {
      const name = `${language}/${file}`;
      const source = await readFile(join(temporary, name), 'utf8');
      // Hugo's JSON conversion emits an indented object before the Markdown body.
      const frontmatter = source.match(/^\{[\s\S]*?^\}/m)?.[0];
      assert.ok(frontmatter, `${name}: missing frontmatter`);
      const content = JSON.parse(frontmatter);
      const shape = contentShape(content, name);
      if (language === 'en') english = content;
      assert.deepEqual(shape, contentShape(english, `en/${file}`), `${name}: frontmatter structure differs`);
      assert.equal(content.translationKey, english.translationKey, `${name}: translationKey differs`);
      if (file === '_index.md') {
        assert.deepEqual(Object.keys(content.hero ?? {}).sort(), ['description', 'eyebrow', 'lines'], `${name}: hero`);
        assert.equal(content.hero.lines.length, 3, `${name}: hero lines`);
        assert.equal(content.features?.length, 3, `${name}: feature cards`);
        assert.deepEqual(content.workflow?.steps?.map((step) => step.id), ['build', 'push', 'pull'], `${name}: workflow steps`);
      } else {
        assert.ok(source.slice(frontmatter.length).trim(), `${name}: missing Markdown body`);
      }
    }
  }
});

test('Hugo builds a complete multilingual site without warnings', async (t) => {
  const temporary = await mkdtemp(join(process.env.JCODE_SCRATCH_DIR || tmpdir(), 'modctl-hugo-'));
  t.after(() => rm(temporary, { recursive: true, force: true }));
  const output = join(temporary, 'dist');
  hugo([
    '--gc', '--cleanDestinationDir', '--destination', output,
    '--printI18nWarnings', '--panicOnWarning',
  ]);
  const outputFiles = await files(output);
  const htmlFiles = outputFiles.filter((file) => file.endsWith('.html'));

  await t.test('publishes both languages and the shared static assets', () => {
    for (const file of [
      'index.html', 'getting-started/index.html',
      'zh/index.html', 'zh/getting-started/index.html', 'styles.css', 'app.js',
    ]) assert.ok(outputFiles.includes(file), `missing output: ${file}`);
    assert.ok(outputFiles.some((file) => file.startsWith('assets/')), 'missing shared assets');
    for (const file of outputFiles) {
      assert.doesNotMatch(file, /^(?:content\/|i18n\/|layouts\/|scripts\/|package\.json$|README\.md$)/,
        `published source file: ${file}`);
    }
  });

  await t.test('local href, src, CSS URLs, and fragment anchors resolve under /modctl/', async () => {
    for (const file of outputFiles.filter((name) => /\.(?:html|css)$/.test(name))) {
      const source = await readFile(join(output, file), 'utf8');
      const references = file.endsWith('.css')
        ? [...source.matchAll(/url\(\s*['"]?([^'"\s)]+)['"]?\s*\)/g)].map((match) => match[1])
        : [...source.matchAll(/\s(?:href|src)\s*=\s*(['"])(.*?)\1/gs)].map((match) => match[2]);
      for (const reference of references) {
        const target = new URL(reference.replaceAll('&amp;', '&'), pageURL(file));
        if (target.origin !== baseURL.origin) continue;
        assert.ok(target.pathname.startsWith(baseURL.pathname), `${file}: ${reference} escapes /modctl/`);
        let targetPath = decodeURIComponent(target.pathname.slice(baseURL.pathname.length));
        if (!targetPath || targetPath.endsWith('/')) targetPath += 'index.html';
        assert.ok(outputFiles.includes(targetPath), `${file}: missing local link ${reference}`);
        assert.ok((await stat(join(output, targetPath))).isFile(), `${file}: not a file: ${reference}`);
        if (target.hash && targetPath.endsWith('.html')) {
          const targetHTML = await readFile(join(output, targetPath), 'utf8');
          const ids = [...targetHTML.matchAll(/\sid\s*=\s*(['"])(.*?)\1/gs)].map((match) => match[2]);
          assert.ok(ids.includes(decodeURIComponent(target.hash.slice(1))), `${file}: missing anchor ${reference}`);
        }
      }
    }
  });

  await t.test('pages declare their language, canonical URL, and reciprocal translations', async () => {
    for (const file of htmlFiles) {
      const html = await readFile(join(output, file), 'utf8');
      const links = [...html.matchAll(/<link\b[^>]*>/gi)].map((match) => attributes(match[0]));
      const canonicals = links.filter((link) => link.rel === 'canonical');
      assert.equal(canonicals.length, 1, `${file}: expected one canonical URL`);
      assert.doesNotMatch(html, /\bdata-zh\b/i, `${file}: legacy inline translation`);
      const redirect = [...html.matchAll(/<meta\b[^>]*>/gi)].map((match) => attributes(match[0]))
        .find((meta) => meta['http-equiv']?.toLowerCase() === 'refresh');
      if (redirect) {
        const target = redirect.content?.match(/^0;\s*url=(.+)$/i)?.[1];
        assert.ok(target, `${file}: missing alias redirect target`);
        assert.equal(new URL(target, pageURL(file)).href, canonicals[0].href, `${file}: alias canonical`);
        assert.notEqual(canonicals[0].href, pageURL(file), `${file}: alias redirects to itself`);
        continue;
      }
      const language = file.startsWith('zh/') ? 'zh-CN' : 'en';
      assert.equal(attributes(html.match(/<html\b[^>]*>/i)?.[0] ?? '').lang, language, `${file}: html lang`);
      assert.equal(canonicals[0].href, pageURL(file), `${file}: canonical URL`);
      const englishFile = file.replace(/^zh\//, '');
      const translations = { en: englishFile, 'zh-CN': `zh/${englishFile}` };
      for (const [code, translatedFile] of Object.entries(translations)) {
        assert.ok(outputFiles.includes(translatedFile), `${file}: missing ${code} page`);
        const alternate = links.filter((link) => link.rel === 'alternate' && link.hreflang === code);
        assert.equal(alternate.length, 1, `${file}: expected one ${code} alternate`);
        assert.equal(alternate[0].href, pageURL(translatedFile), `${file}: ${code} alternate URL`);
      }
    }
  });

  await t.test('each Markdown code fence renders an accessible copy button targeting its exact code', async () => {
    for (const language of ['en', 'zh']) {
      const markdown = await readFile(join(siteRoot, 'content', language, 'getting-started.md'), 'utf8');
      const expected = [...markdown.matchAll(/^(`{3,}|~{3,})[^\n]*\n([\s\S]*?)^\1\s*$/gm)].map((match) => match[2].trim());
      assert.ok(expected.length > 0, `${language}: missing documentation examples`);
      const html = await readFile(join(output, language === 'en' ? '' : language, 'getting-started/index.html'), 'utf8');
      const blocks = [...html.matchAll(/<div class="code-block">[\s\S]*?<\/code><\/pre>\s*<\/div>/g)].map((match) => match[0]);
      assert.equal(blocks.length, expected.length, `${language}: code render hook must wrap every fence`);
      assert.equal([...html.matchAll(/\sdata-copy-target=/g)].length, blocks.length, `${language}: copy button count`);
      const ids = new Set();
      blocks.forEach((block, index) => {
        const buttons = [...block.matchAll(/<button\b[^>]*>/g)].map((match) => attributes(match[0]));
        assert.equal(buttons.length, 1, `${language}: code block ${index} needs one copy button`);
        const code = block.match(/<code\b([^>]*)>([\s\S]*?)<\/code>/);
        const id = attributes(code?.[1] ?? '').id;
        assert.ok(id && !ids.has(id), `${language}: duplicate or missing code ID`);
        ids.add(id);
        assert.equal(buttons[0]['data-copy-target'], id, `${language}: copy button targets the wrong code`);
        assert.ok(buttons[0]['aria-label']?.trim(), `${language}: copy button needs an accessible label`);
        assert.equal(decodeHTML(code[2]).trim(), expected[index], `${language}: copied code differs from Markdown fence ${index}`);
        assert.doesNotMatch(block, /\shidden(?:\s|=|>)/, `${language}: examples must be readable without JavaScript`);
      });
    }
  });

  await t.test('homepages ship all workflow commands and native language links without JavaScript', async () => {
    const commands = JSON.parse(await readFile(join(siteRoot, 'data/commands.json'), 'utf8'));
    for (const file of ['index.html', 'zh/index.html']) {
      const html = await readFile(join(output, file), 'utf8');
      assert.match(attributes(html.match(/<html\b[^>]*>/)[0]).class, /\bno-js\b/);
      const panels = [...html.matchAll(/<div\b[^>]*\bdata-workflow-panel[^>]*>/g)].map((match) => match[0]);
      assert.equal(panels.length, 3, `${file}: all three workflow panels must be server-rendered`);
      for (const step of ['build', 'push', 'pull']) {
        const panel = panels.find((tag) => attributes(tag).id === `panel-${step}`);
        assert.ok(panel, `${file}: missing ${step} panel`);
        assert.doesNotMatch(panel, /\shidden(?:\s|=|>)|aria-hidden=['"]true/, `${file}: no-JS panel is hidden`);
        const code = html.match(new RegExp(`<code id="workflow-code-${step}">([\\s\\S]*?)<\\/code>`));
        assert.ok(code, `${file}: missing rendered ${step} command`);
        assert.equal(decodeHTML(code[1]).trim(), commands[step].trim(), `${file}: workflow code differs`);
        const tab = [...html.matchAll(/<button\b[^>]*>/g)].map((match) => attributes(match[0]))
          .find((button) => button.id === `tab-${step}`);
        assert.equal(tab?.['aria-controls'], `panel-${step}`, `${file}: tab must target its panel`);
      }
      const languageLinks = [...html.matchAll(/<a\b[^>]*>/g)].map((match) => attributes(match[0]))
        .filter((link) => link.hreflang);
      assert.ok(languageLinks.some((link) => link.href === (file.startsWith('zh/') ? '/modctl/' : '/modctl/zh/')),
        `${file}: language switching requires a native link`);
    }
  });

  await t.test('JavaScript has no client-side language switching or translation table', async () => {
    for (const file of outputFiles.filter((name) => name.endsWith('.js'))) {
      const script = await readFile(join(output, file), 'utf8');
      assert.doesNotMatch(script,
        /\b(?:setLanguage|localeIndex)\b|modctl-language|\bdata-zh\b|document\.documentElement\.lang|\b(?:language|locale)\s*={2,3}\s*['"]/,
        `${file}: translations belong in Hugo content and i18n files`);
    }
  });
});

test('localized SVG labels escape special characters and markup as text', async (t) => {
  const temporary = await mkdtemp(join(process.env.JCODE_SCRATCH_DIR || tmpdir(), 'modctl-svg-'));
  t.after(() => rm(temporary, { recursive: true, force: true }));
  const source = join(temporary, 'site');
  const output = join(temporary, 'dist');
  await cp(siteRoot, source, {
    recursive: true,
    filter: (path) => !['dist', 'resources', '.hugo_build.lock'].includes(basename(path)),
  });
  const keys = ['artifactWeights', 'artifactConfig', 'artifactCode', 'artifactDocs'];
  const payload = 'Weights & <adapters></text><script>alert(1)</script>';
  for (const language of ['en', 'zh']) {
    const path = join(source, 'i18n', `${language}.json`);
    const messages = JSON.parse(await readFile(path, 'utf8'));
    for (const key of keys) messages[key].other = `${key}: ${payload}`;
    await writeFile(path, JSON.stringify(messages));
  }
  hugo(['--source', source, '--destination', output, '--ignoreCache', '--printI18nWarnings', '--panicOnWarning']);
  for (const language of ['en', 'zh']) {
    const svg = await readFile(join(output, 'assets', `artifact.${language}.svg`), 'utf8');
    for (const key of keys) {
      const escaped = `${key}: ${payload}`.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
      assert.ok(svg.includes(escaped), `${language}.${key}: SVG label must preserve and escape text`);
    }
    assert.doesNotMatch(svg, /<adapters>|<script\b/i, `${language}: translated text became an SVG element`);
  }
});
