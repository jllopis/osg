# OSG -> Obsidian Site Generator

## Definición

OSG es un programa para generar páginas web estáticas, con una funcionalidad elevada y una simplicidad para su gestión.

Para provisionar contenido en OSG, se utilizará una funcionalidad que obtenga los ficheros markdown de un _Vault_ de Obsidian, y realice las transformaciones necesarias para adaptar el contenido al generador.

Eres un desarrollador experto en Go, WASM, Obsidian y en diseño de herramientas estáticas orientadas a publicación de contenido.

Tu tarea es implementar una herramienta en **Go** que:

1. Lea ficheros Markdown de un **vault de Obsidian**, especificando la carpeta en la que reside el Vault.
2. Analice su frontmatter **YAML**.
3. Seleccione únicamente aquellos ficheros que tengan:
   publish: true
   publish: "true"
   publish: "draft"
4. Convierta su frontmatter YAML en frontmatter el adecuado para publicar. Este frontmatter se ha de definir.
    - Se han de eliminar los atributos que no sean necesarios para la publicación.
    - Se han de añadir los necesarios para la publicación.
    - Se han de obtener todos los datos posibles del frontmatter original.
5 . Copie cada documento seleccionado a la carpeta `content/{date YYYY/MM/DD}/{slug}`, de modo que cada post tenga su propia carpeta y el contenido se organice de manera natural por fecha. Esta, como el resto de opciones, han de ser modificables a través de fichero de configuración, variables de entorno y parámetros en command line.
6. El programa constará de dos partes:
   1. CLI core compilable de modo nativo, A su vez, este podrá lanzarse exclusivamente en línea de comando (para uso en automatizaciones o CI/CD) y en modo TUI.
   2. Sistema de plugins con **WASM**, de modo que aporte seguridad y permita usar distintos lenguajes para crear los plugins. El sistema de plugins también has de diseñarlo.

Debes implementar **todo el proyecto**, con módulos, estructuras de datos, parsing, serialización, errores, CLI, sistema de plugins y bindings WASM.

-------------------------------------------------------------------------------
REQUISITOS FUNCIONALES
-------------------------------------------------------------------------------

### 1. LECTURA DEL VAULT

Implementar un módulo que:

- Recupere el listado de archivos Markdown del vault.
- Recupere el contenido completo de un archivo.

Parámetros configurables:

- -obsidian-vault-base
- -vault <nombre del vault>

### 2. FILTRO DE PUBLICACIÓN

Solo exportar archivos con frontmatter YAML que incluya:

- publish: true
- publish: "true"
- publish: "draft"

### 3. FRONTMATTER YAML DE OBSIDIAN

El frontmatter típico contiene valores como:

```yaml
---
created: 2025-11-02 20:57
area:
  - "[[Filosofia]]"
tags:
  - falacia
type: definición
aliases:
  - falacia del arenque rojo
  - Falacia de la pista falsa
  - Fal·làcia de l'arengada roja (català)
  - Red Herring Fallacy
  - Red Herring
comparte_tema_con:
  - "[[Falacias Informales]]"
  - "[[Falacias de Relevancia]]"
  - [[Lógica]]
relacionado_con:
  - "[[Falacia Ad Hominem]]"
  - "[[Non Sequitur]]"
---
```

### 4. TRATO DEL MARKDOWN

- No modificar el contenido.
- Separar frontmatter YAML (entre `---`) del resto del documento.
- Mantener el body intacto.
- Incluir una librería Markdown para posibles modificaciones en los ficheros Markdown.

### 5. ESTRUCTURA DE DIRECTORIOS

Implementar un comando `init` que inicialice la estructura inicial, que será:

```.
├── config.toml
├── content
├── plugins
├── sass
├── static
├── templates
└── themes
```

El fichero `config.toml` es el fichero de configuración de la aplicación.

Mantener rutas relativas.

Crear directorios si no existen.

### 6. INTERFAZ CLI

La interfaz con el usuario ha de ser moderna y amigable, como una aplicación moderna.

Parámetros:

-vault
-obsidian-vault-base
-vault
-osg-content-dir
-dry-run (no escribir, solo informar)
-verbose

### 7. ESTRUCTURA DEL TUI

## Header Panel (Colapsable con [H])
- Banner ANSI con título "OSG Builder"
- Badge de versión (ej: v0.1.0)
- Botón para ocultar/mostrar el header (también por combinación de teclas)
- Barra de información del sistema mostrando: OS, Agents activos, MCP servers y estado A2A (en esta fase no utilizaremos agentes)

