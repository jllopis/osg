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

## Starter kit
Para crear un theme base:

```bash
osg theme init my-theme
```

Esto copia el theme por defecto en `themes/my-theme` para que lo uses como base.

Luego actualiza `config.yaml`:

```yaml
theme: my-theme
```

## Notas
- `osg init` asegura que exista `themes/default`.
- Si el directorio del theme ya existe, `osg theme init` falla para evitar sobreescritura.
- El theme embebido se sobrescribe en cada build para mantenerlo sincronizado con el binario.
- Las fuentes se sirven desde `themes/default/static/fonts/` (copiadas a `public/fonts/`).
