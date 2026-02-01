# Templates - Especificacion tecnica

## Objetivo
Definir el sistema de plantillas de OSG para renderizar paginas HTML desde contenido Markdown.

## Alcance por fases
- Phase 2 (base): carga de templates, defaults (index/section/page), contexto base, render HTML.
- Phase 3 (avanzado): feeds, sitemap, robots, 404, paginacion, taxonomias, load_data, get_image_metadata.

## Estructura de directorios
- `templates/`: plantillas de usuario
- `themes/<theme>/templates/`: plantillas del tema (si aplica)
- `internal/render/builtins/`: plantillas incorporadas (embebidas)

## Resolucion de plantillas (prioridad)
1) Usuario: `templates/` (raiz y subdirectorios)
2) Tema: `themes/<theme>/templates/`
3) Built-in: `internal/templates/builtins/`

Si existe un archivo con el mismo nombre en un nivel superior, se usa ese.

## Plantillas estandar
- `index.html` -> homepage
- `section.html` -> secciones
- `page.html` -> paginas

## Plantillas especiales (Phase 3)
- `atom.xml`, `rss.xml`
- `sitemap.xml`, `split_sitemap_index.xml`
- `robots.txt`
- `404.html`
- `taxonomy_list.html`, `taxonomy_single.html`
- `$TAXONOMY/list.html`, `$TAXONOMY/single.html`

## Contexto global
Variables disponibles en todas las plantillas:
- `config`: Config global (incluye `base_url` y `theme`)
- `site`: vista global del sitio (paginas y raiz)
- `taxonomies`: mapa de indices (name -> { taxonomy, terms })
- `current_path`: ruta relativa iniciando por `/` (no disponible en 404)
- `current_url`: URL absoluta (no disponible en 404)
- `lang`: idioma actual

Notas:
- `config` expone claves en snake_case, igual que `config.yaml`.

## Contexto de pagina
Estructura propuesta:

Page:
- `title: string`
- `slug: string`
- `path: string` (ruta relativa)
- `permalink: string` (URL absoluta)
- `date: time`
- `updated: time?`
- `draft: bool`
- `summary: string?`
- `content: string` (HTML renderizado)
- `raw_content: string` (Markdown original)
- `taxonomies: map[string][]string`
- `extra: map[string]any` (frontmatter extendido)

## Contexto de seccion
Section:
- `title: string`
- `slug: string`
- `path: string`
- `permalink: string`
- `content: string` (HTML del _index.md)
- `pages: []Page`
- `subsections: []Section`
- `extra: map[string]any`

## Contexto de paginacion (Phase 3)
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

## Contexto de taxonomias (Phase 3)
Ver `docs/TAXONOMIES.md`.

## Filtros y funciones

### Phase 2 (minimo)
- `markdown(input)` -> HTML (render de Markdown inline o completo)
- `base64_encode(input)`
- `base64_decode(input)`
- `regex_replace(input, pattern, repl)`
- `num_format(input, locale)`

### Phase 3 (avanzado)
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

## Diagrama alto nivel

```
content/ -> parser -> Page/Section models -> template resolver -> renderer -> public/
                                    ^
                                    | config + theme
```

## Extensibilidad
- Soportar `themes` para override
- Registrar filtros custom via plugins WASM (Phase 5)
- Cache de templates
- Soporte i18n desde config

## Consideraciones
- `text/template` no tiene sandbox; evitar ejecutar funciones peligrosas.
- Validar rutas de plantillas para evitar path traversal.
