# Release Process

## Versionado

OSG sigue [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (`v1.0.0`): cambios incompatibles en config YAML, frontmatter, CLI flags, o ABI de plugins WASM
- **MINOR** (`v0.3.0`): nuevas funcionalidades retrocompatibles (comandos, shortcodes, hooks, config fields)
- **PATCH** (`v0.2.1`): correcciones de bugs sin cambios en la interfaz publica

### Pre-releases

Versiones `v0.x.y` son pre-release. La API publica (config, CLI, ABI WASM) puede
cambiar entre releases menores. A partir de `v1.0.0` se garantiza estabilidad.

## Inyeccion de version

La version se inyecta en tiempo de compilacion via `-ldflags`:

```go
// internal/app/version.go
var (
    Version = "dev"    // git describe --tags
    Commit  = "none"   // git rev-parse --short HEAD
    Date    = ""       // fecha ISO 8601
)
```

El Makefile calcula estos valores automaticamente:

```make
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X osg/internal/app.Version=$(VERSION) ..."
```

En builds locales sin tag, `Version` muestra el hash corto (e.g. `v0.99.1-3-g2ebc700`).

## Proceso de release

### 1. Preparar el release

```bash
# Verificar que CI esta verde en master
gh run list --branch master --limit 5

# Verificar que los tests pasan localmente
make test
make lint

# Compilar plugins WASM (si hay cambios en plugins-src/)
make plugins-all
```

### 2. Crear el tag

```bash
git tag -a v0.X.Y -m "Release v0.X.Y: breve descripcion"
git push origin v0.X.Y
```

### 3. Pipeline automatico (GitHub Actions)

El tag `v*` dispara el workflow de CI (`.github/workflows/ci.yml`):

1. **test**: `go test -race ./...`
2. **build**: cross-compile para 5 plataformas (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
3. **build-plugins**: compila plugins WASM desde `plugins-src/`
4. **lint**: `golangci-lint run ./...`
5. **vet**: `go vet` (excluye `plugin/sdk` por uso intencional de `unsafe.Pointer`, ver ADR-001)
6. **release** (solo tags): recopila binarios + plugins, genera checksums SHA-256, crea GitHub Release

### 4. Artefactos del release

Cada release incluye:

| Artefacto | Formato | Descripcion |
|---|---|---|
| `osg_vX.Y.Z_linux_amd64.tar.gz` | tar.gz | Binario Linux x86_64 |
| `osg_vX.Y.Z_linux_arm64.tar.gz` | tar.gz | Binario Linux ARM64 |
| `osg_vX.Y.Z_darwin_amd64.tar.gz` | tar.gz | Binario macOS Intel |
| `osg_vX.Y.Z_darwin_arm64.tar.gz` | tar.gz | Binario macOS Apple Silicon |
| `osg_vX.Y.Z_windows_amd64.zip` | zip | Binario Windows x86_64 |
| `search.wasm` | wasm | Plugin de busqueda (bundled) |
| `llmstxt.wasm` | wasm | Plugin llms.txt |
| `mermaid.wasm` | wasm | Plugin Mermaid diagrams |
| `archives.wasm` | wasm | Plugin archives |
| `checksums.txt` | texto | SHA-256 de todos los artefactos |

### 5. GoReleaser (alternativo)

El proyecto tambien tiene `.goreleaser.yml` configurado para builds estructurados:

```bash
goreleaser release --clean
```

GoReleaser genera los mismos artefactos con changelog automatico (filtra commits
de docs, tests y CI).

## Politica de linting

- **Zero tolerance en CI**: `golangci-lint run ./...` debe pasar sin errores.
- **`//nolint` solo con justificacion**: cada supresion debe tener un comentario
  explicando el motivo. Las decisiones no obvias se documentan en un ADR
  (ver `docs/adr/001-unsafe-pointer-wasm-sdk.md`).
- **`go vet` excluye `plugin/sdk`**: por uso intencional de `unsafe.Pointer`
  requerido por el ABI WASM (ver ADR-001).

## Linters habilitados

Configuracion en `.golangci.yml` (golangci-lint v2):

- `errcheck`: errores no comprobados
- `govet`: analisis estatico del compilador
- `staticcheck`: bugs, simplificaciones, performance
- `unused`: codigo muerto
- `ineffassign`: asignaciones ineficaces
- `gocritic`: patrones idiomaticos
- `misspell`: errores ortograficos en comentarios/strings
