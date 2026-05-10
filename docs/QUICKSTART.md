# Quickstart

## 1) Crear un sitio
```bash
osg init
```

## 2) Importar contenido desde Obsidian
```bash
osg update-content --vault-path /ruta/a/mi-vault
```

## 3) Build HTML
```bash
osg build
```

## 4) Previsualizar
```bash
osg serve
```

## 5) TUI
```bash
osg tui
```

Por defecto, `osg` sin comando lanza la TUI.

## 5b) Web UI (dashboard)

```bash
osg ui
```

Levanta un dashboard web en `127.0.0.1:1314` (loopback-only, rechaza
binds publicos). Desde el navegador puedes:

- Ver inventario del vault, plugins, assets, scheduler audit y servicios
- Editar `osg.summary` por pagina (con boton "Sugerencia IA" que llama
  al LLM en directo)
- Ejecutar el pipeline `init → update-content → check → build → deploy`
  via `/actions` con "Run from here"
- Inspeccionar logs en vivo en el drawer inferior
- Consultar `/history` con filtros por nombre/kind/status

La TUI y la Web UI son independientes y comparten los mismos comandos
internos — uses la que prefieras.

## 5c) Mantener `osg ui` siempre activo

Para que el flujo de auto-publish funcione (drafts con `osg.publish_at`)
necesitas que el scheduler service esté corriendo de forma persistente.
Dos pasos:

**1. Auto-arrancar servicios al levantar `osg ui`** — añade `autostart`
a `config.yaml`:

```yaml
ui:
  addr: ":1314"
  autostart:
    - scheduler   # imprescindible para auto-publish
    - watcher     # opcional: rebuild automatico al guardar en el vault
```

**2. Instalar `osg ui` como servicio del sistema** — para que arranque
al login y se reinicie ante crashes:

```bash
osg service install
```

En macOS escribe un LaunchAgent en `~/Library/LaunchAgents/`; en Linux
un servicio `--user` de systemd en `~/.config/systemd/user/`. Ambos
son "user-level" — no piden `sudo`. Los comandos del día a día son:

```bash
osg service status      # ver estado
osg service stop        # parar (la unit file se queda)
osg service start       # rearrancar
osg service uninstall   # quitar la unit del sistema
```

Documentacion completa, opciones (`--workdir`, `--config`, `--exec`,
`--no-start`), logs y troubleshooting en
[docs/SERVICE.md](SERVICE.md).

## 6) Crear un post nuevo

```bash
osg new "Mi Nuevo Post"
```

Crea un fichero `Mi Nuevo Post.md` en el vault (o en la subcarpeta configurada
en `new_notes_dir`) con frontmatter pre-configurado:

```yaml
---
title: Mi Nuevo Post
created: 2025-03-06 10:30
osg:
  publish: draft
  # title: ""          # Override page title (highest precedence)
  # image: ""          # Featured/hero image path
  # featured: false    # Mark as featured post
  # path: ""           # Custom output path override
  # permalink: ""      # URL pattern ({date}, {year}, {month}, {day}, {slug}, {title})
  # menu: false        # Add to navigation menu
  # summary: ""        # Summary/excerpt override (alias of abstract)
  # abstract: ""       # Legacy alias of summary
  # author: ""         # Author override
  # publish_at:        # Auto-publish date (works with publish:draft)
---
```

Todos los campos `osg:` aparecen como comentarios YAML listos para descomentar.

### Programar la publicacion de un draft

Mantente con `osg.publish: draft` y añade `osg.publish_at: 2026-06-15`.
El scheduler service (incluido en `osg ui` o ejecutado independiente
con la operacion `scheduler`) despertara el dia indicado, reescribira
el frontmatter del vault para flipear `osg.publish: draft → true`, y
disparara `update-content` + `build`. Mientras tanto el post:

- Se sincroniza a `content/` para que el scheduler lo encuentre
- Se queda fuera de `public/` (build sigue tratandolo como draft)
- Tampoco se sube en deploys (esta registrado en `deferred_publications`)