## Panel Izquierdo - Workflow Steps
- Lista de los pasos del proceso con estados visuales:
  - ✓ Completados (verde)
  - ● En ejecución (naranja)
  - ○ Pendientes (gris atenuado)
  - ⊘ Deshabilitados (gris muy atenuado)
- información del agente o agentes que están ejecutando el paso.
- Panel de acciones con botones desplegables por server (EventManager, Registry Agentes A2A, MCP, etc.): START SERVER, STOP SERVER

## Panel Central - Agent Output
- Área principal de interacción con mensajes timestamped
- Etiquetas semánticas: [SYS], [AGENT:name], [QUESTION], [PROGRESS]
- Soporte para preguntas interactivas con opciones seleccionables
- Barra de progreso para tareas en curso
- Área de input con prompt (❯) y cursor
- Barra de comandos disponibles (/help, /mcp, /skills, /agents, /stop, /deploy)

## Panel Derecho - Info Panels
- [MCP:SERVERS] - Lista de servidores MCP con estado y consumo de tokens
- [AGENT:SKILLS] - Skills disponibles con estadísticas de tokens
- [A2A:AGENTS] - Agentes activos implementando A2A con estado y tokens
- [SHORTCUTS] - Atajos de teclado disponibles
- "Botón" para ocultar/mostrar el panel (también por combinación de teclas)

## Status Bar (inferior)
- Modelo activo (ej: claude-3.5-sonnet)
- Tokens consumidos (ej: 27.6k / 100k)
- Costo acumulado (ej: $0.0824)
- Agentes activos (ej: 3)
- Paso actual del workflow (ej: 2/5)

Este es una aproximación cuando introduzcamos el uso de agentes en OSG. Adapta la estructura a las necesidades de esta fase de la implementación.

### 8. SISTEMA DE PLUGINS

El core debe poder cargar y ejecutar binarios WASM que amplien las capacidades del programa.

Así, se podrá "ampliar funcionalidad" escribiendo scripts en Rust, TinyGo o Zig, compilandolos a WASM y ejecutarse de forma segura en el Core en Go.

Usar `Wazero`, que es un runtime de WASM escrito en Go puro (sin dependencias de C).

### 9. ERRORES Y LOGGING

- Usar logs estructurados.
- Logging configurable por CLI.
- Warnings cuando un archivo no se pueda parsear.

### 10. TESTS

Incluir tests para la funcionalidad básica.

### 11. OUTPUT FINAL

Debes generar:

- Código completo del proyecto en Rust.
- Documentación y/o guías explicativas de cómo compilar CLI y WASM.
- Ejemplos de uso.
- Estructura final recomendada.

Toda implementación debe ser idiomática, clara y modular.

# SEGUNDA FASE

El propósito del programa es construir la página web estática a partir de los ficheros obtenidos de un Vault de Obsidian.

Para continuar con el desarrollo, se deben implementar a continuación las siguientes tareas:

### Generación de HTML a partir de los ficheros Markdown originales

Los ficheros se encuentran en el directorio `content`. Hay que cambiar el nombre del parámetro `build` por `update-content`. Esto debe realizar las acciones actuales de obtener los ficheros, parsearlos y copiarlos a su ubicación final (dentro de `content`).

El parámetro `update-content` debe ser el parámetro por defecto.

### Generación de HTML a partir de los ficheros Markdown originales

Cuando pase a `osg` al opción `build`, debe generar la página web estática.

En este proceso deberá utilizar las plantillas para las páginas que se van a generar y que estarán en el directorio `templates`. Deberá tener en cuenta la información del frontmatter para generar la página web, por si el frontmatter indica que la página es una página de lista, una página de artículo, etc y si hay que aplicar algún template en particular.

Ejemplo:

```toml
+++
# Template to use to render this page.
template = "page.html"
...
+++
```

### Generación de Secciones

Se crea una sección siempre que un directorio (o subdirectorio) de la sección de contenido contenga un archivo _index.md. Si un directorio no contiene un archivo_index.md, no se creará ninguna sección, pero los archivos Markdown dentro de ese directorio crearán páginas (conocidas como páginas huérfanas).

