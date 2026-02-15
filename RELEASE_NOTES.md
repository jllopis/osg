# Release Notes

## v0.3.0 - 2026-02-14

### Theme profesional con paleta Nord

El theme por defecto ha sido completamente reescrito con un diseño profesional centrado en legibilidad, minimalismo y accesibilidad.

**Nuevo diseño visual:**
- **Paleta de colores Nord** — reemplaza la paleta stone/amber anterior. Polar Night para fondos oscuros, Snow Storm para claros, Frost para acentos, Aurora para highlights.
- **Tipografía self-hosted** — Inter (variable) para texto general + JetBrains Mono para bloques de código, embebidas en el binario.
- **CSS reescrito** (~1000 líneas) — sticky header, `.prose` completo para GFM, card hover transitions, 3 breakpoints responsive, accesibilidad (skip-link, focus-visible, prefers-reduced-motion).
- **Partials DRY** — head, header, footer y card compartidos en las 6 plantillas.

### Bloque `osg` en frontmatter

Nuevo namespace `osg` en el frontmatter YAML para controlar publicación y metadatos sin interferir con campos de Obsidian:

```yaml
osg:
  publish: true       # true | "draft" | false
  featured: true      # destacar en homepage
  image: "foto.jpg"   # imagen de cabecera
```

Los campos `osg.*` tienen prioridad sobre campos top-level equivalentes. Los campos legacy siguen funcionando.

### Image pipeline completo

- **Índice de imágenes del vault** — indexa automáticamente todas las imágenes por basename y path relativo.
- **Reescritura de wikilinks** — `![[foto.png|alt]]` se convierte a `![alt](foto.png)` con resolución y copia automática.
- **Imágenes en frontmatter** — `osg.image` resuelve y copia imágenes del vault con rutas absolutas.
- **Placeholders SVG** — generados deterministamente con patrón geométrico Nord (1200×630, compatible OpenGraph) para posts sin imagen.
- **Imágenes en todos los contextos** — hero en homepage, thumbnails en lista, hero en artículo, meta `og:image`.

### Featured posts múltiples

Cuando varios posts tienen `osg.featured: true`:
- El más reciente por fecha se muestra como hero en la homepage
- Los demás featured se promueven al inicio de la lista de posts
- Si ningún post es featured, el más reciente se usa como hero

### Color scheme configurable

Nueva opción `color_scheme` en config.yaml:
- `auto` (default): respeta preferencia del sistema vía CSS media query
- `light`: fuerza modo claro
- `dark`: fuerza modo oscuro

Sin JavaScript — usa atributo `data-color-scheme` en `<html>` con reglas CSS.

### Nuevos campos en templates

- **Page**: `image`, `word_count`, `reading_time`
- **Section**: `featured_page`, `has_source`
- **Config**: `site_title`, `site_description`, `color_scheme`

### Bajo el capó

- Nuevos paquetes: `internal/placeholder`, `internal/wikilink`
- `internal/vault`: `BuildImageIndex`, `ImageIndex.Resolve`
- `internal/publish`: `GetOSGBlock`, `ShouldPublish` reescrito
- `internal/content`: `NormalizeFrontmatter` lee bloque `osg`
- Tests: publish (10), content (6), wikilink (4), vault (2), placeholder (6)

---

## v0.2.0 - 2026-02-10

### Novedades Brillantes
- **TUI por defecto:** Al ejecutar `osg` se abre la interfaz visual con acciones rápidas y estado en tiempo real.
- **Plugins WASM listos para usar:** CLI/TUI para instalar, habilitar y crear plugins, con ejemplos de RSS y búsqueda.
- **Quickstart y sample site:** Un sitio de ejemplo y guía rápida para arrancar sin depender de un vault real.

### Ajustes y Mejoras
- **Build incremental con cache:** Evita renders innecesarios y acelera iteraciones.
- **Live reload y watch:** Previsualización fluida con rebuild automático.
- **Theme starter kit:** Comando para generar un tema base y empezar a personalizar rápido.
- **TUI más clara y ergonómica:** Paneles, atajos y badges de estado para seguimiento visual.

### Bichos Aplastados
- **Plugins Rust:** Correcciones en targets WASM y strings crudas en el ejemplo de búsqueda.
- **CLI theme init:** Mejora en el manejo de comandos anidados.

### Bajo el Capó
- **`osg doctor`:** Validación de config y entorno con warnings útiles.
- **Limpieza de `public/`:** Evita archivos stale cuando se eliminan contenidos.

## v0.2.1 - 2026-02-10

### Ajustes y Mejoras
- **TUI guiada:** Panel de estado más claro (next action, serve badge, resumen de build/doctor) y salida reciente separada de logs.
- **Perfiles en `doctor`:** `dev|prod` con severidad ajustada y feedback más accionable.
- **Theme base refinado:** Estadísticas en hero y mejoras visuales en cards.

## v0.2.2 - 2026-02-10

### Ajustes y Mejoras
- **Eventos TUI estructurados:** La salida ya no muestra logs crudos, sino eventos legibles y un panel de estado guiado.

## v0.2.3 - 2026-02-10

### Ajustes y Mejoras
- **Wizard en TUI:** comando `next` y control `wizard on/off` para seguir el flujo init → update → build → serve.
- **Doctor más estricto:** validación de `base_url`, checks de watch/live‑reload, sass y templates de theme.

## v0.2.4 - 2026-02-10

### Ajustes y Mejoras
- **TUI mejorada:** panel de flujo y alertas, wizard con `next` y atajos `W/N`.
- **Doctor tests:** cobertura para perfiles `dev/prod`.
- **Theme base:** chips de tags y pills enlazadas.