Si cambias de opinion, basta con borrar el `publish_at` o eliminar el
fichero del vault antes de la fecha.

**Opciones:**

```bash
osg new "Post" --tags tech,go    # Con tags
osg new "Post" --publish         # Publicado (no draft)
osg new "Post" --no-editor       # No abrir editor despues de crear
osg new "Post" --notes-dir 02_Notes  # Crear en subcarpeta del vault
osg new "Post" --dry-run         # Solo muestra que haria
```

Tras crear el fichero, `osg new` abre automaticamente el editor configurado
(`default_editor` en config.yaml o variable de entorno `$EDITOR`). Si no hay
editor configurado, no hace nada. Usa `--no-editor` para omitir este paso.

## 7) Importar desde WordPress o Hugo

Puedes migrar contenido existente a OSG:

**WordPress** (exportar desde WP Admin → Herramientas → Exportar):
```bash
osg import wordpress export.xml             # importar posts
osg import wordpress export.xml --dry-run   # previsualizar sin escribir
```

**Hugo** (apunta al directorio `content/` del proyecto Hugo):
```bash
osg import hugo /ruta/al/hugo/content       # importar posts
osg import hugo /ruta/al/hugo/content --dry-run
```

Ambos importadores:
- Convierten frontmatter (YAML/TOML) al formato OSG
- Preservan tags, categorias y fechas
- WordPress: convierte HTML a Markdown automaticamente
- Hugo: soporta frontmatter YAML (`---`) y TOML (`+++`)
- Los ficheros se escriben en `content/` con ruta `YYYY/MM/slug.md`

## 8) Auditar el sitio generado

Tras un build, valida la calidad del HTML generado:

```bash
osg audit                 # muestra informe en terminal
osg audit --json          # salida JSON (ideal para CI)
```

Comprobaciones:
- Imagenes sin atributo `alt` (accesibilidad)
- Jerarquia de headings incorrecta (ej. `h1` → `h3` sin `h2`)
- Ficheros HTML muy grandes (>500KB)
- Scripts inline excesivos (>10KB)

El comando devuelve exit code 1 si hay errores, ideal para pipelines CI/CD.

## 9) Webhooks

Recibe notificaciones HTTP POST en eventos de build y deploy:

```yaml
webhooks:
  - url: "https://hooks.example.com/osg"
    events: ["build.success", "deploy.success"]
    secret: "mi-secreto"
```

Eventos disponibles: `build.success`, `build.failure`, `deploy.success`.
Si `events` esta vacio, recibe todos los eventos.
Si `secret` esta configurado, se envia un header `X-OSG-Signature: sha256=<hmac>`.

## 10) Analytics

Analytics ligero, first-party, sin scripts de terceros:

```yaml
analytics: true
```

Requiere el servidor API (`osg serve --api` o `osg api`). Genera automaticamente
`/js/analytics.js` que respeta DNT (Do Not Track). Datos almacenados en SQLite.

Endpoints:
- `POST /api/v1/analytics/hit` — registrar pageview
- `GET /api/v1/analytics/summary` — datos agregados (paginas, referrers, navegadores)

## 11) Shortcodes

Puedes usar shortcodes en tus notas para insertar contenido enriquecido:

```markdown
{{< youtube "dQw4w9WgXcQ" />}}

{{< note "Importante" >}}
Recuerda configurar tu `vault_path` antes del primer build.
{{< /note >}}

{{< tabs >}}
{{< tab "Bash" >}}
osg build && osg serve
{{< /tab >}}
{{< tab "Make" >}}
make build && make serve
{{< /tab >}}
{{< /tabs >}}
```

Shortcodes disponibles: `note`, `warning`, `tip`, `details`, `figure`, `tabs`/`tab`,
`youtube`, `twitter`, `codepen`.

Guia completa: [SHORTCODES.md](SHORTCODES.md)

## 12) Ejemplo rapido
En `examples/sample-site/` hay un sitio minimo con `config.yaml` y un contenido de ejemplo.
```bash
cd examples/sample-site
osg build
osg serve
```
