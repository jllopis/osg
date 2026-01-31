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

O usando base + nombre de vault:

```bash
osg --vault-path /ruta/al/vault --include-drafts
```

Dry run:

```bash
osg --vault-path /ruta/al/vault --dry-run
```

Build HTML:

```bash
osg build
```

Serve local:

```bash
osg serve --addr :1313
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
```

Mostrar version:

```bash
osg version
```

## Configuracion
Por defecto se lee `config.yaml`. Se puede sobreescribir con `-c` o `--config`.

Sass:
- `compile_sass: true` para compilar `sass/` a `public/`.
- Las carpetas `themes/<theme>/sass` se compilan siempre (requiere `sass` en PATH).

## Notas
- `update-content` es el comando por defecto.
- `build` se implementara en fase 2.
