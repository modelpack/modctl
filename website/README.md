# modctl website

The website uses **Hugo 0.167.0's native multilingual support** to generate static pages. GitHub Pages hosts only the build output. There is no client-side translation engine, custom template engine, or custom HTTP server.

## Getting started

Install the standard edition of [Hugo 0.167.0](https://github.com/gohugoio/hugo/releases/tag/v0.167.0) and add `hugo` to your PATH. The Extended edition is not required. Node.js 22+ is used only for npm commands and tests. There are no npm package dependencies, so `npm install` is unnecessary.

```sh
cd website
hugo version
npm run dev
```

Hugo's built-in development server watches files and reloads pages when they change:

- English: <http://localhost:4173/>
- Chinese: <http://localhost:4173/zh/>
- English documentation: <http://localhost:4173/getting-started/>
- Chinese documentation: <http://localhost:4173/zh/getting-started/>

You can also run Hugo directly without npm:

```sh
hugo server --bind localhost --port 4173 --baseURL http://localhost:4173/
```

The server listens on localhost by default. Website tooling does not change the Go application's dependencies.

## Directory structure

```text
website/
├── hugo.toml                   # Base URL, languages, and Markdown rendering
├── content/
│   ├── en/
│   │   ├── _index.md           # English homepage content in front matter
│   │   └── getting-started.md  # English documentation in Markdown
│   └── zh/                    # Corresponding Chinese content
├── i18n/
│   ├── en.json                # English UI, accessibility, and feedback messages
│   └── zh.json                # Chinese translations with the same keys
├── data/commands.json         # CLI examples shared across languages
├── layouts/
│   ├── baseof.html            # HTML shell, language, and SEO metadata
│   ├── home.html              # Shared homepage layout, without language branches
│   ├── docs/single.html       # Shared documentation layout and table of contents
│   ├── _markup/               # Markdown code-block render hook with copy controls
│   └── partials/              # Shared navigation, footer, workflows, and diagrams
├── assets/artifact.svg        # SVG template localized at build time
├── static/                    # CSS, interaction scripts, fonts, and shared assets
└── scripts/site.test.mjs      # Tests using real Hugo builds and generated output
```

### Maintaining multilingual content

- Keep page content in each language's `content` directory and short UI messages in `i18n`. Editing text should not require changes to JavaScript or the HTML layout.
- Use matching relative filenames and `translationKey` values for corresponding pages. Hugo associates the translations natively. Homepage front matter has the same structure in each language, with translated values.
- Hugo's `i18n` function renders UI messages into HTML at build time. Menu and copy controls use the labels already rendered for the current page. They do not detect a language or replace page content in the browser.
- Language navigation uses `.Translations` to generate ordinary links to the current page's translations. Content, code examples, and language links remain available with JavaScript disabled.
- The URL determines the language. English is served at the root and Chinese at `/zh/`. The site does not use browser-language redirects, cookies, or localStorage. The prototype's `?lang=zh` parameter no longer selects a language. Use `/zh/` instead.
- HTML `lang`, canonical URLs, `hreflang` links, and sitemaps are generated at build time. To add a language, update `hugo.toml`, add corresponding `content/<language>/` and `i18n/<language>.json` files, and extend the current bilingual acceptance tests.
- Keep CLI examples consistent with the repository's `docs/getting-started.md`. Illustrations and fonts are served locally. Font licenses are included in `static/assets/*-LICENSE.txt`.

### JavaScript responsibilities

`static/app.js` handles only the mobile menu, terminal tabs, clipboard controls, and table-of-contents highlighting. All localized page and terminal content is generated at build time. With JavaScript disabled, all workflow examples remain visible and inactive copy buttons are hidden.

## Testing and building

```sh
npm test
npm run build
```

Build output is written to `dist/`, including separate language pages, shared assets, and sitemaps. Hosting the generated files requires neither Node.js nor Hugo nor a backend service. Use the same Hugo version as CI, which verifies the official release archive against a pinned SHA-256 checksum.

Tests use Node's built-in test runner and invoke Hugo directly rather than maintaining a separate rendering or translation implementation. They check:

- Matching translation keys, nonempty messages, corresponding content files, and front matter structure across languages.
- Build failures for missing translations and other warnings instead of silently hiding missing translations through fallback behavior.
- Generated files, resource links and anchors under `/modctl/`, canonical URLs, and reciprocal `hreflang` links.
- Copy buttons generated from Markdown code blocks and their corresponding code element IDs.
- The absence of legacy `data-zh` attributes, client-side language-switching functions, and language storage.

If Hugo is not on your PATH, set `HUGO_BINARY=/absolute/path/to/hugo` when running tests. Development and build commands invoke `hugo` directly, without a custom installer or wrapper.

Browser regression checks should cover both pages in both languages, viewport widths from 320px to 1920px, JavaScript-disabled access, language navigation, copying, keyboard tab controls, mobile navigation, and reduced motion. Automated accessibility checks do not replace visual and maintainability reviews.

## GitHub Pages

The configured site URL is <https://modelpack.github.io/modctl/>. Chinese pages are served at <https://modelpack.github.io/modctl/zh/>.

`.github/workflows/website.yml` tests and builds pull requests. It uploads `website/dist/` and deploys only when website or workflow changes reach `main` in the upstream `modelpack/modctl` repository, or when manually triggered on `main`. Forks do not deploy automatically.

Initial setup:

1. Select **GitHub Actions** in [Settings → Pages](https://github.com/modelpack/modctl/settings/pages).
2. Merge into `main` using the repository's existing review process. Do not bypass branch protection.
3. Check results in [Actions → Website](https://github.com/modelpack/modctl/actions/workflows/website.yml). To redeploy, select **Run workflow** on `main`.
4. The `github-pages` environment permits deployments only from `main`. If environment reviewers are configured, their approval is also required.

The build job has read-only permissions. The deploy job's `pages: write` and `id-token: write` permissions are scoped to GitHub Pages deployment. No additional PAT, CNAME, or `gh-pages` branch is needed. Deployments are serialized without interrupting an in-progress deployment.

Hugo's `baseURL` is `https://modelpack.github.io/modctl/`. Templates use `.RelPermalink` and `relURL` rather than hard-coding `/modctl/`. The development command overrides this with a local URL. To use another domain or subpath, update `baseURL` and the test target.

## Design references

The design uses a cool-white, ink-blue, and blue palette with layered model-artifact illustrations. Content organization draws on Cilium, Envoy, and Harbor. Multilingual project organization follows the [Kubernetes website](https://github.com/kubernetes/website/blob/main/hugo.toml) and [Hugo's multilingual documentation](https://gohugo.io/content-management/multilingual/). No external project's page code or endorsement is reused.
