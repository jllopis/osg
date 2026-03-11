# ADR-002: Dual-file sync para el tema por defecto

**Estado:** Aceptado
**Fecha:** 2025-07-15
**Contexto:** internal/theme/default.go, internal/theme/default/

## Contexto

OSG incluye un tema por defecto (templates, CSS, JS, i18n, fuentes) que debe
cumplir dos requisitos contradictorios:

1. **Autonomia del binario**: un `osg init` debe funcionar sin clonar repos
   externos ni descargar ficheros.
2. **Editabilidad en disco**: el usuario necesita poder personalizar templates y
   assets; ademas, `osg serve --watch` necesita ficheros reales que monitorizar.

## Decision

Mantenemos **dos copias identicas** del tema por defecto:

| Ubicacion | Proposito |
|---|---|
| `internal/theme/default/` | Embebida en el binario via `//go:embed default` |
| `themes/default/` | Copia de trabajo en disco, editable por el usuario |

La sincronizacion la realiza `installTheme()` en `default.go`, invocada en dos
puntos:

- **`EnsureDefaultTheme(themesDir)`**: cada build y cada `osg init`. Usa
  `overwrite=true` para mantener la copia en disco actualizada con la version
  del binario.
- **`ScaffoldTheme(themesDir, name)`**: al crear un tema hijo con
  `osg theme init`. Usa `overwrite=false` para no pisar ediciones del usuario.

### Optimizacion de idempotencia

`installTheme()` compara el contenido (`bytes.Equal`) antes de escribir:

```go
if existing, readErr := os.ReadFile(destPath); readErr == nil && bytes.Equal(existing, data) {
    return nil // no toca el mtime
}
```

**Razon critica**: sin esta comprobacion, `EnsureDefaultTheme()` al inicio de
cada build modificaria el mtime de los ficheros del tema, lo que dispararia el
file watcher de `osg serve`, que lanzaria otro build, generando un **bucle
infinito de rebuilds**.

## Consecuencias

### Positivas

- **Zero-dependency bootstrap**: `osg init` funciona sin red ni repos externos.
- **Auto-actualizacion**: al actualizar el binario, el tema se sincroniza
  automaticamente en el proximo build.
- **File watching funcional**: `osg serve` puede monitorizar ficheros reales en
  `themes/default/` para hot reload.
- **Herencia de temas**: `ResolveChain()` resuelve la cadena de herencia
  recorriendo el disco, lo cual requiere ficheros reales.
- **Compatibilidad con el pipeline de assets**: `PrepareWithChain()` copia
  estaticos y compila Sass desde rutas en disco.

### Negativas

- **Duplicacion de ficheros en el repo**: cada cambio al tema debe reflejarse
  en dos directorios. Esto se mitiga con el patron "dual-file sync" documentado
  en TASKS.md (cada feature que toca el tema lista explicitamente ambas rutas).
- **Sobreescritura de ediciones**: `EnsureDefaultTheme` con `overwrite=true`
  reemplaza ediciones del usuario en `themes/default/` cada build. El usuario
  debe crear un tema hijo para personalizaciones persistentes.

## Alternativas consideradas

1. **Solo embebido (sin copia en disco)**: Imposible. El file watcher, el
   pipeline de Sass, y la herencia de temas requieren ficheros reales. No se
   puede hacer `filepath.WalkDir()` sobre un `embed.FS` con las mismas
   garantias.

2. **Solo en disco (sin embed)**: El binario no seria auto-contenido. El
   usuario tendria que clonar un repo de temas o descargar un zip tras
   instalar, degradando la experiencia de primer uso.

3. **Git submodule para el tema**: Anade complejidad al workflow de desarrollo
   y rompe el modelo de binario unico. El tema es parte integral del producto,
   no un componente externo.

4. **Generar en disco solo en `osg init`**: El tema quedaria desactualizado
   tras upgrades del binario. No habria forma de aplicar fixes de seguridad o
   mejoras de CSS sin intervencion manual.
