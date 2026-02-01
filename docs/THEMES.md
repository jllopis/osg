# Themes

## Estructura
Un theme vive en `themes/<name>/` y puede incluir:

- `templates/` (HTML/XML/TXT)
- `static/` (assets copiados a `public/`)
- `sass/` (opcional, compilado a CSS)

La resolucion de templates es:
1) `templates/` (usuario)
2) `themes/<name>/templates/`
3) `internal/render/builtins/`

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