La página de inicio (es decir, la página que se muestra cuando un usuario accede a su base_url) es una sección, que se crea independientemente de si agrega un archivo_index.md en la raíz de su directorio de contenido. Si no crea un archivo _index.md en su directorio de contenido, esta sección principal de contenido no tendrá contenido ni metadatos. Si desea agregar contenido o metadatos, puede agregar un archivo_index.md en la raíz del directorio de contenido y editarlo como cualquier otro archivo _index.md; su plantilla index.html tendrá acceso a ese contenido y metadatos.

Cualquier archivo que no sea Markdown en un directorio de sección se agrega a la colección de recursos de la sección. Estos archivos estarán luego disponibles en el archivo Markdown mediante enlaces relativos.

### Sass

Always compile Sass files in theme directories. However, to process files in the sass folder, you need to set `compile_sass = true` in config.toml.

Always process any files with the sass or scss extension in the sass folder, and place the processed output into a css file with the same folder structure and base name into the public folder:

.
└── sass
    ├── style.scss // -> ./public/style.css
    ├── indented_style.sass // -> ./public/indented_style.css
    ├── _include.scss # This file won't get put into the `public` folder, but other files can @import it.
    ├── assets
    │   ├── fancy.scss // -> ./public/assets/fancy.css
    │   ├── same_name.scss // -> ./public/assets/same_name.css
    │   ├── same_name.sass # CONFLICT! This has the same base name as the file above, so OSG will return an error.
    │   └── _common_mixins.scss # This file won't get put into the `public` folder, but other files can @import it.
    └── secret-side-project
        └── style.scss // -> ./public/secret-side-project/style.css

Files with a leading underscore in the name are not placed into the public folder, but can still be used as @import dependencies.

Files with the `scss` extension use "Sassy CSS" syntax, while files with the `sass` extension use the "indented" syntax: <https://sass-lang.com/documentation/syntax>. osg will return an error if scss and sass files with the same base name exist in the same folder to avoid confusion -- see the example above.

### Templates

Diseñar e implementar un sistema de plantillas (template engine) para mi programa que cubra los siguientes requisitos completos:

1. Estructura de directorios: un directorio templates en el que se almacenan todos los archivos de plantilla.

2. Variables globales accesibles en todas las plantillas, como:

- config (configuración global del programa, por ejemplo base_url, default_language, etc)

- current_path (ruta de la página actual, URL relativa, empezando con /)

- current_url (URL completa de la página actual)

- lang (idioma de la página actual)

- Para la plantilla 404, current_path y current_url pueden no estar disponibles.

3. Plantillas “estándar” por defecto, que se aplican si no se especifica lo contrario:

- index.html → para homepage

- section.html → para secciones (directorios dentro de content)

- page.html → para páginas individuales de contenido

4. Soporte para plantillas “built-in” adicionales como atom.xml, rss.xml (feeds), sitemap.xml, robots.txt, 404.html (o 404.html/404.tpl) y mecanismo de sobrescritura: si el usuario crea un archivo con el mismo nombre en templates, se usa ese archivo en lugar del incorporado.

5. Soporte para plantillas personalizadas: el usuario puede crear otros archivos .html en templates (en subdirectorios) y aplicarlos a páginas específicas mediante un campo en el front matter o metadato como template = "about.html".

6. En las plantillas debe haber filtros/utilidades básicas implementadas, al menos los siguientes:

- markdown → convierte una variable Markdown a HTML.

- base64_encode / base64_decode → codificar/decodificar Base64.

- regex_replace → reemplazo por expresión regular.

- num_format → formatear números según localización.

7. Soporte para funciones que pueden usarse dentro de las plantillas, tales como:

- get_page(path, lang?) → obtiene información de otra página dada su ruta.

- get_section(path, metadata_only?) → obtiene información de una sección.

- get_taxonomy_url(kind, name, lang?) → obtiene el permalink para un término de taxonomía.

- get_taxonomy(kind) → obtiene toda la taxonomía de un tipo.

- get_url(path, trailing_slash?, cachebust?) → obtiene URL/permalink para un recurso o página.

- get_hash(path, sha_type?, base64?) → obtiene hash de un archivo o cadena.

- get_image_metadata(path, allow_missing?) → obtiene metadatos de una imagen (ancho, formato, mime).

- load_data(path?|url?, format?, required?) → carga datos desde fichero local o URL (toml, json, csv, yml/xml, bibtex).

- trans(key, lang?) → traducción para un idioma.

