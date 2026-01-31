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
osg --obsidian-vault-base /ruta/a/vaults --vault MiVault
```

Incluye drafts:

```bash
osg --vault-path /ruta/al/vault --include-drafts
```

Dry run:

```bash
osg --vault-path /ruta/al/vault --dry-run
```

## Makefile
Comandos habituales:

```bash
make tidy
make fmt
make test
make build
make update-content VAULT_PATH=/ruta/al/vault
```

Mostrar version:

```bash
osg version
```

## Configuracion
Por defecto se lee `config.yaml`. Se puede sobreescribir con `-c` o `--config`.

## Notas
- `update-content` es el comando por defecto.
- `build` se implementara en fase 2.
