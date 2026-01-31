# Taxonomias - Especificacion tecnica

## Objetivo
Definir el sistema de taxonomias para agrupar contenido por terminos (tags, categories, etc.).

## Configuracion (config.yaml)
Ejemplo:

```yaml
taxonomies:
  - name: tags
    paginate_by: 10
    paginate_path: page
    feed: true
    render: true
  - name: area
    render: true
```

Campos:
- `name` (obligatorio)
- `paginate_by` (opcional)
- `paginate_path` (opcional)
- `feed` (bool)
- `render` (bool)

## Uso en frontmatter

```yaml
taxonomies:
  tags: ["go", "obsidian"]
  area: ["Filosofia"]
  type: ["concepto"]
```

Nota: En el flujo de Obsidian, OSG tambien detecta automaticamente `tags`, `area` y `type` en el frontmatter y los mapea a taxonomias con esos mismos nombres.

## Objetos de contexto

TaxonomyConfig:
- `name: string`
- `paginate_by: int?`
- `paginate_path: string?`
- `feed: bool`
- `render: bool`

TaxonomyTerm:
- `name: string`
- `slug: string`
- `path: string`
- `permalink: string`
- `pages: []Page`
- `page_count: int`

## Rutas y plantillas
Para cada taxonomia:
- Lista de terminos: `/$TAXONOMY/` -> `templates/$TAXONOMY/list.html` o `taxonomy_list.html`
- Pagina de termino: `/$TAXONOMY/$TERM/` -> `templates/$TAXONOMY/single.html` o `taxonomy_single.html`

Si `render=false`, no se generan paginas.
Si `paginate_by` esta definido, el termino se pagina en `/$TAXONOMY/$TERM/$paginate_path/N/`.

## Flujo de generacion
1) Parsear contenido -> Page
2) Extraer `taxonomies` de frontmatter
3) Agregar por taxonomia y termino
4) Generar paginas list/single
5) Generar feeds por termino si `feed=true` (atom.xml y/o rss.xml si las plantillas existen)

## Diagrama de flujo

```
content -> pages -> taxonomies index -> list pages + term pages -> templates -> public/
```

## Paginacion
Cuando un termino tiene mas paginas que `paginate_by`:
- Construir `Paginator`
- Render `single.html` por pagina
- Exponer `paginator` en contexto

## Mejores practicas / limitaciones
- Muchos terminos incrementan tiempo de build
- Paginacion amplia puede generar miles de paginas
- SEO: evitar contenido duplicado en terminos muy similares

## Extensibilidad
- Permitir filtros custom para ordenar terminos
- Permitir slug rules personalizadas
- Multi-idioma: taxonomia por idioma o prefijos