8. Soporte para paginación: en una sección o término de taxonomía, se debe exponer un objeto paginator con los campos (paginate_by, base_url, number_pagers, first, last, previous, next, pages, current_index, total_pages).

9. Soporte para taxonomías: en la plantilla taxonomy_list (lista de todos los términos) y taxonomy_single (página de un término) deben estar disponibles variables como taxonomy, terms (o term), y las páginas asociadas.

10. Soporte para feeds: generación de atom.xml y/o rss.xml; plantilla con variables config, feed_url, last_updated, pages, lang, y para taxonomías también taxonomy, term.

11. Soporte para sitemap: plantilla sitemap.xml con variable entries: Array<SitemapEntry> (cada SitemapEntry tiene permalink, updated, extra). Si muchas páginas, mecanismo de archivo dividido con split_sitemap_index.xml y variable sitemaps.

12. Soporte para robots.txt: plantilla simple que tiene acceso a config y usa función get_url(), y permite incluir otros archivos.

13. Soporte para página de 404: plantilla especial que puede usar config, lang, pero no current_path ni current_url.

14. Soporte para página de archivo (“archive”): aunque no haya funcionalidad incorporada, permitir plantilla personalizada que recorra section.pages (o conjunto de páginas) y agrupe por año o fecha para mostrar un listado tipo “archivo”.

15. Sobrescritura de plantilla por tema o por usuario: si un tema proporciona plantillas, el usuario debe poder reemplazarlas creando archivos con el mismo nombre; el sistema debe resolver la ruta de carga con prioridad: usuario > tema > built-in.

16. Documentar cómo se hace la “resolución de rutas” de archivos (por ejemplo, para load_data, get_image_metadata, etc): la lógica debe explicar orden de búsqueda (program root, static dir, content dir, theme static dir, etc).

Por favor genera como salida:

- Un diagrama de alto nivel de cómo se organiza el sistema de plantillas (directorios, carga, contexto, renderizado).

- Una especificación técnica de los objetos de contexto que estarán disponibles en plantilla (qué campos tienen page, section, paginator, taxonomy, term, SitemapEntry, etc).

- Una lista de filtros y funciones que deben implementarse, con breve descripción.

- La implementación que muestre: cómo registrar filtros, cómo cargar una plantilla, cómo renderizarla con contexto, cómo permitir sobrescritura de plantillas y cómo manejar paginación/taxonomía/feeds/sitemap/robots/404/archive.

- Algunas consideraciones de extensibilidad (por ejemplo: permitir temas, permitir que el usuario añada filtros personalizados, caché de templates, soporte de múltiples idiomas, etc).

- Valorar si los filtros y funciones pueden ser implementados como plugins WASM. Si es factible, es la implementación preferida.

Nota: Ten en consideración la funcionalidad indicada como inspiración, pero adapta la terminología al contexto de OSG. Asegúrate de que la especificación incluya todas las funcionalidades mencionadas arriba.

### Taxonomies

#### Definiciones

- “Taxonomy”: una categoría definida por el usuario para agrupar contenido.
- “Term”: un grupo específico dentro de una taxonomía.
- “Value”: un contenido que se asocia a un término.

### Ejemplo conceptual

Por ejemplo, en un sitio de películas: taxonomías podrían ser _Director_, _Genres_, _Awards_, _Release year_. Un film es el “value”, cada director/género/award/año es un “term”.

Imagine again we have the following movies:

- Shape of water                             Value
  - Director ............................ Taxonomy
    - Guillermo Del Toro                      Term
  - Genres .............................. Taxonomy
    - Thriller                                Term
    - Drama                                   Term
  - Awards .............................. Taxonomy
    - Golden globe                            Term
    - Academy award                           Term
    - BAFTA                                   Term
  - Release year ........................ Taxonomy
    - 2017                                    Term

- The Room                                   Value
  - Director ............................ Taxonomy
    - Tommy Wiseau                            Term
  - Genres .............................. Taxonomy
    - Romance                                 Term
    - Drama                                   Term
  - Release Year ........................ Taxonomy
    - 2003                                    Term

- Bright                                     Value
  - Director ............................ Taxonomy
    - David Ayer                              Term
  - Genres .............................. Taxonomy
    - Fantasy                                 Term
    - Action                                  Term
  - Awards .............................. Taxonomy
    - California on Location Awards           Term
  - Release Year ........................ Taxonomy
    - 2017                                    Term

Entonces, la página de la taxonomía Release year mostraría enlaces a los términos 2003 y 2017; y cada término (como 2017) listaría todos los films que tienen ese año.

