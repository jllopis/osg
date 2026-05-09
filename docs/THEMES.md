# Themes

## Estructura
Un theme vive en `themes/<name>/` y puede incluir:

- `templates/` (HTML/XML/TXT) — incluye subdirectorio `partials/`
- `static/` (assets copiados a `public/`) — incluye `fonts/` y `style.css`
- `sass/` (opcional, compilado a CSS)

La resolucion de templates es:
1) `templates/` (usuario)
2) `themes/<name>/templates/`
3) `internal/render/builtins/`

## Theme por defecto

El theme por defecto esta embebido en el binario (`internal/theme/default/`) y se extrae automaticamente a `themes/default/` en cada build via `theme.EnsureDefaultTheme()` con `overwrite=true`. Esto garantiza que el theme siempre este actualizado con la version del binario.

### Paleta de colores: Nord

El theme usa la paleta [Nord](https://www.nordtheme.com/) con dos modos:

**Polar Night** (fondos oscuros):
| Token | Hex | Uso |
|---|---|---|
| nord0 | `#2e3440` | Fondo principal dark |
| nord1 | `#3b4252` | Fondo elevado dark |
| nord2 | `#434c5e` | Bordes dark |
| nord3 | `#4c566a` | Texto secundario dark |

**Snow Storm** (fondos claros / texto):
| Token | Hex | Uso |
|---|---|---|
| nord4 | `#d8dee9` | Texto secundario dark |
| nord5 | `#e5e9f0` | Fondo elevado light |
| nord6 | `#eceff4` | Fondo principal light |

**Frost** (acentos):
| Token | Hex | Uso |
|---|---|---|
| nord7 | `#8fbcbb` | Acento secundario |
| nord8 | `#88c0d0` | Links, acento primario |
| nord9 | `#81a1c1` | Headers, elementos interactivos |
| nord10 | `#5e81ac` | Acento deep |

**Aurora** (highlights):
| Token | Hex | Uso |
|---|---|---|
| nord11 | `#bf616a` | Error, danger |
| nord12 | `#d08770` | Warning |
| nord13 | `#ebcb8b` | Info, highlight |
| nord14 | `#a3be8c` | Success |
| nord15 | `#b48ead` | Decorativo |

### Tipografia

Fuentes self-hosted, embebidas en el binario via `//go:embed`:

- **Inter** (variable, latin) — texto general, ~95 KB woff2
- **JetBrains Mono** (regular, latin) — code blocks, ~20 KB woff2

Ambas usan `font-display: swap` para evitar FOIT.

### Templates y partials

6 templates principales + 4 partials compartidos:

**Templates:**
- `index.html` — homepage con hero featured + lista de posts
- `page.html` — articulo individual con hero image
- `section.html` — listado de seccion
- `404.html` — pagina de error
- `taxonomy_list.html` — listado de taxonomias
- `taxonomy_single.html` — pagina de una taxonomia

**Partials (`templates/partials/`):**
- `head.html` — `<head>` compartido (charset, viewport, title, og:image, @font-face, stylesheet)
- `header.html` — skip-to-content, brand link, nav dinamico desde taxonomias y `menu_pages`
- `footer.html` — footer con site_title y año
- `card.html` — card reutilizable con titulo, fecha, reading time, summary, pills

### CSS (~1000 lineas)

El stylesheet (`static/style.css`) incluye:

- Custom properties con escala de spacing
- `@font-face` para Inter + JetBrains Mono
- `.prose` completo para output de goldmark GFM (headings, blockquotes, code, tables, task lists, images, hr, listas, strikethrough)
- Sticky header con `backdrop-filter: blur`
- Card hover transitions, pill/chip variants
- Tres breakpoints responsive (480px, 640px, 1024px)
- Accesibilidad: skip-link, `:focus-visible`, `prefers-reduced-motion`

## Color scheme

Controlado por `color_scheme` en `config.yaml`:

```yaml
color_scheme: auto   # auto | light | dark
```

### Comportamiento

- **`auto`** (default): dark mode via `@media (prefers-color-scheme: dark)` CSS media query. El `<html>` no tiene `data-color-scheme` forzado, asi que respeta la preferencia del sistema.
- **`light`**: `<html data-color-scheme="light">`. Las reglas CSS dark nunca se aplican gracias a selectores `:not([data-color-scheme="light"])`.
- **`dark`**: `<html data-color-scheme="dark">`. Se activa la regla `html[data-color-scheme="dark"]` que aplica variables dark incondicionalmente.

### Implementacion CSS

```css
/* Forzar dark mode */
html[data-color-scheme="dark"] {
  --bg: var(--nord0);
  --text: var(--nord4);
  /* ... */
}

/* Auto dark mode (respeta preferencia del sistema) */
@media (prefers-color-scheme: dark) {
  html:not([data-color-scheme="light"]):not([data-color-scheme="dark"]) {
    --bg: var(--nord0);
    --text: var(--nord4);
    /* ... */
  }
}
```

No se usa JavaScript para el toggle. Todo se resuelve con CSS y el atributo `data-color-scheme`.

## Imagenes en el theme

### Featured hero (homepage)
El post destacado mas reciente se muestra como hero con imagen a ancho completo. La imagen viene de `featured_page.image` en el contexto de seccion.

### Thumbnails (lista de posts)
Cada post en la lista muestra un thumbnail de su imagen. Si no tiene imagen, se usa un placeholder SVG generado automaticamente.

### Article hero (pagina de articulo)
La pagina individual del articulo muestra la imagen como hero en la cabecera.

### OpenGraph image
Todas las paginas incluyen `<meta property="og:image">` con la imagen del post (o placeholder).

### Placeholders SVG
- Generados deterministicamente a partir del SHA-256 del titulo
- Dimensiones 1200x630 (compatible OpenGraph)
- Fondo gradiente con colores Polar Night
- 5-9 formas geometricas semitransparentes con colores Frost/Aurora
- Sin texto embebido

## Featured posts (multiples)

Cuando multiples posts tienen `featured: true`:

1. El **mas reciente** por fecha se convierte en el hero de la homepage
2. Los demas featured se promueven al **inicio** de la lista de posts
3. Los posts no-featured siguen en orden de fecha descendente
4. Si ningun post es featured, el mas reciente se usa como hero automaticamente

## theme.yaml (metadatos del tema)

Cada tema puede incluir un fichero `theme.yaml` en su raiz con metadatos:

```yaml
name: my-theme
description: A custom Nord-based theme with sidebar layout
author: Your Name
version: "1.0"
min_osg_version: "0.1"
parent: default        # opcional: hereda de otro tema
```

Campos:
- `name` — nombre del tema (obligatorio, se usa como identificador)
- `description` — descripcion breve
- `author` — autor del tema
- `version` — version del tema
- `min_osg_version` — version minima de OSG requerida
- `parent` — nombre del tema padre para herencia (opcional)

## Herencia de temas

Un tema puede declarar `parent: <nombre>` en `theme.yaml` para heredar de otro tema. La herencia es transitiva (child -> parent -> grandparent).

### Cadena de resolucion

Para templates, static, i18n y sass, la resolucion es:

1. **Builtins** (embebidos en el binario) — fallback minimo
2. **Root ancestor theme** (ej. `default`) — primer tema en la cadena
3. **...temas intermedios...** — en orden ascendente
4. **Active theme** (child) — sobreescribe cualquier fichero con el mismo nombre
5. **User overrides** (`templates/`, `static/`, `i18n/`) — maxima prioridad

Ejemplo: si `my-theme` tiene `parent: default`, y el usuario solo define `templates/partials/footer.html` en `my-theme`, heredara todos los demas templates de `default`.

### Deteccion de ciclos

OSG detecta ciclos en la cadena de herencia (ej. A -> B -> A) y reporta un error antes del build.

## Sidebar widgets (homepage)

La portada soporta un layout opcional de 3 columnas con widgets en la columna derecha. Se activa configurando `sidebar_widgets` en `config.yaml`:

```yaml
sidebar_widgets:
  - author
  - newsletter
  - popular
newsletter_action: "https://buttondown.email/api/emails/embed-subscribe/handle"
```

### Comportamiento del layout

- **Activacion:** solo cuando `sidebar_widgets` no esta vacio. Sin la clave, la home se renderiza single-column igual que antes.
- **Desktop (>=1400px):** layout 3-col con grid `240px | 1080px | 240px` centrado. Los widgets viven en la columna derecha con `position: sticky` para que sigan al lector mientras hace scroll.
- **Por debajo de 1400px:** los widgets pasan a apilarse **debajo** del listado de posts (no se ocultan). El ancho se ajusta al de `.container` (1080px max) y se centran horizontalmente, separados del contenido por un margen superior.
- **Centrado del contenido:** en modo desktop la columna central mantiene la misma posicion en viewport con o sin sidebars (grid simetrico, `justify-content: center`).
- **Wrapper transparente:** sin widgets configurados, el layout usa `display: contents` para no introducir cambios visibles en el DOM render.
- **Sidebar izquierda:** reservada para overrides de temas hijos. En v1 no tiene contenido por defecto y se oculta en stacked mode.

### Widgets disponibles

Cada widget es un `partials/widget-<name>.html` independiente y se auto-oculta cuando le faltan datos.

| Widget | Lee de | Se oculta cuando |
|--------|--------|------------------|
| `author` | `author`, `author_bio`, `author_avatar`, `author_url`, `social` | Los tres campos `author/author_bio/author_avatar` estan vacios |
| `newsletter` | `newsletter_action` | `newsletter_action` esta vacio |
| `popular` | `.osg/interactions.db` (top 5 paginas por views) | Interactions deshabilitado, DB inexistente, o sin datos |

El orden en `sidebar_widgets` es el orden de renderizado. Nombres desconocidos hacen fallar `osg build` con error de validacion.

### Sobreescribir widgets o anadir uno nuevo

Para sobreescribir un widget existente, copia `themes/default/templates/partials/widget-<name>.html` a tu tema (o a `templates/`) y modifica.

Para anadir un widget propio sin tocar los existentes, sobreescribe el bloque `sidebar-right`:

```html
{{/* templates/partials/sidebar-right-override.html */}}
{{ define "sidebar-right" }}
  {{ template "partials/widget-author.html" . }}
  {{ template "partials/widget-mio.html" . }}
{{ end }}
```

El bloque por defecto itera `sidebar_widgets`; sobreescribirlo te da control total.

## Bloques sobreescribibles

Los templates principales del tema default usan `{{ block }}` para permitir sobreescritura parcial sin copiar el template completo:

**page.html:**
- `page-header` — cabecera del articulo (breadcrumbs, titulo, meta, summary)
- `page-hero` — imagen hero del articulo
- `page-content` — contenido principal (TOC + prose)
- `page-taxonomies` — pills de taxonomias en el footer
- `page-nav` — navegacion prev/next
- `page-related` — posts relacionados
- `page-scripts` — scripts al final del body (lightbox, progress)

**index.html:**
- `index-featured` — post destacado hero
- `index-intro` — intro de seccion o descripcion del sitio
- `index-recent-cards` — grid de cards (primeros N posts en pagina 1)
- `index-posts` — listado de posts recientes
- `index-sections` — grid de subsecciones
- `sidebar-right` — contenido de la sidebar derecha (solo cuando `sidebar_widgets` esta activa)

**section.html:**
- `section-breadcrumbs` — breadcrumbs
- `section-hero` — titulo y contenido de la seccion
- `section-posts` — listado de posts en cards
- `section-subsections` — grid de subsecciones

Para sobreescribir un bloque, crea un template con `{{ define "block-name" }}...{{ end }}` en tu tema o en `templates/`. Ejemplo para cambiar solo el footer de taxonomias en `page.html`:

```html
{{/* templates/partials/page-taxonomies-override.html */}}
{{ define "page-taxonomies" }}
<footer class="my-custom-footer">
  <p>Custom taxonomy section</p>
</footer>
{{ end }}
```

## Starter kit

### Copia completa (standalone)

```bash
osg theme init my-theme
```

Copia el tema default completo en `themes/my-theme/` para modificarlo libremente.

### Child theme (herencia)

```bash
osg theme init my-child --parent default
```

Crea un tema minimal en `themes/my-child/` con `parent: default` en `theme.yaml` y directorios vacios (`templates/`, `static/`, `i18n/`) listos para sobreescrituras selectivas.

Luego actualiza `config.yaml`:

```yaml
theme: my-child
```

## Listar temas

```bash
osg theme list
```

Muestra todos los temas instalados con su metadata, tema activo marcado con `(active)`, y relacion padre si existe.

## Notas
- `osg init` asegura que exista `themes/default`.
- Si el directorio del theme ya existe, `osg theme init` falla para evitar sobreescritura.
- El theme embebido se sobrescribe en cada build para mantenerlo sincronizado con el binario.
- Las fuentes se sirven desde `themes/default/static/fonts/` (copiadas a `public/fonts/`).
- `osg doctor` valida el `theme.yaml`, la existencia del parent, y detecta ciclos de herencia.
