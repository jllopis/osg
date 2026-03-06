# DESIGN - OSG (Obsidian Site Generator)

## Overview
OSG es un generador de contenido estatico a partir de un vault de Obsidian. El core en Go 1.25 sincroniza Markdown con frontmatter YAML hacia `content/`, y luego renderiza HTML en `public/` con templates, taxonomias (feeds atom/rss opcionales), sitemap y assets.

El sistema soporta themes, plugins WASM para extender el pipeline y un modo TUI para orquestar tareas y ver logs.

## Architecture
Componentes principales:

- cmd/osg: CLI y wiring
- internal/app: comandos (init, update-content, build, serve, tui)
- internal/config: carga y merge de config (YAML + env + flags)
- internal/vault: descubrimiento de archivos, lectura e indice de imagenes
- internal/frontmatter: parse YAML y split frontmatter/body
- internal/publish: filtro publish + drafts, extraccion del bloque `osg`
- internal/site: modelos y indexado del sitio (Page, Section, Site)
- internal/content: normalizacion de frontmatter y writer a `content/`
- internal/build: pipeline de render, generacion de placeholders y output
- internal/render: templates, helpers, builtins y overrides
- internal/taxonomy: indices, paginacion y feeds por taxonomia
- internal/assets: static + sass
- internal/placeholder: generacion determinista de imagenes SVG placeholder (Nord)
- internal/summary: auto-generacion de summaries (Provider interface, PlainText, truncate, Kairos AI)
- internal/wikilink: deteccion y reescritura de wikilinks de imagen Obsidian
- internal/plugin: host WASM y hooks
- internal/tui: UI con Bubble Tea
- internal/i18n: carga de traducciones YAML, funcion trans() y date_format() para plantillas
- internal/theme: theme embebido por defecto con EnsureDefaultTheme

## Data flow

### update-content
1. Load config
2. Discover vault: listar archivos `.md` via `vault.ListMarkdownFiles`
3. Build image index: `vault.BuildImageIndex(vaultPath)` indexa todas las imagenes por basename y path relativo
4. Para cada archivo:
   a. Parse frontmatter + split body
   b. Filtrar con `publish.ShouldPublish` (lee `osg.publish` > `publish` top-level)
   c. Derivar fecha y slug
   d. Normalizar frontmatter via `content.NormalizeFrontmatter` (lee bloque `osg` para image/featured)
   e. Resolver imagen de frontmatter: URLs externas intactas, paths del vault resueltos via `imageIndex.Resolve`, copiados con ruta absoluta
   f. Reescribir wikilinks de imagen en body: `wikilink.RewriteImageLinks` convierte `![[file.png|alt]]` a `![alt](file.png)`, resolviendo y copiando imagenes
   g. Escribir markdown normalizado a `content/`

### build
1. Load config
2. Parsear archivos `content/` -> `site.ParseFile` (lee `osg.image`, `osg.featured` con fallbacks)
3. Indexar sitio: `Site.BuildHierarchy()` asigna pages a sections, ordena por fecha. Las pages con `menu: true` se excluyen de sections y del root (solo accesibles via `menu_pages`)
4. Generar summaries: `fillSummaries()` usa `summary.Provider` para pages sin summary (segun `summary_strategy`)
5. Generar placeholders: `generatePlaceholders()` crea SVG para pages sin imagen
6. Build taxonomias
7. Render templates -> `public/`
8. Copiar assets (static/theme static + sass si aplica)
9. Plugins: hooks en `build.started`/`build.finished`
10. Build incremental: cache de inputs para saltar renders sin cambios

### serve/tui
Previsualizacion y control del flujo con watch + live reload opcional.

## Bloque `osg` en frontmatter

OSG soporta un bloque `osg` en el frontmatter YAML para controlar la publicacion y metadatos sin interferir con otros campos de Obsidian:

```yaml
---
title: Mi Post
tags:
  - filosofia
osg:
  publish: true       # true | "draft" | false
  featured: true      # mostrar como destacado en homepage
  image: "foto.jpg"   # imagen de cabecera (nombre o path relativo en vault)
  path: "about"       # override de content_layout -> /about/ en vez de /YYYY/MM/DD/slug/
  menu: true          # incluir como enlace en la navegacion del sitio
  abstract: "Resumen escrito a mano que aparece en listados, meta description y OG tags."
  author: "Joan Llopis"  # autor del post, mostrado junto a la fecha
---
```

### Prioridades de resolucion
- **publish**: `osg.publish` > `publish` top-level
- **image**: `osg.image` > `image` > `cover` > `banner` top-level
- **featured**: `osg.featured` > `featured` top-level
- **abstract**: `osg.abstract` > `summary` > `description` > `excerpt` top-level. Si ninguno esta presente, se aplica la estrategia configurada en `summary_strategy` (auto/ai)
- **author**: `osg.author` > `author` top-level. Se muestra en formato **Autor** &bull; fecha en articulos y listados
- **path**: solo `osg.path` (sin fallback top-level)
- **menu**: solo `osg.menu` (sin fallback top-level)
- **permalink**: solo `osg.permalink` (sin fallback top-level). Precedencia sobre `osg.path` y `content_layout`

### Permalinks configurables

OSG soporta placeholders extendidos tanto en `content_layout` (config global) como en `osg.permalink` (per-page):

**Placeholders disponibles:**

| Placeholder | Descripcion | Ejemplo |
|-------------|-------------|---------|
| `{date}` | Fecha completa YYYY/MM/DD | `2025/03/06` |
| `{year}` | Ano 4 digitos | `2025` |
| `{month}` | Mes 2 digitos | `03` |
| `{day}` | Dia 2 digitos | `06` |
| `{slug}` | Slug derivado del titulo/frontmatter | `mi-post` |
| `{title}` | Titulo slugificado (via `slug.Slugify()`) | `mi-titulo-largo` |

**Precedencia de URL:**

1. `osg.permalink` (maxima precedencia, soporta placeholders)
2. `osg.path` (URL fija, tambien setea `menu_title`)
3. `content_layout` de config (patron global)

**Diferencia entre `osg.permalink` y `osg.path`:**

- `osg.permalink` es **puramente URL**: no tiene side-effects. Soporta placeholders como `{year}`, `{slug}`, etc.
- `osg.path` ademas setea `menu_title` cuando `osg.menu=true`. No soporta placeholders.

**Ejemplos en frontmatter:**

```yaml
---
title: Mi Articulo Especial
osg:
  publish: true
  permalink: "blog/{year}/{slug}"   # -> /blog/2025/mi-articulo-especial/
---
```

```yaml
---
title: Guia de Instalacion
osg:
  publish: true
  permalink: "docs/{title}"         # -> /docs/guia-de-instalacion/
---
```

**Ejemplos de `content_layout` en config.yaml:**

```yaml
# Layout por defecto (fecha completa + slug)
content_layout: "{date}/{slug}"           # -> /2025/03/06/mi-post/

# Solo ano y slug
content_layout: "{year}/{slug}"           # -> /2025/mi-post/

# Prefijo blog + titulo
content_layout: "blog/{year}/{month}/{title}"  # -> /blog/2025/03/mi-titulo/

# Flat: solo slug
content_layout: "{slug}"                  # -> /mi-post/
```

**Resolucion de titulo para `{title}`:**

El placeholder `{title}` usa la misma cadena de precedencia que el titulo de pagina: `osg.title` > `fm.title` > `fm.name` > filename. El titulo se pasa por `slug.Slugify()` para generar URLs validas.

### Paginas standalone (osg.path + osg.menu)

Cuando una nota tiene `osg.path`, el output se escribe en `content/{path}/index.md` en lugar de seguir el `content_layout` por defecto. Esto permite URLs limpias como `/about/` o `/contacto/`.

Si ademas tiene `osg.menu: true`, la pagina:
- Se incluye en la variable de template `menu_pages` (disponible en todas las plantillas)
- Se **excluye** del listado de posts en homepage/secciones (no aparece en `section.pages`)
- Aparece como enlace en el `<nav>` del header junto a las taxonomias

### Valores de publish
- `true` (bool) o `"true"` (string): publicar normalmente
- `"draft"` (string, case-insensitive): publicar como borrador (visible solo con `--drafts`)
- Cualquier otro valor: no publicar

### Draft preview mode

El flag `--drafts` en `osg serve` permite previsualizar borradores sin publicarlos:

```bash
osg serve --drafts
```

Comportamiento:
- Las notas con `publish: "draft"` se incluyen en el build y son navegables
- Cada pagina draft muestra un **banner rojo** indicando que es un borrador
- En listados (homepage, secciones, cards) aparece un **badge "Draft"** junto a la fecha
- Los drafts se **excluyen** de feeds RSS/Atom y sitemap (no se filtran al exterior)
- El search index puede contener drafts (son visibles en preview, es intencional)
- `page.draft` esta disponible en templates como boolean para logica condicional

## Image pipeline

