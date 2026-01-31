# REQUIREMENTS - OSG

## Goal
Construir un generador de contenido estatico desde un vault de Obsidian, con un pipeline claro para copiar y normalizar contenido en `content/` y, en fases posteriores, renderizar a `public/`.

## Scope
In-scope (MVP):
- Lectura de vault y archivos Markdown
- Parse YAML frontmatter
- Filtro publish (true, "true", "draft")
- Normalizacion de frontmatter de salida (YAML)
- Copia a `content/{YYYY/MM/DD}/{slug}/index.md`
- CLI moderna (init, update-content default)
- Logging estructurado
- Tests basicos

Out-of-scope (MVP):
- Render HTML completo
- Taxonomias, feeds, sitemap
- Plugins WASM
- TUI avanzada

## User stories (con aceptacion)
US01 - Leer frontmatter
- Como autor, quiero que OSG lea frontmatter YAML y preserve el body sin cambios
- Aceptacion: el body no se modifica y el frontmatter se parsea sin perder campos

US02 - Filtrar por publish
- Como editor, quiero exportar solo notas con publish=true/"true"/"draft"
- Aceptacion: notas sin publish no se exportan

US03 - Respetar drafts
- Como editor, quiero excluir drafts por defecto y poder incluirlos con flag
- Aceptacion: publish="draft" solo se copia si se pasa --include-drafts

US04 - Normalizar metadatos
- Como usuario, quiero un frontmatter de salida consistente
- Aceptacion: los campos minimos aparecen en salida y los originales quedan bajo `obsidian`

US05 - Estructura de directorios
- Como operador, quiero una estructura inicial con `osg init`
- Aceptacion: se crean directorios y config base sin errores

US06 - Observabilidad
- Como operador, quiero logs estructurados y modo verbose
- Aceptacion: logs incluyen nivel, evento y ruta de archivo

## Functional requirements
- Leer vault dado `--vault-path` o `--obsidian-vault-base` + `--vault`.
- Parsear frontmatter YAML entre `---`.
- Filtro publish: true, "true", "draft".
- Convertir frontmatter a esquema de salida definido.
- Copiar a `content/{YYYY/MM/DD}/{slug}/index.md`.
- Soportar `--dry-run`.
- Soportar `--include-drafts`.
- Soportar config file (propuesto: YAML).

## Non-functional requirements
- Go 1.25.x.
- Rendimiento aceptable en vault medianos.
- Errores de parseo deben registrar warning y continuar.
- Pruebas unitarias para parsing y mapping basicos.

## Constraints / Dependencies
- Go modules.
- Librerias: kong (CLI), koanf (config), log estructurado (slog o zap), yaml parser.
- Sin dependencia C (wazero cuando llegue fase WASM).