### Configuración en `config.toml`

Una taxonomía se define con estos campos principales:

- `name`: cadena obligatoria que se usará en las URLs (normalmente en plural: “tags”, “categories”, etc).
- `paginate_by`: número (opcional) que determina cuántos valores (“values”/páginas) por página de término si la página de término es paginada.
- `paginate_path`: opcional; si se define, controla la ruta de la paginación (por ejemplo “page/1”).
- `feed` (o `rss` en versiones antiguas): bool que indica si se generará un feed para cada término de la taxonomía.
- `render`: bool que indica si la taxonomía debería generar páginas automáticamente (si es `false`, no se generan las páginas de la taxonomía).

### Cómo se aplican las taxonomías al contenido

- En el front-matter de un contenido (por ejemplo `.md`), se puede especificar algo como:

  ```toml
  [taxonomies]
  tags = ["rust", "webdev"]
  categories = ["programming"]
  ```

  Esto hace que ese contenido valore los términos “rust”, “webdev” dentro de la taxonomía “tags”.
- osg recogerá esos valores y los agrupará para producir páginas de taxonomía (si `render=true`).

### Plantillas para taxonomías

- En el directorio `templates`, osg busca por cada taxonomía:

  - `$TAXONOMY_NAME/list.html` (lista de todos los términos de esa taxonomía)
  - `$TAXONOMY_NAME/single.html` (una página para un término)
- Si esos archivos específicos no existen, osg usará plantillas genéricas: `taxonomy_list.html` y `taxonomy_single.html`.

### Variables disponibles en las plantillas de taxonomía

#### Para `list.html` (lista de términos)

Las variables disponibles son:

- `config: Config` → la configuración del sitio.
- `taxonomy: TaxonomyConfig` → datos de la taxonomía que se está procesando.
- `current_url: String` → URL completa actual.
- `current_path: String` → ruta relativa actual.
- `terms: Array<TaxonomyTerm>` → todos los términos de esta taxonomía.
- `lang: String` → idioma de la página actual.

#### Para `single.html` (una página de término)

Variables:

- `config: Config`
- `taxonomy: TaxonomyConfig`
- `current_url: String`
- `current_path: String`
- `term: TaxonomyTerm` → el término actual que se está renderizando.
- `lang: String`
- Si la página del término está paginada, también estará disponible `paginator` (ver la sección de paginación).

#### Estructura de `TaxonomyTerm` y `TaxonomyConfig`

- `TaxonomyTerm` tiene los campos:

  - `name: String`
  - `slug: String`
  - `path: String`
  - `permalink: String`
  - `pages: Array<Page>` → los valores (contenidos) asociados al término.
  - `page_count: Number` → número de páginas asociadas.

- `TaxonomyConfig` tiene los campos:

  - `name: String`
  - `paginate_by: Number?`
  - `paginate_path: String?`
  - `feed: Bool` (o `rss`)
  - `render: Bool`

### Comportamiento importante

- Las páginas de términos individuales pueden estar paginadas si `paginate_by` se definió. Cuando haya paginación, la plantilla de término obtiene un objeto `paginator`.
- Si `render=false` en la configuración de la taxonomía, no se generan páginas para esa taxonomía.

