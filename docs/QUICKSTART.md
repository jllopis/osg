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

## 6) Shortcodes

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

## Ejemplo rapido
En `examples/sample-site/` hay un sitio minimo con `config.yaml` y un contenido de ejemplo.
```bash
cd examples/sample-site
osg build
osg serve
```
