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

## 6) Crear un post nuevo

```bash
osg new "Mi Nuevo Post"
```

Crea un fichero `Mi Nuevo Post.md` en el vault con frontmatter pre-configurado:

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
  # abstract: ""       # Summary/excerpt override
  # author: ""         # Author override
---
```

Todos los campos `osg:` aparecen como comentarios YAML listos para descomentar.

**Opciones:**

```bash
osg new "Post" --tags tech,go    # Con tags
osg new "Post" --publish         # Publicado (no draft)
osg new "Post" --no-editor       # No abrir editor despues de crear
osg new "Post" --dry-run         # Solo muestra que haria
```

Tras crear el fichero, `osg new` abre automaticamente el editor configurado
(`default_editor` en config.yaml o variable de entorno `$EDITOR`). Si no hay
editor configurado, no hace nada. Usa `--no-editor` para omitir este paso.

## 7) Shortcodes

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

## 8) Ejemplo rapido
En `examples/sample-site/` hay un sitio minimo con `config.yaml` y un contenido de ejemplo.
```bash
cd examples/sample-site
osg build
osg serve
```
