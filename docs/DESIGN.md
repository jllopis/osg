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
---
```

### Prioridades de resolucion
- **publish**: `osg.publish` > `publish` top-level
- **image**: `osg.image` > `image` > `cover` > `banner` top-level
- **featured**: `osg.featured` > `featured` top-level
- **path**: solo `osg.path` (sin fallback top-level)
- **menu**: solo `osg.menu` (sin fallback top-level)

### Paginas standalone (osg.path + osg.menu)

Cuando una nota tiene `osg.path`, el output se escribe en `content/{path}/index.md` en lugar de seguir el `content_layout` por defecto. Esto permite URLs limpias como `/about/` o `/contacto/`.

Si ademas tiene `osg.menu: true`, la pagina:
- Se incluye en la variable de template `menu_pages` (disponible en todas las plantillas)
- Se **excluye** del listado de posts en homepage/secciones (no aparece en `section.pages`)
- Aparece como enlace en el `<nav>` del header junto a las taxonomias

### Valores de publish
- `true` (bool) o `"true"` (string): publicar normalmente
- `"draft"` (string, case-insensitive): publicar como borrador
- Cualquier otro valor: no publicar

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

`internal/i18n/` provee traduccion de cadenas UI en las plantillas del tema. No es multi-language content routing — solo traduce strings de interfaz.

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

Flags (osg build):
- --force-ai-summaries (regenera summaries AI ignorando cache, requiere confirmacion)
- --yes / -y (bypass confirmacion de --force-ai-summaries)

## Comando `osg new`

Crea una nueva nota Markdown en el vault de Obsidian con frontmatter pre-configurado para OSG.

### Uso

```bash
osg new "Mi Nuevo Post"                           # crea draft
osg new "Mi Nuevo Post" --publish                 # crea publicado
osg new "Mi Nuevo Post" --tags filosofia,logica   # con tags
osg new "Mi Nuevo Post" --dry-run                 # solo muestra que haria
osg new "Mi Nuevo Post" --vault-path /otro/vault  # override de vault
```

### TUI

```
/new Mi Nuevo Post
```

En TUI, el titulo se pasa como argumentos del slash command (todos los args se unen con espacio). No soporta --tags ni --publish (siempre crea draft).

### Frontmatter generado

```yaml
---
title: Mi Nuevo Post
created: "2025-02-15 10:30"
tags:
  - filosofia
  - logica
osg:
  publish: "draft"
---
```

- `title`: titulo original del post
- `created`: fecha y hora de creacion en formato Obsidian (`YYYY-MM-DD HH:MM`)
- `tags`: opcional, lista de tags pasados con --tags
- `osg.publish`: `"draft"` por defecto, `true` si --publish

### Ruta del fichero

El fichero se crea en `{vault_path}/{Title}.md` (convencion Obsidian: ficheros planos con nombres legibles). Si el fichero ya existe, el comando retorna error.

### Implementacion

- `internal/app/new.go`: `RunNew()` con `NewPostOptions` (Title, Tags, Publish, Editor)
- `cmd/osg/main.go`: `NewCmd` struct con Kong, dispatch en switch
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

## Open questions
- (ninguna pendiente)