### En update-content
1. **Indice de imagenes** (`internal/vault`): `BuildImageIndex` recorre el vault e indexa toda imagen por basename y path relativo al vault. Extensiones: `.png`, `.jpg`, `.jpeg`, `.gif`, `.svg`, `.webp`, `.bmp`, `.avif`.
2. **Imagen de frontmatter**: si no es URL externa ni path absoluto, se resuelve via `ImageIndex.Resolve` (primero path exacto, luego basename). Si se encuentra, se copia al directorio de salida y se establece la ruta absoluta (ej. `/2025/09/16/mi-post/foto.jpg`). Si no se encuentra, se borra el campo para que el generador de placeholders actue.
3. **Wikilinks en body** (`internal/wikilink`): `RewriteImageLinks` detecta `![[imagen.png|texto alt]]` y los convierte a Markdown estandar `![texto alt](imagen.png)`, resolviendo y copiando las imagenes. Las rutas se codifican con `url.PathEscape` para nombres con espacios.

### En build
4. **Placeholders** (`internal/placeholder`): para toda page sin `Image`, se genera un SVG determinista (1200x630, paleta Nord, patron geometrico basado en SHA-256 del titulo). Se escribe en `public/img/placeholder-{hash}.svg` y se asigna a `page.Image`.

### Flujo de resolucion de Image
```
Vault note -> osg.image / image / cover / banner -> resolve via ImageIndex -> copy to content/
                                                                                    |
content/ -> site.ParseFile -> Page.Image -> if empty -> placeholder SVG -> public/img/
                                         -> if set -> used in templates (hero, thumbnail, og:image)
```

## Image lightbox y galeria

### Lightbox

Las imagenes standalone en el contenido (Markdown `![alt](src)` que son unico hijo de un parrafo) se envuelven automaticamente en `<figure data-lightbox>` con `<figcaption>` opcional (derivada del alt text). El renderer custom (`internal/markdown/figure.go`) sustituye el `<p><img>` por defecto de Goldmark.

Caracteristicas:
- **Click-to-zoom**: las imagenes con `data-lightbox` abren un overlay fullscreen Nord-styled
- **Navegacion**: flechas izquierda/derecha, swipe tactil en movil, teclado (Esc, Left, Right)
- **Captions**: texto alternativo mostrado como leyenda debajo de la imagen ampliada
- **Counter**: indicador N/M cuando hay multiples imagenes en la pagina
- **Accesible**: `role="dialog"`, `aria-modal`, labels en botones, respeta `prefers-reduced-motion`
- **Zero-dependency**: ~120 lineas de JS vanilla, sin librerias externas

### Galeria automatica

Las `<figure>` consecutivas (sin texto entre ellas) se agrupan automaticamente en un `<div class="gallery">` con CSS grid responsive:
- Grid auto-fill con columnas de minimo 280px
- Imagenes con `aspect-ratio: 4/3` y `object-fit: cover` para uniformidad
- En movil: columna unica

### Config

```yaml
lightbox: true   # default: true, habilita lightbox JS en pages
```

`lightbox: false` desactiva el script JS en las paginas (las imagenes siguen renderizandose con `<figure>` pero sin overlay).

### Imagenes inline vs standalone

- **Standalone**: `![alt](src)` como unico contenido de un parrafo → `<figure data-lightbox>` con lightbox
- **Inline**: `texto ![icon](src) mas texto` → `<img>` dentro de `<p>`, sin figure ni lightbox

### Ficheros