### Implementación

 Actúa como un arquitecto de sistemas para generadores de sitios estáticos o programas de contenido.
 Los requisitos son los siguientes:

 1. Permitir al usuario definir en la configuración global una lista de taxonomías, cada una con los siguientes campos:

    - `name`: cadena (obligatoria) que define la URL básica de la taxonomía (por ejemplo “tags”, “categories”).
    - `paginate_by`: número opcional que indica cuántos contenidos (valores) se muestran por página en el listado de un término.
    - `paginate_path`: cadena opcional que indica la ruta de paginación (por ejemplo “page/”).
    - `feed`: booleano que indica si se debe generar un feed por cada término.
    - `render`: booleano que indica si se deben generar páginas de taxonomía (lista de términos y página por término).

 2. En cada contenido (por ejemplo página Markdown), permitir en el front matter/metadatos asociar uno o más términos de una o más taxonomías.

 3. Durante la generación/compilación:

    - Para cada taxonomía con `render=true`, generar una página de lista: `/taxonomía_name/` que muestra todos los términos.
    - Para cada término dentro de esa taxonomía, generar una página (o páginas si paginada) en `/taxonomía_name/term_slug/` que lista todos los contenidos asociados.
    - Si `paginate_by` está definido, dividir la lista de contenidos del término en varias páginas y proveer enlaces de paginación (`first`, `next`, etc).
    - Si `feed=true`, generar un feed (por ejemplo Atom o RSS) que incluye los contenidos de ese término.

 4. En el sistema de plantillas de mi programa (por ejemplo usando un motor tipo Tera/Jinja), definir un mecanismo para dos plantillas de taxonomía:

    - `TAXONOMY_NAME/list.html` → para la lista de términos.
    - `TAXONOMY_NAME/single.html` → para una página de término.
      Si no existen plantillas específicas para esa taxonomía, usar plantillas genéricas: `taxonomy_list.html` y `taxonomy_single.html`.

 5. En el contexto de renderización de las plantillas, las variables disponibles deben incluir:

    - `config`: configuración global del sitio.
    - `taxonomy`: objeto que representa la configuración de la taxonomía (`TaxonomyConfig`).
    - En `list.html`: `terms: Array<TaxonomyTerm>`, `current_url`, `current_path`, `lang`.
    - En `single.html`: `term: TaxonomyTerm`, además de `config`, `taxonomy`, `current_url`, `current_path`, `lang`. Si hay paginación, además `paginator`.
    - Estructura de `TaxonomyTerm`: `name`, `slug`, `path`, `permalink`, `pages`, `page_count`.
    - Estructura de `TaxonomyConfig`: `name`, `paginate_by?`, `paginate_path?`, `feed`, `render`.

 6. Documentar cómo se integrará en el flujo de compilación: lectura de front matter, agregación de valores para cada término, generación de páginas, aplicación de plantillas, paginación, feed si aplica.

 7. Proveer un esqueleto de implementación en el lenguaje de tu elección (por ejemplo Python o JavaScript) que muestre:

    - Registro de taxonomías desde config.
    - Lectura de los valores de taxonomía de los contenidos.
    - Construcción de los objetos `TaxonomyTerm` con la lista de páginas asociadas.
    - Renderizado de la plantilla `list.html` para cada taxonomía.
    - Renderizado de la plantilla `single.html` para cada término (incluyendo paginación).
    - Generación de feed por término si aplica.

 8. Consideraciones de extensibilidad: permitir al usuario añadir filtros o funciones de plantilla específicas para taxonomía; permitir temas que sobrescriban plantillas de taxonomía; manejo de múltiples idiomas (i18n) si el sitio es multilingüe; permitir personalización de URL para taxonomías (por ejemplo cambiar `/tags/` a `/topics/`).

 Por favor genera como salida:

- Una **especificación técnica** detallada del sistema de taxonomías (objetos, variables, rutas, plantillas).
- Un **diagrama de flujo** o de arquitectura de cómo se genera: config → contenido → agregación taxonomías → páginas → plantillas.
- Un **esqueleto de código** (en Python o JavaScript) que muestre los pasos clave.
- Algunas **mejores prácticas** y **limitaciones** a tener en cuenta (por ejemplo: muchos términos pueden afectar el rendimiento, planificación de paginación, impacto SEO, etc).
- Un directorio con un documento markdown, numerado ({US_number}_{US_TITLE}.md, ej: 03_Leer-frontmatter.md).
- Un roadmap o plan de implementación por fases.


-------------------------------------------------------------------------------
STACK TECNOLÓGICO
-------------------------------------------------------------------------------

| Componente | Tecnologia | Justificacion |
|------------|------------|---------------|
| **Lenguaje** | Go 1.25+ | Binario unico, rendimiento, ecosistema CLI |
| **CLI Framework** | [Kong](https://github.com/alecthomas/kong) | Declarativo, type-safe, extensible |
| **TUI** | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | TUI profesional, componentes Charm |
| **Configuracion** | [koanf](https://github.com/knadh/koanf) | Multi-source, flexible, type-safe |
| **Framework Agentes** | [Kairos](https://github.com/jllopis/kairos) | MCP, A2A, Planner, observabilidad, multi-LLM |
| **LLM** | Kairos providers | Multi-provider (Anthropic, OpenAI, Gemini, Qwen, Ollama) |
| **Templates** | Go text/template | Nativo, potente, familiar |
| **Testing** | Go testing + testify | Tooling nativo Go |
| **Orquestacion** | Kairos | Orquestacion de flujos de trabajo, gestion de eventos |
