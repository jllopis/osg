# Templates - Especificacion tecnica

## Objetivo
Definir el sistema de plantillas de OSG para renderizar paginas HTML desde contenido Markdown.

## Estructura de directorios
- `templates/`: plantillas de usuario
- `themes/<theme>/templates/`: plantillas del tema (incluye `partials/`)
- `internal/render/builtins/`: plantillas incorporadas (embebidas)

## Resolucion de plantillas (prioridad)
1) Usuario: `templates/` (raiz y subdirectorios)
2) Tema: `themes/<theme>/templates/`
3) Built-in: `internal/templates/builtins/`

Si existe un archivo con el mismo nombre en un nivel superior, se usa ese.

## Plantillas estandar
- `index.html` -> homepage (hero featured + lista de posts con thumbnails)
- `section.html` -> secciones
- `page.html` -> paginas (article hero image + contenido)

## Plantillas especiales
- `atom.xml`, `rss.xml`
- `sitemap.xml`, `split_sitemap_index.xml`
- `robots.txt`
- `404.html`
- `taxonomy_list.html`, `taxonomy_single.html`
- `$TAXONOMY/list.html`, `$TAXONOMY/single.html`

## Partials (templates/partials/)
- `head.html` — `<head>` compartido: charset, viewport, `<title>` (usa `site_title`), og:image meta, @font-face, link a stylesheet
- `header.html` — skip-to-content, brand link (`site_title`), nav dinamico iterando taxonomias y `menu_pages`
- `footer.html` — footer con `site_title` y año actual
- `card.html` — card de articulo: titulo, fecha, reading time badge, summary, pills de taxonomia, thumbnail

## Contexto global
Variables disponibles en todas las plantillas:
- `config`: Config global (ver detalle abajo)
- `site`: vista global del sitio (paginas y raiz)
- `taxonomies`: mapa de indices (name -> { taxonomy, terms })
- `menu_pages`: lista de paginas con `osg.menu: true`, ordenadas por titulo (ver Paginas standalone en DESIGN.md)
- `current_path`: ruta relativa iniciando por `/` (no disponible en 404)
- `current_url`: URL absoluta (no disponible en 404)
- `lang`: idioma actual

### Config (`.config`)

Expone todas las claves de `config.yaml` en snake_case:

| Clave | Tipo | Descripcion |
|---|---|---|
| `base_url` | string | URL absoluta del sitio |
| `site_title` | string | Titulo del sitio (default: `"OSG"`) |
| `site_description` | string | Descripcion para meta tags y OG |
| `theme` | string | Nombre del theme activo |
| `color_scheme` | string | `auto`, `light` o `dark` |
| `vault_path` | string | Path al vault de Obsidian |
| `content_dir` | string | Directorio de contenido |
| `public_dir` | string | Directorio de salida |
| `templates_dir` | string | Directorio de templates de usuario |
| `static_dir` | string | Directorio de assets estaticos |
| `themes_dir` | string | Directorio de themes |
| `plugins_dir` | string | Directorio de plugins |
| `plugins_enabled` | []string | Plugins habilitados |
| `sass_dir` | string | Directorio de SASS |
| `content_layout` | string | Patron de URL (`{date}/{slug}`) |
| `include_drafts` | bool | Incluir borradores |
| `compile_sass` | bool | Compilar SASS |
| `tui_prefix` | string | Tecla prefijo TUI |
| `tui_prefix_ms` | int | Timeout prefijo TUI |
| `serve_watch` | bool | Watch en serve |
| `serve_live_reload` | bool | Live reload en serve |
| `serve_debounce_ms` | int | Debounce en ms |
| `build_incremental` | bool | Build incremental |
| `build_cache_dir` | string | Directorio de cache |
| `doctor_profile` | string | Perfil de doctor |
| `logging` | map | `level` y `format` |
| `taxonomies` | []map | Configuracion de taxonomias |

Uso en templates: `{{ .config.site_title }}`, `{{ .config.color_scheme }}`, etc.

El valor de `color_scheme` se usa en el atributo `data-color-scheme` del `<html>`:
```html
<html lang="{{ .lang }}" data-color-scheme="{{ .config.color_scheme }}">
```

## Contexto de pagina

Page (`.page`):

| Campo | Tipo | Descripcion |
|---|---|---|
| `title` | string | Titulo del post |
| `slug` | string | Slug URL-safe |
| `path` | string | Ruta relativa (ej. `/2025/01/15/mi-post/`) |
| `permalink` | string | URL absoluta |
| `date` | time | Fecha de publicacion |
| `updated` | time? | Fecha de actualizacion |
| `draft` | bool | Es borrador |
| `image` | string | Ruta a la imagen (absoluta o placeholder SVG) |
| `summary` | string? | Resumen del post |
| `content` | template.HTML | HTML renderizado (safe, no escapa) |
| `raw_content` | string | Markdown original |
| `word_count` | int | Numero de palabras |
| `reading_time` | int | Minutos de lectura (word_count / 200, min 1) |
| `taxonomies` | map[string][]string | Taxonomias asignadas |
| `menu` | bool | Pagina de menu (excluida de listados, visible en nav) |
| `extra` | map[string]any | Frontmatter extendido (incluye `featured: true`) |

