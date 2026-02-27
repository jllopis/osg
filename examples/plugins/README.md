# Example Plugins

Ejemplos de referencia para aprender a escribir plugins WASM para OSG.

## feed/

Plugin de ejemplo que genera un RSS basico. Sirve como referencia para:
- Escuchar eventos del build pipeline
- Acceder al payload (config, site, pages)
- Escribir ficheros via WASI filesystem

> Los feeds RSS/Atom se generan de forma nativa por OSG.
> Este plugin es solo un ejemplo educativo.

## Plugins oficiales

Los plugins distribuidos con OSG viven en `plugins-src/` en la raiz del proyecto:
- **search**: Genera indice de busqueda JSON + pagina HTML con busqueda client-side.
  Se embebe en el binario y se habilita por defecto.

## Escribir tu propio plugin

```bash
osg plugin init my-plugin             # scaffold Rust
osg plugin init my-plugin --lang=go   # scaffold TinyGo (futuro)
```

Ver `docs/PLUGINS.md` para la especificacion completa.