- `internal/markdown/figure.go`: renderers custom (figureImageRenderer, figureParagraphRenderer)
- `internal/theme/default/static/js/lightbox.js`: JS de lightbox (//go:embed via theme)
- `internal/theme/default/static/style.css`: CSS de lightbox, galeria y figure

## Featured posts (multiples)

Cuando multiples posts tienen `osg.featured: true` (o `featured: true` en frontmatter):

1. El post featured **mas reciente** por fecha se convierte en el hero de la homepage (`featured_page` en el contexto de Section)
2. Los demas posts featured se promueven al **inicio** de la lista de posts (antes que los no-featured), manteniendo orden por fecha
3. Si ningun post es featured, el mas reciente de la seccion se usa como hero

La logica esta en `Section.View()` con el helper `isFeaturedPage()`.

## Summary pipeline

`internal/summary/` provee auto-generacion de summaries para pages que no tienen uno en frontmatter.

### Provider interface
- `Provider.Summarize(rawContent string, maxLen int) string`
- `ExtractProvider`: extrae primera oracion significativa del markdown body
- `NoopProvider`: no genera summary (para `summary_strategy: manual`)

### Estrategias (`summary_strategy` en config)
- `auto` (default): usa ExtractProvider — strip markdown con PlainText(), trunca a 160 chars en limite de oracion/palabra
- `manual`: usa NoopProvider — solo summaries puestos a mano en frontmatter
- `ai`: usa KairosProvider con el LLM configurado en la seccion `ai` del config. Si falla la creacion del provider (e.g. API key no disponible), cae a `auto` con un warning en el log

### PlainText()
Elimina formateo markdown: headings, bold/italic (6 regexes por incompatibilidad RE2 con backreferences), links, images, code blocks, inline code, blockquotes, HR, listas. Colapsa whitespace.

### truncateSentence()
Corta en el ultimo `.`/`!`/`?` antes de maxLen. Si no hay puntuacion, corta en el ultimo espacio y agrega `...`.

### Flujo en build
```
Site.BuildHierarchy() -> fillSummaries(opts) -> generatePlaceholders()
                         |
                         opts.SkipAI == true (serve mode):
                           log "skipping AI summaries"
                           fallback a auto strategy
                         strategy == "ai":
                           load AI cache (.osg/cache/ai-summaries.json)
                           NewKairosProvider(ctx, aiCfg) con Language del config
                           fillWithAI(): para cada page:
                             - hash = SHA-256(content)
                             - cache hit? usar summary cacheado
                             - cache miss? LLM call -> store en cache
                           save AI cache
                           si opts.ForceAISummaries: ignorar cache, regenerar todo
                           si falla creacion provider: fallback a auto
                         strategy == "auto":
                           fillWithProvider() secuencial con ExtractProvider
                         strategy == "manual":
                           NoopProvider (nada que hacer)
```

### KairosProvider (AI summaries)

`internal/summary/kairos.go` implementa generacion de summaries via LLM usando Kairos como backend multi-provider.

**Providers soportados**: gemini (default), anthropic, openai, qwen, ollama.

**Arquitectura**:
- `KairosProvider` struct wrapping `llm.Provider` de Kairos
- `Summarize(ctx, title, rawMarkdown)`: strip markdown con PlainText(), envia system prompt + user content (titulo + texto plano), temperature 0.3
- `NewKairosProvider(ctx, AIConfig)`: factory que crea el provider correcto segun configuracion

**Flujo de un request AI**:
```
page.RawContent -> PlainText() -> "Title: {title}\n\n{plain text}"
                                       |
                                       v
                              LLM.Chat(system prompt + user content)
                                       |
                                       v
                              TrimSpace(response.Content) -> page.Summary
```

**Concurrency** (`fillWithAI` en build.go):
- Semaphore channel de tamano `ai.concurrency` (default 3)
- Cada goroutine adquiere slot, genera summary, libera slot
- Per-request timeout via `context.WithTimeout` (default 30s)
- Errores se logean como warnings, no interrumpen el batch

### Cache de summaries AI

`internal/build/ai_cache.go` implementa un cache persistente para evitar regenerar summaries AI en cada build.

**Ubicacion**: `.osg/cache/ai-summaries.json` (dentro de `build_cache_dir`).

**Clave**: SHA-256 del contenido markdown crudo (sin frontmatter). Si el contenido cambia, el hash cambia y se regenera el summary.

**Entrada** (`AICacheEntry`):
- `summary`: el texto generado
- `provider`: nombre del provider usado (e.g. "gemini")
- `model`: modelo usado (e.g. "gemini-3-flash-preview")
- `generated_at`: timestamp ISO 8601

**Flujo**:
```
fillSummaries() -> loadAICache(path)
                -> fillWithAI(): para cada page sin summary:
                     hash = SHA-256(rawContent)
                     if cache.Lookup(hash) -> usar summary cacheado (log "cache hit")
                     else -> llamar LLM -> cache.Store(hash, entry)
                -> saveAICache(cache, path)
```

**Invalidacion**:
- Cambio de contenido: automatica (hash diferente)
- Cambio de provider/modelo: NO automatica. Usar `--force-ai-summaries` para regenerar todo
- `--force-ai-summaries`: ignora el cache y regenera todos los summaries. Requiere confirmacion interactiva (bypass con `--yes`/`-y`)

**Thread-safety**: `AICache` usa `sync.RWMutex` para acceso concurrente seguro.

### Prompts con idioma

El system prompt por defecto inyecta el idioma configurado en `default_language`:

- Si `default_language` esta definido (e.g. "es"), el prompt incluye "Write the summary in Spanish."
- Si no hay `default_language`, el prompt no menciona idioma
- Si se define un `system_prompt` custom en la config AI, se usa tal cual sin inyectar idioma
- `langDisplayName()` mapea codigos BCP-47 a nombres en ingles: es->Spanish, en->English, fr->French, de->German, pt->Portuguese, it->Italian, ca->Catalan, etc. Codigos desconocidos se pasan tal cual

### Aislamiento de serve

`osg serve` ahora establece `SkipAI=true` para todos los builds (inicial y por watch/rebuild):

- En modo serve, las pages sin summary reciben fallback automatico a estrategia `auto` (extraccion de primeras oraciones)
- Esto evita requests LLM costosos y lentos durante desarrollo
- El build normal (`osg build`) sigue usando la estrategia AI configurada
- El TUI (`osg tui` -> build) tampoco salta AI, solo serve

### CLI flags de build

- `--force-ai-summaries`: regenera todos los summaries AI ignorando el cache. Requiere confirmacion interactiva
- `--yes` / `-y`: bypass de confirmacion para `--force-ai-summaries` (util en CI/scripts)

## Featured overlay (CSS)

El post destacado en homepage muestra la imagen con un overlay de texto:

## i18n (internacionalizacion de plantillas)

`internal/i18n/` provee traduccion de cadenas UI en las plantillas del tema. Ademas, desde Phase 15, OSG soporta contenido multi-idioma: paginas con el mismo slug en distintos idiomas se vinculan automaticamente como traducciones, con hreflang alternates, selector de idioma y feeds/sitemap multi-idioma.

### Arquitectura

- **Ficheros de traduccion**: `i18n/{lang}.yaml` — YAML plano clave-valor. Se cargan primero del directorio del tema, luego del directorio del usuario (el usuario puede override).
- **Bundle**: struct que almacena traducciones para multiples idiomas, con idioma por defecto.
- **Funciones de template**:
  - `trans(key, lang?)`: busca la traduccion para la clave. Fallback: idioma solicitado -> idioma por defecto -> clave cruda.
  - `date_format(time, layout, lang?)`: formatea una fecha con Go time layout y reemplaza nombres de mes en ingles por los equivalentes localizados (es/fr/de/pt/it/ca).
- **Config**: `default_language` (default `"es"`). Se pasa al contexto de render como `lang`.

### Ficheros incluidos en tema por defecto

- `i18n/en.yaml`: ~31 claves (UI, 404, ARIA, feeds, formatos de fecha)
- `i18n/es.yaml`: equivalente en castellano

### Flujo en build

```
Load config.default_language
  -> cargar i18n del tema (themes/{theme}/i18n/*.yaml)
  -> cargar i18n del usuario (i18n/*.yaml, override)
  -> crear i18n.Bundle con idioma por defecto
  -> inyectar en render.Context (I18n, DefaultLanguage)
  -> trans() y date_format() disponibles en todas las plantillas
```

### Dual-file sync

Los ficheros YAML de traduccion existen en DOS ubicaciones que deben mantenerse sincronizadas:
1. `internal/theme/default/i18n/` — fuente embebida (//go:embed)
2. `themes/default/i18n/` — copia runtime

### Multi-idioma (Phase 15)

OSG soporta contenido en multiples idiomas. El idioma por defecto se configura con `default_language` y los secundarios con `languages`:

```yaml
default_language: es
languages:
  - code: en
    label: English
  - code: fr
    label: Francais
```

**Deteccion de idioma**: cada nota del vault especifica `lang: en` (o equivalente) en su frontmatter. Si no se especifica, se asume `default_language`.

**URL prefix**: el idioma por defecto NO tiene prefijo (`/mi-post/`). Los idiomas secundarios tienen prefijo `/{lang}/` (`/en/mi-post/`). Esto mantiene compatibilidad con URLs existentes.

**Translation linking**: paginas con el mismo `Slug` pero distinto `Lang` se vinculan automaticamente como traducciones via `LinkTranslations()`. La vinculacion se realiza despues de `BuildHierarchy()`.

**Templates**: todas las llamadas `{{ trans "key" }}` reciben `.lang` como segundo argumento (`{{ trans "key" .lang }}`), lo que permite que cada pagina se renderice en su propio idioma.

**SEO**:
- `<link rel="alternate" hreflang>` en `<head>` para paginas con traducciones
- `x-default` apunta a la version en idioma por defecto
- `og:locale` usa el idioma real de la pagina
- Sitemap incluye `<xhtml:link rel="alternate" hreflang>` para paginas traducidas

**Feeds**: `xml:lang` en Atom `<feed>`, `<language>` en RSS `<channel>`.

**Language switcher**: nav element en el header que muestra el idioma actual (bold) y links a las traducciones. Solo aparece en paginas que tienen traducciones.

## Featured overlay (CSS)

El post destacado en homepage muestra la imagen con un overlay de texto:
- `.featured-body`: `position: absolute; bottom: 0` con gradiente `rgba(0,0,0,0.78)` -> transparente
- Todo texto forzado a blanco con `text-shadow` sutil
- Label badge: frosted glass (`rgba(255,255,255,0.18)` + `backdrop-filter: blur(4px)`)
- Meta y summary con blancos translucidos (0.8 y 0.88)
- Sin borde en `.featured` (limpio contra fondo)
- Responsive: a 640px se reduce padding y font sizes

## Config schema

### Campos principales
- `base_url`, `site_title`, `site_description`
- `theme`, `color_scheme` (auto|light|dark)
- `vault_path`
- `content_dir`, `public_dir`, `templates_dir`, `static_dir`, `themes_dir`, `plugins_dir`, `plugins_enabled`, `sass_dir`
- `content_layout`, `include_drafts`, `compile_sass`
- `tui_prefix`, `tui_prefix_ms`
- `serve_watch`, `serve_live_reload`, `serve_debounce_ms`
- `build_incremental`, `build_cache_dir`
- `clean_public`
- `doctor_profile`
- `summary_strategy` (auto|manual|ai)
- `ai` (provider, model, api_key, base_url, system_prompt, timeout, concurrency)
- `default_language` (default: "es")
- `logging` (level, format)
- `taxonomies` (name, paginate_by, paginate_path, feed, render)

### color_scheme
- `auto` (default): dark mode via `@media (prefers-color-scheme: dark)` CSS media query
- `light`: fuerza modo claro, dark mode nunca se aplica
- `dark`: fuerza modo oscuro via `html[data-color-scheme="dark"]`
- Valores invalidos producen error en `Load()`: `invalid color_scheme "X": must be auto, light, or dark`

Se expone como `data-color-scheme` en el atributo del `<html>` de todas las plantillas.

### summary_strategy
- `auto` (default): genera summary automaticamente si la page no tiene uno en frontmatter
- `manual`: solo usa summaries puestos a mano en frontmatter
- `ai`: usa provider AI (Kairos) si disponible, sino cae a `auto`
- Valores invalidos producen error en `Load()`: `invalid summary_strategy "X": must be auto, manual, or ai`

### ai (AIConfig)
- `provider` (default: "gemini"): LLM provider. Valores validos: gemini, anthropic, openai, qwen, ollama
- `model`: modelo LLM. Si vacio, usa el default del provider (gemini: "gemini-3-flash-preview", anthropic: "claude-haiku-4-20250514", openai: "gpt-5-mini", qwen: "qwen-turbo")
- `api_key`: API key. Si vacio, el provider usa su env var por defecto (GOOGLE_API_KEY, ANTHROPIC_API_KEY, OPENAI_API_KEY). Qwen requiere key explicita
- `base_url`: override del endpoint. Util para ollama ("http://localhost:11434") o proxies
- `system_prompt`: instruccion de sistema custom. Si vacio, usa prompt por defecto con idioma inyectado segun `default_language`. Default: "Summarize the following blog post in 1-2 short sentences (max 120 characters) in {Language} for use as a preview excerpt. Return only the summary text, no labels or prefixes."
- `timeout` (default: 30): timeout per-request en segundos. Valores <= 0 se normalizan a 30
- `concurrency` (default: 3): max goroutines paralelas para requests LLM. Valores <= 0 se normalizan a 3
- Provider invalido produce error: `invalid ai.provider "X": must be gemini, anthropic, openai, qwen, or ollama`

### default_language
- `es` (default): idioma por defecto para traducciones de plantillas
- Acepta cualquier codigo de idioma valido (en, es, fr, de, pt, it, ca, etc.)
- Se normaliza a minusculas en `Load()` y se valida como no vacio
- Se expone como `lang` en el contexto de todas las plantillas
- Controla que traducciones usa `trans()` y que localizacion de meses usa `date_format()`

## Frontmatter output (normalizado)

Campos escritos a `content/`:
- `title`, `date`, `slug`, `draft`
- `summary` (de summary/description/excerpt)
- `image` (ruta absoluta o URL externa)
- `featured` (bool)
- `menu` (bool, si `osg.menu: true`)
- `tags`, `area`, `type` (taxonomias)
- `template`, `lang`
- `obsidian` (source filename + frontmatter original)

## APIs / Interfaces
CLI:
- osg init
- osg tui (default)
- osg update-content
- osg build
- osg serve
- osg doctor
- osg new <title>
- osg theme init <name>
- osg plugin install/enable/disable/list/init
- osg version

Flags (core):
- --vault-path
- --include-drafts
- --dry-run
- --verbose
- --osg-content-dir
- --public-dir
- -c / --config

Flags (osg new):
- --tags (comma-separated list)
- --publish (default: false -> draft)
- --editor / --no-editor (negatable; auto-detect por defecto)
- --notes-dir (subdirectorio destino dentro del vault; override de new_notes_dir config)

Flags (osg build):
- --force-ai-summaries (regenera summaries AI ignorando cache, requiere confirmacion)
- --yes / -y (bypass confirmacion de --force-ai-summaries)

## Comando `osg new`

Crea una nueva nota Markdown en el vault de Obsidian con frontmatter pre-configurado para OSG.

### Uso

```bash
osg new "Mi Nuevo Post"                           # crea draft, abre editor
osg new "Mi Nuevo Post" --publish                 # crea publicado
osg new "Mi Nuevo Post" --tags filosofia,logica   # con tags
osg new "Mi Nuevo Post" --dry-run                 # solo muestra que haria
osg new "Mi Nuevo Post" --vault-path /otro/vault  # override de vault
osg new "Mi Nuevo Post" --no-editor               # no abrir editor
osg new "Mi Nuevo Post" --notes-dir 02_Notes      # crear en subcarpeta del vault
```

### TUI

```
/new Mi Nuevo Post
```

En TUI, el titulo se pasa como argumentos del slash command (todos los args se unen con espacio). No soporta --tags ni --publish (siempre crea draft).

### Frontmatter generado

El frontmatter incluye todos los campos reconocidos del bloque `osg`. Los campos activos se establecen con valores; el resto aparecen como comentarios YAML para que el usuario los descomente cuando los necesite.

```yaml
---
title: Mi Nuevo Post
created: 2025-02-15 10:30
tags:
  - filosofia
  - logica
osg:
  publish: draft
  # title: ""          # Override page title (highest precedence)
  # image: ""          # Featured/hero image path
  # featured: false    # Mark as featured post
  # path: ""           # Custom output path override
  # permalink: ""      # URL pattern ({date}, {year}, {month}, {day}, {slug}, {title})
  # menu: false        # Add to navigation menu
  # abstract: ""       # Summary/excerpt override
  # author: ""         # Author override
---
```

- `title`: titulo original del post
- `created`: fecha y hora de creacion en formato Obsidian (`YYYY-MM-DD HH:MM`)
- `tags`: opcional, lista de tags pasados con --tags
- `osg.publish`: `draft` por defecto, `true` si --publish
- `osg.title` a `osg.author`: comentados como placeholders, descomentables por el usuario

El frontmatter se genera manualmente (no via `yaml.Marshal`) para poder incluir comentarios YAML. `yamlScalar()` se encarga de entrecomillar valores que contengan caracteres especiales.

### Apertura de editor

Tras crear el fichero, `osg new` intenta abrir un editor automaticamente:

1. **Resolucion**: `resolveEditor()` devuelve `config.DefaultEditor` si esta configurado, o `$EDITOR` del entorno.
2. **Comportamiento por defecto (auto-detect)**: Sin flag explicito, `Editor=true` + `EditorAuto=true`. Si no hay editor resolvible, se omite silenciosamente (sin error).
3. **Flag explicito `--editor`**: Fuerza apertura; si no hay editor configurado, muestra warning (no-fatal).
4. **Flag `--no-editor`**: Omite la apertura siempre.
5. **Ejecucion**: `openEditor()` divide el comando en partes (soporta args como `"code --wait"`) y conecta stdin/stdout/stderr para uso interactivo.

Configuracion del editor:

```yaml
# En config.yaml:
default_editor: "code --wait"    # o "vim", "nvim", "nano", "subl"

# O via variable de entorno:
export EDITOR=vim
```

### Ruta del fichero

Por defecto, el fichero se crea en `{vault_path}/{Title}.md` (convencion Obsidian: ficheros planos con nombres legibles).

Si `new_notes_dir` esta configurado (o se pasa `--notes-dir`), la ruta es `{vault_path}/{new_notes_dir}/{Title}.md`. El directorio se crea automaticamente si no existe. Prioridad: `--notes-dir` CLI > `new_notes_dir` config > raiz del vault.

Si el fichero ya existe, el comando retorna error.

Configuracion:

```yaml
# En config.yaml:
new_notes_dir: "02_Notes"    # subcarpeta dentro del vault

# O override puntual:
osg new "Post" --notes-dir Drafts
```

### Implementacion

- `internal/app/new.go`: `RunNew()` con `NewPostOptions` (Title, Tags, Publish, Editor, EditorAuto, NotesDir)
- `internal/app/new.go`: `buildFrontmatter()` genera YAML manual con comentarios
- `internal/app/new.go`: `yamlScalar()` entrecomilla valores YAML con caracteres especiales
- `internal/app/new.go`: `resolveEditor()` resuelve editor (config > $EDITOR)
- `internal/app/new.go`: `openEditor()` ejecuta editor interactivo
- `cmd/osg/main.go`: `NewCmd` struct con Kong, `Editor *bool` negatable, `NotesDir` string, dispatch en switch
- TUI: `/new` registrado en `commands.go`, handler en `update.go`, closure en `app/tui.go`

## Decisions (trade-offs)
- YAML como formato de config: reduce dependencias y alinea con frontmatter.
- update-content como comando por defecto: evita confusiones y favorece CI simple.
- Theme por defecto: entrega salida usable sin obligar a templates custom.
- Plugins WASM: extensibilidad con ABI reducido y sin dependencias nativas.
- TUI: feedback inmediato sin dependencias externas.
- Build incremental: usa stamps (mtime/size) y requiere full rebuild si cambian templates/assets/plugins.
- Bloque `osg` en frontmatter: namespace propio para no interferir con campos de Obsidian/otros tools.
- Placeholders SVG deterministas: mismo titulo siempre genera la misma imagen, evita churn en builds.
- Imagenes con rutas absolutas: funcionan tanto en homepage como en pagina de articulo sin paths relativos rotos.
- Color scheme sin JavaScript: CSS media queries + atributo `data-color-scheme` para forzar tema.
- Paleta Nord: coherencia visual entre placeholders, theme CSS, dark mode y light mode.

## Risks / Non-goals
Riesgos:
- Vault grande: rendimiento y memoria en lectura masiva.
- Frontmatter inconsistente: fechas mixtas, campos faltantes.
- Colisiones de slug/fecha -> rutas duplicadas.
- Plugins defectuosos: deben fallar en modo warning.
- Eliminacion de contenido no limpia `public/` automaticamente (stale files).
- Imagenes del vault con el mismo basename: solo la primera encontrada se indexa.

No goals (por ahora):
- CMS completo o editor online.
- Live reload avanzado.
- Search index integrado.
- Toggle de dark mode con JavaScript.

Planificado (Phase 10):
- Optimizacion de imagenes: WebP, srcset, `<picture>` (Step 6).
- Global site feed RSS/Atom (Step 2).
- Doctor improvements con diagnosticos accionables (Step 3).

## Shortcodes

Los shortcodes se expanden ANTES de que Goldmark procese el Markdown.
Dos tipos: block (pares) e inline (auto-cerrados).

### Block shortcodes (pares)

```
{{< name [args] >}}contenido{{< /name >}}
```

| Shortcode | Descripcion | Argumentos |
|-----------|-------------|------------|
| `note` | Admonicion informativa (azul Nord #5e81ac) | Titulo opcional (bare) |
| `warning` | Admonicion de aviso (naranja Nord #d08770) | Titulo opcional (bare) |
| `tip` | Admonicion consejo (verde Nord #a3be8c) | Titulo opcional (bare) |
| `details` | Bloque colapsable `<details>` | Texto del summary (bare) |
| `figure` | Figura con imagen, caption, enlace | `src`, `caption`, `alt`, `class`, `width`, `link` |
| `tabs` | Contenedor de pestanas | Ninguno |
| `tab` | Pestana individual (dentro de `tabs`) | Titulo (bare) |

### Inline shortcodes (auto-cerrados)

```
{{< name args />}}
```

| Shortcode | Descripcion | Argumentos |
|-----------|-------------|------------|
| `youtube` | Embed 16:9 responsive (youtube-nocookie.com) | ID de video o URL completa |
| `twitter` | Embed tweet via oEmbed + widgets.js | URL del tweet (x.com se normaliza a twitter.com) |
| `codepen` | Embed CodePen iframe | URL, `height`, `theme`, `tab` |

### parseArgs

Soporta: `key="value"`, `key='value'`, `key=value`, argumentos posicionales bare.
El primer argumento posicional se asigna a `_pos`.

### Archivos

- Motor: `internal/markdown/shortcode.go`
- Tests: `internal/markdown/shortcode_test.go` (33 tests)
- CSS: `style.css` (secciones ADMONITIONS, DETAILS, FIGURE, EMBEDS, TABS)
- JS: `static/js/tabs.js` (zero-dependency, a11y, keyboard nav)

## Interacciones (vistas y likes)

Sistema de interacciones para articulos: contador de vistas (total + unicas) y votacion like/dislike.

### Arquitectura

```
Browser (interactions.js)
  |
  |  POST /api/v1/pageview  { path, fp }
  |  POST /api/v1/vote      { path, fp, vote }
  |
  v
API Server (internal/api/)
  |
  v
SQLite (modernc.org/sqlite, pure Go)
  page_views: total + dedup por fingerprint/dia
  page_votes: un voto por fingerprint/pagina
```

### Modos de ejecucion

1. **Standalone**: `osg api` levanta solo el API server (para produccion con proxy inverso)
2. **Embedded**: `osg serve --api` embebe la API en el mismo servidor de preview (para desarrollo)

### Configuracion

Seccion `interactions:` en `config.yaml`:

```yaml
interactions:
  enabled: true                    # Activa vistas y likes (default: false)
  api_url: ""                      # URL del API para el browser (vacio = same origin)
  listen: ":8090"                  # Direccion del servidor standalone (osg api)
  db_path: ".osg/interactions.db"  # Ruta del fichero SQLite
  cors_origins:                    # Origenes permitidos para CORS
    - "https://misite.com"
  view_dedup_hours: 24             # Ventana de dedup para vistas unicas (horas)
```

**Campos:**

| Campo | Default | Descripcion |
|-------|---------|-------------|
| `enabled` | `false` | Activa el sistema de interacciones |
| `api_url` | `""` | URL base de la API vista desde el browser. Vacio = mismo origen (util con `osg serve --api`) |
| `listen` | `":8090"` | Direccion para `osg api` standalone |
| `db_path` | `".osg/interactions.db"` | Ruta al fichero SQLite (se crea automaticamente) |
| `cors_origins` | `[]` | Lista de origenes CORS permitidos. Vacia = sin CORS headers |
| `view_dedup_hours` | `24` | Horas de ventana para dedup de vistas unicas por fingerprint/pagina |

### API Endpoints

#### POST /api/v1/pageview

Registra una vista de pagina. Siempre incrementa el total; la vista unica se dedup por fingerprint + dia.

```json
// Request
{ "path": "/2025/03/06/mi-post/", "fp": "a1b2c3d4..." }

// Response
{ "views": 42, "unique": 28, "likes": 5, "dislikes": 1, "user_vote": 0 }
```

#### POST /api/v1/vote

Registra un voto. `vote`: 1=like, -1=dislike, 0=retract (elimina voto previo).

```json
// Request (like)
{ "path": "/2025/03/06/mi-post/", "fp": "a1b2c3d4...", "vote": 1 }

// Response
{ "views": 42, "unique": 28, "likes": 6, "dislikes": 1, "user_vote": 1 }
```

#### GET /api/v1/health

```json
{ "status": "ok" }
```

### Fingerprinting client-side

El fingerprint se genera 100% en el browser, sin usar IP del servidor (usuarios detras de proxies comparten IP; IPs domesticas cambian).

**Algoritmo:**

1. UUID aleatorio generado y almacenado en `localStorage` (persistente por browser)
2. Caracteristicas del browser: User-Agent, screen resolution, devicePixelRatio, timezone, language, platform, hardwareConcurrency, colorDepth
3. Se concatena UUID + caracteristicas y se hashea con SHA-256 (via `crypto.subtle.digest`)

Resultado: hash de 64 chars hex, estable para el mismo browser/dispositivo.

**Privacidad:**
- No se recoge IP del servidor
- No se usan cookies
- Respeta `navigator.doNotTrack` (si activo, no registra nada)
- El fingerprint es un hash one-way, no reversible a datos personales

### Schema SQLite

```sql
-- Vistas: una fila por visit (total), dedup por unique index
CREATE TABLE page_views (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  page_path  TEXT NOT NULL,
  fingerprint TEXT NOT NULL,
  created_at DATETIME DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_page_views_dedup
  ON page_views(page_path, fingerprint, date(created_at));

-- Votos: un voto por fingerprint/pagina
CREATE TABLE page_votes (
  page_path   TEXT NOT NULL,
  fingerprint TEXT NOT NULL,
  vote        INTEGER NOT NULL CHECK(vote IN (-1, 1)),
  created_at  DATETIME DEFAULT (datetime('now')),
  updated_at  DATETIME DEFAULT (datetime('now')),
  PRIMARY KEY (page_path, fingerprint)
);
```

**Dedup de vistas:** `INSERT OR IGNORE` + unique index. Dentro del mismo dia natural (UTC), el mismo fingerprint en la misma pagina solo genera una fila (una vista unica). `COUNT(*)` = vistas por dia-visitor, `COUNT(DISTINCT fingerprint)` = visitantes unicos.

### UI en templates

El bloque `page-interactions` se inserta entre `page-taxonomies` y `page-nav` en `page.html`:

- **Icono de ojo** + contador de vistas
- **Boton like** (pulgar arriba) + contador
- **Boton dislike** (pulgar abajo) + contador
- Estado visual: botones con `aria-pressed="true"` y clase `.active` cuando el usuario ha votado

CSS: seccion INTERACTIONS en `style.css` (~80 lineas). Colores Nord, layout flex, responsive.

JS condicional: solo se carga `interactions.js` si `interactions.enabled=true` en config.

### Ejemplos de uso

#### Desarrollo local (todo embebido)

```yaml
# config.yaml
interactions:
  enabled: true
  # api_url vacio = mismo origen (osg serve --api)
```

```bash
osg serve --api
# Sirve el sitio en :1313 con API embebida
# Las interacciones funcionan inmediatamente
```

#### Produccion (API standalone)

```yaml
# config.yaml
interactions:
  enabled: true
  api_url: "https://api.misite.com"
  listen: ":8090"
  db_path: "/var/lib/osg/interactions.db"
  cors_origins:
    - "https://misite.com"
    - "https://www.misite.com"
```

```bash
# Servidor 1: sitio estatico servido por nginx/caddy
# Servidor 2 (o mismo servidor, otro puerto):
osg api
# -> interactions API listening addr=:8090 db=/var/lib/osg/interactions.db
```

#### Detras de proxy inverso (nginx)

```nginx
server {
    listen 443 ssl;
    server_name api.misite.com;

    location /api/ {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Despliegue en servidor propio

Guia paso a paso para ejecutar la API de interacciones en un servidor dedicado
(VPS, maquina propia, etc.) con el sitio estatico servido desde Cloudflare u
otro CDN.

#### 1. Compilar el binario

```bash
# En la maquina de desarrollo:
GOOS=linux GOARCH=amd64 go build -o osg ./cmd/osg

# Copiar al servidor:
scp osg user@miservidor:/usr/local/bin/osg
```

#### 2. Configurar en el servidor

Crear directorio de datos y config minimo:

```bash
sudo mkdir -p /var/lib/osg
sudo chown osg:osg /var/lib/osg

cat > /etc/osg/config.yaml << 'EOF'
interactions:
  enabled: true
  listen: ":8090"
  db_path: "/var/lib/osg/interactions.db"
  cors_origins:
    - "https://misite.com"
    - "https://www.misite.com"
EOF
```

La base de datos SQLite se crea automaticamente en el primer arranque.

#### 3. Servicio systemd

```ini
# /etc/systemd/system/osg-api.service
[Unit]
Description=OSG Interactions API
After=network.target

[Service]
Type=simple
User=osg
Group=osg
WorkingDirectory=/var/lib/osg
ExecStart=/usr/local/bin/osg api --config /etc/osg/config.yaml
Restart=on-failure
RestartSec=5

# Seguridad
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/osg

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now osg-api
sudo systemctl status osg-api
# -> Active: active (running)
# -> interactions API listening addr=:8090 db=/var/lib/osg/interactions.db
```

#### 4. Proxy inverso (Caddy o nginx)

**Caddy** (opcion simple, HTTPS automatico):

```
api.misite.com {
    reverse_proxy localhost:8090
}
```

**nginx** (con certificado propio o Let's Encrypt):

```nginx
server {
    listen 443 ssl;
    server_name api.misite.com;
    ssl_certificate     /etc/letsencrypt/live/api.misite.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.misite.com/privkey.pem;

    location /api/ {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### 5. Configurar el sitio estatico

En el `config.yaml` del sitio (el que se usa para `osg build`):

```yaml
interactions:
  enabled: true
  api_url: "https://api.misite.com"
```

`api_url` es la URL publica que el navegador del visitante usara para
llamar a la API. Debe coincidir con el dominio del proxy inverso.

Tras `osg build`, el HTML generado incluira el JS de interacciones que
apunta a esa URL. No requiere rebuild para cambiar el servidor API (el
JS lee `api_url` del atributo `data-api-url` en el HTML).

#### 6. Verificar

```bash
# Desde cualquier maquina:
curl https://api.misite.com/api/v1/health
# -> {"status":"ok"}

# Registrar una vista de prueba:
curl -X POST https://api.misite.com/api/v1/pageview \
  -H "Content-Type: application/json" \
  -d '{"path":"/test/","fp":"abc123"}'
# -> {"views":1,"unique":1,"likes":0,"dislikes":0,"user_vote":0}
```

#### Backup de la base de datos

La BD es un unico fichero SQLite. Backup simple con cron:

```bash
# /etc/cron.daily/osg-backup
#!/bin/sh
sqlite3 /var/lib/osg/interactions.db ".backup /var/backups/osg/interactions-$(date +%Y%m%d).db"
find /var/backups/osg -name "interactions-*.db" -mtime +30 -delete
```

### Despliegue con container OCI (Podman / Docker)

Alternativa al despliegue con systemd: ejecutar `osg api` como container.
Los comandos usan `podman`, pero son intercambiables con `docker`.

#### Containerfile

```dockerfile
# -- Build stage --
FROM docker.io/library/golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /osg ./cmd/osg

# -- Runtime stage --
FROM docker.io/library/alpine:3.21
RUN apk add --no-cache sqlite ca-certificates \
    && addgroup -S osg && adduser -S osg -G osg
COPY --from=build /osg /usr/local/bin/osg
USER osg
EXPOSE 8090
VOLUME ["/data"]
ENTRYPOINT ["osg", "api"]
CMD ["--config", "/etc/osg/config.yaml"]
```

`sqlite` se instala para disponer del CLI de backup (`.backup`). El
binario Go no lo necesita (usa `modernc.org/sqlite`, pure Go).

#### Construir la imagen

```bash
podman build -t osg-api:latest .
```

#### Ejecutar

```bash
# Crear volumen para persistir la BD
podman volume create osg-data

# Ejecutar con config minima via variables de entorno
podman run -d \
  --name osg-api \
  --restart always \
  -p 8090:8090 \
  -v osg-data:/data \
  -e OSG_INTERACTIONS__ENABLED=true \
  -e OSG_INTERACTIONS__LISTEN=":8090" \
  -e OSG_INTERACTIONS__DB_PATH="/data/interactions.db" \
  -e 'OSG_INTERACTIONS__CORS_ORIGINS=https://misite.com,https://www.misite.com' \
  osg-api:latest

# Verificar
podman logs osg-api
# -> interactions API listening addr=:8090 db=/data/interactions.db

curl http://localhost:8090/api/v1/health
# -> {"status":"ok"}
```

Alternativa con fichero de config montado:

```bash
podman run -d \
  --name osg-api \
  --restart always \
  -p 8090:8090 \
  -v osg-data:/data \
  -v ./config.yaml:/etc/osg/config.yaml:ro \
  osg-api:latest
```

#### Compose (podman-compose / docker compose)

```yaml
# compose.yaml
services:
  osg-api:
    build: .
    container_name: osg-api
    restart: always
    ports:
      - "8090:8090"
    volumes:
      - osg-data:/data
      - ./config.yaml:/etc/osg/config.yaml:ro
    environment:
      - OSG_INTERACTIONS__ENABLED=true
      - OSG_INTERACTIONS__DB_PATH=/data/interactions.db

volumes:
  osg-data:
```

```bash
podman compose up -d
```

#### Backup de la BD en container

```bash
# Backup manual (copia el fichero SQLite del volumen)
podman exec osg-api sqlite3 /data/interactions.db ".backup /data/backup.db"
podman cp osg-api:/data/backup.db ./interactions-backup.db

# Backup automatico con cron en el host
# /etc/cron.daily/osg-container-backup
#!/bin/sh
podman exec osg-api sqlite3 /data/interactions.db \
  ".backup /data/interactions-$(date +%Y%m%d).db"
podman exec osg-api find /data -name "interactions-2*.db" -mtime +30 -delete
```

#### Actualizacion

```bash
# Reconstruir imagen con codigo nuevo
podman build -t osg-api:latest .

# Reemplazar container (la BD persiste en el volumen)
podman stop osg-api && podman rm osg-api
podman run -d \
  --name osg-api \
  --restart always \
  -p 8090:8090 \
  -v osg-data:/data \
  -v ./config.yaml:/etc/osg/config.yaml:ro \
  osg-api:latest
```

### Archivos

- Store: `internal/api/store.go` (SQLite, schema, RecordView, Vote, GetStats)
- Server: `internal/api/server.go` (HTTP handlers, routes)
- Middleware: `internal/api/middleware.go` (CORS)
- Validation: `internal/api/validation.go` (request types, validation)
- CLI: `internal/app/api.go` (RunAPI, StartAPIHandler)
- JS: `internal/theme/default/static/js/interactions.js` (fingerprinting, API calls, UI)
- Template: `internal/theme/default/templates/page.html` (bloque page-interactions)
- CSS: `internal/theme/default/static/style.css` (seccion INTERACTIONS)
- Tests: `internal/api/store_test.go` (14), `server_test.go` (13), `validation_test.go` (12)

## Sharing (boton compartir con popover)

Boton compacto de compartir con popover desplegable (patron Medium). Al hacer clic en el boton "Compartir" aparece un dropdown con opciones de redes sociales y copiar enlace.

### Configuracion

```yaml
# config.yaml
sharing: true   # default: true
```

Campo `Sharing bool` en `Config` struct. Default `true` en `Default()`. Expuesto como `sharing` en `configView()` para templates.

### Funcionalidad

Boton unico `.share-toggle` con icono de red (circulos conectados) y texto "Compartir"/"Share". Al hacer clic, abre un popover (`.share-popover`) con las opciones:

   - **X** (Twitter): `https://x.com/intent/tweet?url=<url>&text=<title>`
   - **LinkedIn**: `https://www.linkedin.com/sharing/share-offsite/?url=<url>`
   - **Bluesky**: `https://bsky.app/intent/compose?text=<title> <url>`
   - **Email**: `mailto:?subject=<title>&body=<url>`
   - *(separador)*
   - **Copy link**: copia la URL al clipboard con feedback visual (check icon + texto "Enlace copiado!")

El popover se cierra al hacer clic fuera o al pulsar Escape. El boton usa `aria-expanded` y `aria-haspopup` para accesibilidad.

### Layout: barra de acciones unificada

Interactions y sharing se fusionan en una sola barra `.article-actions`:

```
[eye + views] [like] [dislike]          [Compartir ▸]
```

- Izquierda: `.interactions-group` (vistas + votos juntos, flex row)
- Derecha: `.share-wrap` (margin-left auto)
- Si solo hay interactions (sharing disabled): la barra muestra solo vistas + votos
- Si solo hay sharing (interactions disabled): la barra muestra solo el boton compartir
- El bloque unico `page-actions` reemplaza los antiguos `page-interactions` + `page-share`

### Resolucion de URLs

Las URLs de sharing necesitan ser absolutas. `share.js` usa `resolveURL()` que crea un elemento `<a>` temporal para que el browser resuelva la URL relativa a absoluta (funciona incluso sin `base_url` en config).

### Template gating

El bloque share esta condicionado a `{{ if and .config.sharing .page.permalink }}`. El script `share.js` solo se carga con `{{ if .config.sharing }}`.

### i18n

| Clave | en | es |
|-------|----|----|
| `share` | Share | Compartir |
| `share_on` | Share on | Compartir en |
| `share_via_email` | Share via email | Compartir por email |
| `copy_link` | Copy link | Copiar enlace |
| `link_copied` | Link copied! | Enlace copiado! |

### CSS

- **ARTICLE ACTIONS BAR**: `.article-actions` (flex, space-between, borde, background surface). `.interactions-group` (flex row, vistas + votos). `.share-wrap` (margin-left auto, position relative).
- **SHARE POPOVER**: `.share-toggle` (pill button, border-radius-full), `.share-popover` (absolute bottom-right, shadow-md, z-index 100), `.share-option` (fila con icono + texto, colores de marca en hover), `.share-divider` (hr entre opciones sociales y copy-link).

### Archivos

- JS: `internal/theme/default/static/js/share.js` (popover toggle, clipboard, resolveURL)
- Template: `internal/theme/default/templates/page.html` (bloque page-actions)
- CSS: `internal/theme/default/static/style.css` (secciones ARTICLE ACTIONS BAR y SHARE POPOVER)
- i18n: `internal/theme/default/i18n/en.yaml`, `es.yaml` (5 claves)
- Config: `internal/config/config.go` (Sharing bool, default true)
- Tests: `internal/config/config_test.go` (TestDefault_SharingEnabled, TestLoad_SharingDisabled)
- Build: `internal/build/build.go` (sharing en configView), `build_test.go` (sharing en TestConfigView)

## Comentarios (sistema de comentarios con OAuth2)

Sistema de comentarios con autenticacion OAuth2 (GitHub, Google), hilos anidados con profundidad ilimitada y base de datos SQLite separada.

### Arquitectura

```
Browser (comments.js)
  |
  |  GET  /api/v1/comments?page=/path          (publico)
  |  POST /api/v1/comments                     (autenticado)
  |  DELETE /api/v1/comments/{id}              (autenticado, solo propio)
  |  GET  /api/v1/auth/me                      (cookie)
  |  POST /api/v1/auth/logout                  (cookie)
  |
  v
API Server (internal/api/)
  |
  v
SQLite separada (comments.db)
  users: id, provider, provider_id, name, avatar_url, timestamps
  sessions: token, user_id, expires_at
  comments: id, page_path, user_id, parent_id, body, deleted, timestamps
```

### Configuracion

Seccion anidada bajo `interactions:` en `config.yaml`:

```yaml
interactions:
  comments:
    enabled: true                    # Activa comentarios (default: false)
    db_path: ".osg/comments.db"      # BD SQLite separada de interactions.db
    auth_session_days: 30            # Duracion de sesion (dias)
    auth_callback_url: "https://misite.com"  # URL base para callbacks OAuth2
    providers:
      - provider: github
        client_id: "..."
        client_secret: "..."
      - provider: google
        client_id: "..."
        client_secret: "..."
```

**Campos:**

| Campo | Default | Descripcion |
|-------|---------|-------------|
| `enabled` | `false` | Activa el sistema de comentarios |
| `db_path` | `".osg/comments.db"` | Ruta al fichero SQLite de comentarios (separada de interactions.db) |
| `auth_session_days` | `30` | Dias de duracion de la sesion de autenticacion |
| `auth_callback_url` | `""` | URL base para construir callbacks OAuth2 (https -> cookies seguras) |
| `providers` | `[]` | Lista de proveedores OAuth2 (github, google) |

Cada provider requiere `client_id` y `client_secret`. Providers invalidos (ni github ni google) producen error en `Load()`.

### OAuth2 flow

1. Usuario hace clic en "Login with GitHub/Google" → JS navega a `/api/v1/auth/{provider}?return_to=/current/page/`
2. Server genera `state` aleatorio (crypto/rand), almacena `state|return_to` en cookie `osg_auth_state` (10 min, httpOnly, SameSite=Lax)
3. Server redirige a la URL de autorizacion del provider (GitHub: `github.com/login/oauth/authorize`, Google: `accounts.google.com/o/oauth2/v2/auth`)
4. Provider redirige a `/api/v1/auth/{provider}/callback?code=...&state=...`
5. Server verifica state contra cookie, intercambia code por token, obtiene info de usuario
6. Upsert user en BD (actualiza nombre/avatar si cambio), crea sesion (token aleatorio 32 bytes hex)
7. Setea cookie `osg_session` (httpOnly, SameSite=Lax, 30 dias, Secure si `auth_callback_url` es https)
8. Redirige al `return_to` original

**Scopes:**
- GitHub: `read:user` (nombre, avatar, login)
- Google: `openid profile email` (nombre, avatar)

### API Endpoints

#### GET /api/v1/auth/{provider}

Inicia flujo OAuth2. `provider` puede ser `github` o `google`. Query param `return_to` indica a donde redirigir tras login.

#### GET /api/v1/auth/{provider}/callback

Callback OAuth2. Verifica state, intercambia code, upsert user, crea sesion, redirige.

#### GET /api/v1/auth/me

Retorna el usuario autenticado (de la cookie `osg_session`). 401 si no hay sesion valida.

```json
// Response
{ "id": 1, "name": "Joan", "avatar_url": "https://avatars...", "provider": "github" }
```

#### POST /api/v1/auth/logout

Elimina sesion de la BD y borra cookie `osg_session`.

#### GET /api/v1/comments?page=/path

Lista comentarios de una pagina como arbol anidado. Publico (sin auth).

```json
// Response
{
  "comments": [
    {
      "id": 1,
      "body": "Gran articulo!",
      "author": { "name": "Joan", "avatar_url": "..." },
      "created_at": "2025-03-06T10:00:00Z",
      "deleted": false,
      "replies": [
        {
          "id": 2,
          "body": "Gracias!",
          "parent_id": 1,
          "author": { "name": "Ana", "avatar_url": "..." },
          "created_at": "2025-03-06T11:00:00Z",
          "deleted": false,
          "replies": []
        }
      ]
    }
  ]
}
```

Los comentarios eliminados (soft-delete) se preservan en el arbol si tienen respuestas no eliminadas. Su body se vacia y el autor se anonimiza como "[deleted]". Los comentarios eliminados sin respuestas se podan del arbol.

#### POST /api/v1/comments

Crea un comentario. Requiere sesion valida.

```json
// Request
{ "page_path": "/2025/03/06/mi-post/", "body": "Gran articulo!", "parent_id": null }

// Response (comment object)
{ "id": 3, "body": "Gran articulo!", "author": { ... }, "created_at": "...", "deleted": false }
```

Validacion: `page_path` obligatorio, `body` obligatorio y max 10000 caracteres, `parent_id` si presente debe existir y pertenecer a la misma pagina.

#### DELETE /api/v1/comments/{id}

Soft-delete de un comentario. Solo el autor puede eliminar sus propios comentarios. 403 si el usuario no es el autor.

### CommentStore (BD separada)

`internal/api/comment_store.go` — SQLite separada de interactions.db para facilitar migracion futura a PostgreSQL u otro RDBMS.

**Schema:**

```sql
CREATE TABLE users (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  provider    TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  name        TEXT NOT NULL,
  avatar_url  TEXT DEFAULT '',
  created_at  DATETIME DEFAULT (datetime('now')),
  updated_at  DATETIME DEFAULT (datetime('now')),
  UNIQUE(provider, provider_id)
);

CREATE TABLE sessions (
  token      TEXT PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at DATETIME NOT NULL,
  created_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE comments (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  page_path  TEXT NOT NULL,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  parent_id  INTEGER REFERENCES comments(id) ON DELETE CASCADE,
  body       TEXT NOT NULL,
  deleted    BOOLEAN DEFAULT FALSE,
  created_at DATETIME DEFAULT (datetime('now')),
  updated_at DATETIME DEFAULT (datetime('now'))
);
CREATE INDEX idx_comments_page ON comments(page_path, created_at);
```

**Foreign keys**: habilitadas via `PRAGMA foreign_keys = ON`. WAL mode + busy timeout para concurrencia.

**Tree building**: `buildCommentTree()` usa algoritmo de dos pasos: (1) crea mapa id→comment, (2) enlaza hijos. `pruneDeletedLeaves()` elimina recursivamente comentarios borrados sin respuestas.

### Frontend (comments.js)

IIFE con `"use strict"` y declaraciones `var` (patron de compatibilidad, igual que `interactions.js` y `share.js`).

**Funcionalidad:**
- Deteccion de autenticacion via `GET /api/v1/auth/me` al cargar
- Si no autenticado: muestra botones de login (GitHub/Google segun providers configurados)
- Si autenticado: muestra formulario de comentario + avatar + nombre + boton logout
- Listado recursivo de comentarios con profundidad visual (clases CSS `comment-depth-N`)
- Respuesta inline: formulario de respuesta debajo de cada comentario
- Eliminacion de propios comentarios
- `timeAgo()` para timestamps relativos ("hace 5 minutos", "hace 2 dias")
- HTML escaping para prevenir XSS
- `credentials: "include"` en todos los fetch para enviar cookies

### CSS

Seccion COMMENTS en `style.css` (~200 lineas):

- `.comments-section`: borde superior, padding, margin-top
- `.comments-login`: botones de login con iconos SVG de GitHub/Google
- `.comment-form`: textarea + boton submit con estilo Nord
- `.comment-list`: lista de comentarios
- `.comment`: flex row con avatar + contenido, borde izquierdo sutil
- `.comment-depth-1` a `.comment-depth-5`: margin-left incremental (0, 1.5rem, 3rem, 4.5rem, 6rem). Profundidad >5 se aplana visualmente
- `.comment-author`: nombre bold + timestamp
- `.comment-body`: texto del comentario
- `.comment-actions`: responder + eliminar
- `.comment-deleted`: texto en italic gris para comentarios eliminados
- Responsive: en movil se reduce el margin-left de nesting

### i18n

| Clave | en | es |
|-------|----|----|
| `comments` | Comments | Comentarios |
| `comments_login` | Log in to comment | Inicia sesion para comentar |
| `comments_login_github` | Log in with GitHub | Iniciar sesion con GitHub |
| `comments_login_google` | Log in with Google | Iniciar sesion con Google |
| `comments_placeholder` | Write a comment... | Escribe un comentario... |
| `comments_submit` | Post comment | Publicar comentario |
| `comments_reply` | Reply | Responder |
| `comments_delete` | Delete | Eliminar |
| `comments_deleted` | This comment has been deleted. | Este comentario ha sido eliminado. |
| `comments_logout` | Logout | Cerrar sesion |
| `comments_none` | No comments yet. Be the first! | Aun no hay comentarios. Se el primero! |
| `comments_reply_to` | Reply to | Responder a |
| `comments_cancel` | Cancel | Cancelar |

### Template gating

El bloque `page-comments` esta condicionado a `{{ if .config.comments_enabled }}`. El script `comments.js` solo se carga con la misma condicion. Los providers disponibles se pasan como `comments_providers` en `configView()` (lista de mapas con `name` y `label`).

### Despliegue

#### Dockerfile

Multi-stage build: `golang:1.25-alpine` (builder) → `alpine:3.21` (runtime). Non-root user `osg`, volumenes para `/data` y `/site`. Entrypoint: `osg api --config /etc/osg/config.yaml`.

#### docker-compose.yml

Servicio `osg-api` con volumen `osg-data` para persistir las BDs. Soporta override de config via variables de entorno o fichero montado.

#### Kubernetes

- `deploy/k8s/configmap.yaml`: config.yaml como ConfigMap
- `deploy/k8s/pvc.yaml`: 1Gi PVC para datos SQLite
- `deploy/k8s/deployment.yaml`: single replica, probes en `/api/v1/health`, resource limits, non-root securityContext
- `deploy/k8s/service.yaml`: ClusterIP en port 8090

### Archivos

- CommentStore: `internal/api/comment_store.go` (SQLite, schema, users, sessions, comments, tree building)
- Auth handlers: `internal/api/auth.go` (OAuth2 flow, BuildAuthProviders, HandleLogin/Callback/Me/Logout)
- Comment handlers: `internal/api/comments.go` (HandleList/Create/Delete, validation)
- Server: `internal/api/server.go` (5-arg NewServer, rutas condicionales)
- Middleware: `internal/api/middleware.go` (CORS con withCredentials)
- CLI: `internal/app/api.go` (RunAPI con CommentStore), `internal/app/serve.go` (StartAPIHandler 4 retornos)
- Config: `internal/config/config.go` (CommentsConfig, AuthProviderConfig, normalizacion)
- Build: `internal/build/build.go` (comments_enabled, comments_providers en configView)
- JS: `internal/theme/default/static/js/comments.js` (IIFE, auth, CRUD, tree rendering)
- Template: `internal/theme/default/templates/page.html` (bloque page-comments)
- CSS: `internal/theme/default/static/style.css` (seccion COMMENTS)
- i18n: `internal/theme/default/i18n/en.yaml`, `es.yaml` (13 claves)
- Deployment: `Dockerfile`, `docker-compose.yml`, `deploy/k8s/*.yaml`
- Tests: `comment_store_test.go` (25), `auth_test.go` (21), `comments_test.go` (19), `config_test.go` (6 nuevos)

## Open questions
- (ninguna pendiente)

## TUI avanzado (Fase 17)

El TUI de Bubble Tea se ha ampliado con gestion de servicios, panel de logs,
y editor de configuracion modal. Las funcionalidades se organizan en 6 fases
(A-F) documentadas en `docs/TUI-ENHANCEMENTS.md`.

### Arquitectura

```
+------------------------------------------------------------------+
|  Header: title  badges (serve/api running, mode, addresses)      |
+---------------+--------------------------------------------------+
|               |                                                  |
|   Sidebar     |   Viewport (output log + history)                |
|   - Project   |                                                  |
|   - Workflow   |                                                  |
|   - Services  |                                                  |
|   - Plugins   |                                                  |
|   - Build     |                                                  |
|               +--------------------------------------------------+
|               |   Log Panel (togglable, F7)                      |
|               |   Tabs: Serve | API | All                        |
+---------------+--------------------------------------------------+
|  Input: slash commands                                           |
+------------------------------------------------------------------+
|  Hint Bar: contextual hints segun modo activo                    |
+------------------------------------------------------------------+
```

El Config Editor es un modal full-screen separado (F8):

```
+------------------------------------------------------------------+
|  Config Editor (modified*)            Ctrl+S save  Esc back      |
+-------------------+----------------------------------------------+
|  Sections         |  Fields                                      |
|  > General      |  site_title: [My Site]                       |
|    Paths          |  base_url: [https://...]                     |
|    Content        |  description: [...]                          |
|    Logging        |  include_drafts: false  (Space toggle)       |
|    ...            |  ...                                         |
+-------------------+----------------------------------------------+
|  Ctrl+S save  Tab panel  Enter edit  ^/v nav  Space toggle       |
+------------------------------------------------------------------+
```

### Modulos

Fase A - **Sistema de logs multi-canal** (`logsink.go`):
- `LogSink` con `source` tag (general/serve/api)
- `TaggedLine` struct: `Source` + `Line`
- `MergeChannels()`: fan-in de multiples sinks en un canal unico
- Buffers por source en Model: `serveMessages`, `apiMessages`

Fase B - **Gestion de procesos serve/api** (`update.go`, `commands.go`):
- `Actions.ServeWithAPI`: lanza dev server con API embebida
- `Actions.RunAPI`: lanza API server standalone
- 3 modos: `/serve` (static), `/serve --api` (embebido), `/api` (standalone)
- `/stop serve|api` para detener servicios
- F5 (serve toggle), F6 (api toggle), badges en header y sidebar

Fase C - **Panel de logs** (`logpanel.go`):
- Panel inferior togglable (F7 o `/logs`)
- Viewport propio con scroll independiente (Mod+Up/Down, Mod = configurable)
- Tabs: Serve, API, All (Mod+Left/Right)
- Tecla modificadora configurable via `tui_log_modifier`: "shift" (defecto) o "alt"
- Altura: ~1/3 terminal, clamped 4-20 lineas
- Focus mode: cuando el log panel tiene foco, hint bar cambia

Fase D - **Infraestructura de config** (`config/schema.go`, `config/yamlnode.go`):
- `ConfigSchema()`: 24 secciones con todos los campos tipados
- `FieldType` enum: String, Bool, Int, StringList, IntList, StringMap, Struct, StructList
- CRUD via `yaml.Node`: `LoadNode`, `SaveNode`, `GetNodeValue`, `SetNodeValue`,
  `SetNodeSequence`, `GetNodeSequence`, `DeleteNodeKey`
- Preservacion de comentarios YAML via round-trip con `gopkg.in/yaml.v3`

Fase E - **Editor de config modal** (`configscreen.go`, `configfields.go`):
- `ConfigScreen`: full-screen modal con panel izquierdo (secciones) + derecho (campos)
- Navegacion: Up/Down por secciones/campos, Tab entre paneles
- Edicion inline: Enter para editar, Esc para cancelar, Enter para confirmar
- Bool toggle con Space, listas con add (a) / delete (d)
- `FieldEditor`: text input con validacion por tipo (int, bool, options)
- Dirty tracking por campo, indicador visual `(modified*)`
- Save explicito con Ctrl+S; Esc con cambios sin guardar muestra confirmacion (y/n/Esc)
- Campos sensibles (passwords, secrets) con masking

Fase F - **Integracion y pulido** (`statusbar.go`, update.go):
- Hint bar contextual: cambia segun modo normal/log-focus/config
- Config reload: al guardar, `reloadOptionsFromConfig()` actualiza sidebar
- Status bar muestra estado de servicios running con direcciones

### Keybindings

| Tecla | Accion |
|-------|--------|
| F5 | Toggle serve (static) |
| F6 | Toggle API (standalone) |
| F7 | Toggle log panel |
| F8 | Abrir/cerrar config editor |
| Mod+Up/Down | Scroll log panel (Mod = Shift por defecto) |
| Mod+Left/Right | Cambiar tab log panel |
| Ctrl+S | Guardar config (en editor) |
| Tab | Cambiar panel (sidebar/config sections/fields) |
| Esc | Cerrar modal / volver |

### Configuracion TUI

Tres opciones en `config.yaml`:

```yaml
tui_prefix: space       # Tecla prefijo ("space" o "ctrl")
tui_prefix_ms: 600      # Timeout en ms para la segunda tecla tras el prefijo
tui_log_modifier: shift  # Tecla modificadora para el log panel ("shift" o "alt")
```

#### tui_log_modifier

Controla que tecla modificadora se usa para navegar el log panel (scroll y
cambio de tabs). Valores posibles: `"shift"` (por defecto) o `"alt"`.

**`shift` (recomendado)** — Funciona en todos los terminales y sistemas
operativos sin configuracion adicional. Las combinaciones son Shift+Up/Down
para scroll y Shift+Left/Right para cambiar de tab.

**`alt`** — Usa Alt+flechas en lugar de Shift+flechas. Puede ser preferible
si Shift+flechas interfiere con otros atajos (tmux, screen).

> **Nota para macOS**: La tecla Option (⌥) de macOS **NO** funciona como
> Alt/Meta por defecto en la mayoria de terminales. Option+flechas produce
> caracteres especiales Unicode en lugar de enviar secuencias `alt+up`, etc.
>
> Para que `tui_log_modifier: alt` funcione en macOS hay que configurar el
> terminal manualmente:
>
> - **iTerm2**: Preferences → Profiles → Keys → Left/Right Option key → **Esc+**
> - **Terminal.app**: Preferences → Profiles → Keyboard → activar **"Use Option as Meta key"**
> - **Ghostty**: `macos-option-as-alt = true` en config
> - **Kitty**: `macos_option_as_alt yes` en `kitty.conf`
> - **Alacritty**: no tiene equivalente directo; Option no envia Meta
>
> Si no se configura el terminal, las combinaciones Alt+flecha simplemente no
> tendran efecto. No hay error: el TUI funcionara normalmente pero las teclas
> de navegacion del log panel no responderan. En ese caso usar `shift`.

### Slash commands

| Comando | Descripcion |
|---------|-------------|
| `/serve [--api]` | Lanza dev server (opcionalmente con API embebida) |
| `/api` | Lanza API server standalone |
| `/stop [serve\|api]` | Detiene servicios |
| `/logs` | Toggle panel de logs |
| `/config` | Abre editor de configuracion |

### Archivos

- LogSink: `internal/tui/logsink.go` (TaggedLine, MergeChannels)
- Log panel: `internal/tui/logpanel.go` (LogPanel, LogTab)
- Config screen: `internal/tui/configscreen.go` (ConfigScreen)
- Field editor: `internal/tui/configfields.go` (FieldEditor)
- Schema: `internal/config/schema.go` (ConfigSchema, 24 secciones)
- YAML node CRUD: `internal/config/yamlnode.go` (LoadNode, SaveNode, etc.)
- Commands: `internal/tui/commands.go` (/serve, /api, /stop, /logs, /config)
- Keys: `internal/tui/keys.go` (F5-F8)
- Styles: `internal/tui/styles.go` (14+ config styles, log panel styles)
- Status bar: `internal/tui/statusbar.go` (hint bar contextual)
- Tests: `logsink_test.go`, `logpanel_test.go`, `configscreen_test.go`, `schema_test.go`, `yamlnode_test.go`