### Notas sobre `image`
- Siempre tiene valor: imagen del vault, URL externa, o placeholder SVG autogenerado
- La ruta es absoluta desde la raiz del sitio (ej. `/2025/09/16/mi-post/foto.jpg` o `/img/placeholder-abc123.svg`)
- Compatible con `og:image` cuando se combina con `base_url`

### Uso tipico en templates
```html
<!-- Hero image en pagina de articulo -->
{{ if .page.image }}
<div class="article-hero">
  <img src="{{ .page.image }}" alt="{{ .page.title }}">
</div>
{{ end }}

<!-- Reading time -->
<span class="reading-time">{{ .page.reading_time }} min</span>

<!-- Word count -->
<span class="word-count">{{ .page.word_count }} words</span>
```

## Contexto de seccion

Section (`.section`):

| Campo | Tipo | Descripcion |
|---|---|---|
| `title` | string | Titulo de la seccion |
| `slug` | string | Slug |
| `path` | string | Ruta relativa |
| `permalink` | string | URL absoluta |
| `content` | template.HTML | HTML del `_index.md` |
| `pages` | []Page | Paginas de la seccion (ordenadas) |
| `subsections` | []Section | Subsecciones |
| `extra` | map[string]any | Frontmatter extendido |
| `featured_page` | Page? | Post destacado para hero |
| `has_source` | bool | Tiene `_index.md` fuente |

### Featured page y orden de pages

- `featured_page`: el post featured mas reciente por fecha. Si no hay featured, el post mas reciente de la seccion.
- `pages`: lista ordenada donde los posts featured restantes aparecen al inicio (antes de los no-featured), seguidos del resto en orden de fecha descendente. El `featured_page` **no** aparece en `pages`.

### Uso tipico en templates
```html
<!-- Hero -->
{{ if .section.featured_page }}
<article class="featured-hero">
  <img src="{{ .section.featured_page.image }}" alt="{{ .section.featured_page.title }}">
  <h2>{{ .section.featured_page.title }}</h2>
</article>
{{ end }}

<!-- Post list -->
{{ range .section.pages }}
<article class="post-card">
  <img src="{{ .image }}" alt="{{ .title }}" class="thumbnail">
  <h3>{{ .title }}</h3>
  <span>{{ .reading_time }} min</span>
</article>
{{ end }}
```

## Contexto de paginacion
Paginator:
- `paginate_by: int`
- `base_url: string`
- `number_pagers: int`
- `first: string`
- `last: string`
- `previous: string?`
- `next: string?`
- `pages: []Page`
- `current_index: int`
- `total_pages: int`

## Contexto de taxonomias
Ver `docs/TAXONOMIES.md`.

## Filtros y funciones

### Base
- `markdown(input)` -> HTML (render de Markdown inline o completo)
- `base64_encode(input)`
- `base64_decode(input)`
- `regex_replace(input, pattern, repl)`
- `num_format(input, locale)`

### Avanzado
Funciones:
- `get_page(path, lang?)`
- `get_section(path, metadata_only?)`
- `get_taxonomy_url(kind, name, lang?)`
- `get_taxonomy(kind)`
- `get_url(path, trailing_slash?, cachebust?)`
- `get_hash(path, sha_type?, base64?)`
- `get_image_metadata(path, allow_missing?)`
- `load_data(path|url, format?, required?)`
- `trans(key, lang?)`

Notas de implementacion:
- `load_data` soporta json/yaml/toml/csv/xml (otros formatos se devuelven como string).
- `trans` devuelve la key (placeholder hasta i18n).
- `get_url(..., cachebust=true)` agrega `?v=` con hash del recurso (usa `get_hash`).

## Implementacion (Go text/template)

- `template.FuncMap` para filtros y funciones.
- Resolver rutas segun prioridad y mantener cache de templates parseados.
- Cada render recibe un contexto raiz con `config` y `current_*`.
- Para subtemplates, usar `template.ParseFiles` o `ParseFS`.
- `content` se devuelve como `template.HTML` para evitar doble escaping.

## Diagrama alto nivel

```
vault -> update-content -> content/ -> parser -> Page/Section models
                                                       |
                                          BuildHierarchy + generatePlaceholders
                                                       |
                                          template resolver -> renderer -> public/
                                                       ^
                                                       | config + theme + taxonomies
```

## Extensibilidad
- Soportar `themes` para override
- Registrar filtros custom via plugins WASM (Phase 5)
- Cache de templates
- Soporte i18n desde config

## Consideraciones
- `text/template` no tiene sandbox; evitar ejecutar funciones peligrosas.
- Validar rutas de plantillas para evitar path traversal.
- `image` siempre tiene valor (placeholder si no hay imagen real) — no necesita check de empty en templates para og:image.
