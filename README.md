# OSG (Obsidian Site Generator)

## Requisitos
- Go 1.25.x

## Uso rapido
Inicializa la estructura y el archivo de configuracion:

```bash
osg init
```

Lanza la TUI (comando por defecto):

```bash
osg
```

TUI con vault preconfigurado:

```bash
osg --vault-path /ruta/al/vault
```

Sincroniza contenido desde el vault:

```bash
osg update-content --vault-path /ruta/al/vault
```

Con include-drafts:

```bash
osg update-content --vault-path /ruta/al/vault --include-drafts
```

Dry run:

```bash
osg update-content --vault-path /ruta/al/vault --dry-run
```

Build HTML:

```bash
osg build
```

Serve local:

```bash
osg serve --addr :1313
```

Serve con watch + live reload:

```bash
osg serve --watch --live-reload
```

Crear un theme starter:

```bash
osg theme init my-theme
```

Luego actualiza `config.yaml` con `theme: my-theme`.

TUI:

```bash
osg tui
```

Wizard en TUI (pasos guiados):

```
wizard on
next
```

Doctor (validacion de config y entorno):

```bash
osg doctor
```

## Makefile
Comandos habituales:

```bash
make tidy
make fmt
make test
make build
make update-content VAULT_PATH=/ruta/al/vault
make serve SERVE_ADDR=:1313
make tui
```

Mostrar version:

```bash
osg version
```

## Configuracion
Por defecto se lee `config.yaml`. Se puede sobreescribir con `-c` o `--config`.

### Campos principales

```yaml
base_url: "https://mi-sitio.com"
site_title: "Mi Blog"
site_description: "Blog personal sobre filosofia y tecnologia"
theme: default
color_scheme: auto          # auto | light | dark
vault_path: "../mi-vault/"
```

### Frontmatter: bloque `osg`

OSG soporta un namespace `osg` en el frontmatter YAML de las notas de Obsidian para controlar la publicacion sin interferir con otros campos:

```yaml
---
title: Mi Post
tags:
  - filosofia
osg:
  publish: true          # true | "draft" | false (omitir = no publicar)
  featured: true         # destacar en homepage como hero
  image: "cabecera.jpg"  # imagen de cabecera (nombre o path relativo en vault)
---
```

**Prioridades de resolucion** (el bloque `osg` siempre gana sobre campos top-level):
- `osg.publish` > `publish`
- `osg.image` > `image` > `cover` > `banner`
- `osg.featured` > `featured`

Los campos top-level siguen funcionando para compatibilidad.

### Color scheme

```yaml
color_scheme: auto   # valor por defecto
```

- `auto`: respeta la preferencia del sistema (dark/light) via CSS media query
- `light`: fuerza modo claro siempre
- `dark`: fuerza modo oscuro siempre

No usa JavaScript. Se implementa con el atributo `data-color-scheme` en `<html>` y reglas CSS.

### Imagenes

OSG maneja imagenes del vault de Obsidian automaticamente:

1. **Imagenes de frontmatter** (`osg.image`): se resuelven por nombre o path relativo, se copian al directorio de salida con rutas absolutas.
2. **Wikilinks de imagen** (`![[foto.png|alt]]`): se detectan en el body y se convierten a Markdown estandar `![alt](foto.png)`.
3. **Placeholders automaticos**: si un post no tiene imagen, se genera un SVG placeholder determinista con patron geometrico usando la paleta Nord.

Las imagenes aparecen en:
- Hero de homepage (post featured)
- Thumbnails en la lista de posts
- Hero de la pagina del articulo
- Meta tag `og:image`

### Featured posts

Cuando multiples posts tienen `osg.featured: true`:
- El mas reciente por fecha se muestra como hero en la homepage
- Los demas featured aparecen al inicio de la lista de posts
- Si ningun post es featured, el mas reciente se usa como hero

### Theme

- `theme: default` usa el tema base con paleta Nord, fonts Inter/JetBrains Mono.
- El theme por defecto se embebe en el binario y se extrae en cada build.
- `osg init` crea `themes/default` si no existe.
- Ver `docs/THEMES.md` para estructura, paleta Nord y personalización.

Sass:
- `compile_sass: true` para compilar `sass/` a `public/`.
- Las carpetas `themes/<theme>/sass` se compilan siempre (requiere `sass` en PATH).

TUI:
- `tui_prefix`: tecla prefijo para atajos (por defecto `space`).
- `tui_prefix_ms`: timeout del prefijo en milisegundos (por defecto `600`).

Serve:
- `serve_watch`: habilita watch y rebuild (por defecto `true`).
- `serve_live_reload`: habilita live reload (por defecto `true`).
- `serve_debounce_ms`: debounce de eventos (por defecto `300`).

Build:
- `build_incremental`: cache incremental del build (por defecto `true`).
- `build_cache_dir`: directorio para el cache (por defecto `.osg/cache`).
  - guarda `build.json` con stamps de contenido/templates/assets/plugins.
- `clean_public`: limpia `public/` en rebuilds completos o si se eliminan contenidos (por defecto `true`).

Doctor:
- `doctor_profile`: `dev` o `prod` (por defecto `dev`).

Plugins WASM:
- Instala el `.wasm` en `plugins/` o usa `osg plugin install <path>`.
- Activa con `plugins_enabled` en config o `osg plugin enable <name>`.
- Desactiva con `osg plugin disable <name>`.
- Ver `docs/PLUGINS.md` para ABI, eventos y lifecycle.
- Ejemplo: `examples/plugins/feed`.
- Search: `examples/plugins/search` genera `search.json` + `search/index.html`.

## Documentacion
- `docs/DESIGN.md` — arquitectura, data flow, decisiones
- `docs/THEMES.md` — estructura de themes, paleta Nord, color scheme
- `docs/TEMPLATES.md` — sistema de plantillas, contexto, filtros
- `docs/PLAN_THEME_UPGRADE.md` — plan del theme profesional (completado)
- `docs/ROADMAP.md` — fases del proyecto
- `docs/QUICKSTART.md` — guia rapida
- `docs/TAXONOMIES.md` — configuracion de taxonomias
- `docs/PLUGINS.md` — sistema de plugins WASM

## Notas
- `tui` es el comando por defecto.
- `build` genera HTML en `public/`.
- Ver `docs/QUICKSTART.md` y `examples/sample-site/` para empezar rapido.
