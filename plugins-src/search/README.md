# Search Plugin (bundled)

Plugin oficial distribuido con OSG. Se embebe en el binario y se habilita
por defecto.

## Funcionalidad

- **MiniSearch**: Motor de busqueda con indice invertido (~10KB gzipped)
- **Full-text search**: Indexa contenido en texto plano (HTML stripped)
- **Multi-field**: Busca en titulo, resumen, contenido y tags
- **Fuzzy matching**: Tolerancia a errores tipograficos (0.2)
- **Prefix search**: Encuentra "philosophy" al escribir "philo"
- **Scoring**: Boost en titulo (3x) > tags (1.5x) > resumen (2x) > contenido
- **Excerpts**: Muestra contexto con terminos resaltados
- **JSON optimizado**: ~10-20% mas pequeno al eliminar HTML

## Archivos generados

| Archivo | Descripcion |
|---------|-------------|
| `public/search.json` | Indice con todos los documentos |
| `public/search/index.html` | Pagina de busqueda dedicada |
| `public/js/search.js` | Modulo JavaScript reutilizable |

## Integracion en templates

### Header search bar

```html
<!-- MiniSearch desde CDN (~10KB gzipped) -->
<script src="https://cdn.jsdelivr.net/npm/minisearch@7.1.0/dist/umd/index.min.js" defer></script>
<script src="/js/search.js" defer></script>
<div class="header-search">
  <input type="search" id="header-search-input" placeholder="Search..." autocomplete="off" />
  <div id="header-search-results" class="search-dropdown"></div>
</div>
<script>
  document.addEventListener('DOMContentLoaded', function() {
    new OSGSearch({
      inputSelector: '#header-search-input',
      resultsSelector: '#header-search-results',
      maxResults: 8,
      minChars: 2
    });
  });
</script>
```

### Pagina dedicada

La pagina `/search/` se genera automaticamente con estilos Nord y funcionalidad
completa (hasta 50 resultados).

## OSGSearch API

```javascript
const search = new OSGSearch({
  inputSelector: '#my-input',     // requerido: input[type="search"]
  resultsSelector: '#my-results', // requerido: contenedor para resultados
  maxResults: 10,                 // opcional: max resultados (default: 10)
  minChars: 2                     // opcional: chars minimos (default: 2)
});
```

### Metodos

| Metodo | Descripcion |
|--------|-------------|
| `search(query)` | Ejecuta busqueda manualmente |
| `show()` | Muestra el dropdown |
| `hide()` | Oculta el dropdown |

### Eventos de teclado

- `ArrowDown` / `ArrowUp`: Navegar resultados
- `Enter`: Ir a pagina seleccionada
- `Escape`: Cerrar dropdown

## Build

```bash
./build.sh
```

El script:
1. Detecta automaticamente el target WASI (`wasm32-wasi` o `wasm32-wasip1`)
2. Compila con `cargo build --release`
3. Copia el `.wasm` a `internal/plugin/bundled/search.wasm`

Tambien desde la raiz del proyecto:

```bash
make plugins-search
```

**IMPORTANTE**: El target debe ser WASI (`wasm32-wasip1`), no `wasm32-unknown-unknown`.
Sin WASI el plugin no tiene acceso al filesystem.

## Desarrollo

El codigo fuente esta en `src/lib.rs`:

1. Escucha evento `build.finished`
2. Extrae `site.pages` del payload
3. Para cada pagina:
   - Extrae: title, summary, content (HTML), permalink, date, taxonomies
   - Convierte HTML a texto plano con `strip_html()` (reduce JSON 10-20%)
4. Genera `search.json` con documentos (texto plano, sin tags HTML)
5. Genera `search/index.html` con estilos Nord + script MiniSearch CDN
6. Genera `js/search.js` con la clase OSGSearch (usa MiniSearch)

## Sobreescribir

Los usuarios pueden sobreescribir el plugin bundled colocando su propio
`search.wasm` en el directorio `plugins/` del sitio. OSG no sobreescribe
plugins existentes.

## Dependencias

- **MiniSearch 7.1.0**: Cargado desde CDN (https://cdn.jsdelivr.net/npm/minisearch@7.1.0/)
  - Tamano: ~10KB gzipped
  - Features: inverted index, fuzzy search, prefix search, field boosting
  - Sin dependencias de build, cargado en runtime
