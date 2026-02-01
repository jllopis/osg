# OSG (Obsidian Site Generator)

## Requisitos
- Go 1.25.x

## Uso rapido
Inicializa la estructura y el archivo de configuracion:

```bash
osg init
```

Sincroniza contenido desde el vault (comando por defecto):

```bash
osg --vault-path /ruta/al/vault
```

Con include-drafts:

```bash
osg --vault-path /ruta/al/vault --include-drafts
```

Dry run:

```bash
osg --vault-path /ruta/al/vault --dry-run
```

Comando explicito:

```bash
osg update-content --vault-path /ruta/al/vault
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

Theme:
- `theme: default` usa el tema base en `themes/default` (templates + CSS).
- `osg init` crea `themes/default` si no existe.
- Ver `docs/THEMES.md` para estructura y scaffold.

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

Plugins WASM:
- Coloca `.wasm` en `plugins/`.
- Ver `docs/PLUGINS.md` para ABI y eventos.
- Ejemplo: `examples/plugins/feed`.

## Notas
- `update-content` es el comando por defecto.
- `build` genera HTML en `public/`.
