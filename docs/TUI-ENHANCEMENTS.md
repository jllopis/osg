# TUI Enhancements: Server Management + Config Editor

Phase 17 del roadmap de OSG. Dos mejoras principales al TUI:

1. **Gestion de servicios**: Lanzar/parar serve y API desde el TUI con panel de logs dedicado.
2. **Editor de configuracion**: Pantalla modal completa para consultar y editar config.yaml.

## Decisiones de diseno

| Aspecto | Decision |
|---------|----------|
| Serve/API | 3 modos: `/serve` (static), `/serve --api` (embebido), `/api` (standalone) |
| Logs | Panel inferior dedicado con toggle (F7), tabs serve/API/all |
| Config | Pantalla modal completa, edicion inline con formularios |
| Persistencia config | Guardado explicito (Ctrl+S) con indicador de cambios pendientes |
| Controles rapidos | Slash commands + teclas dedicadas (F5/F6/F7/F8) |
| YAML comments | Preservados via `yaml.Node` round-trip (gopkg.in/yaml.v3) |
| Log panel modifier | Configurable via `tui_log_modifier` ("shift" por defecto, "alt" opt-in). Shift funciona en todos los terminales; alt requiere configurar Option como Meta en macOS |

## Fase A: Sistema de logs multi-canal

### Problema

Actualmente hay un solo `LogSink` -> un canal -> viewport principal. Los logs de
serve, API y acciones generales (build, update) se mezclan. Para tener tabs
serve/API necesitamos canales separados con etiqueta de origen.

### Cambios

**`internal/tui/logsink.go`** — Anadir campo `source string` al constructor.
Cada linea emitida incluye la fuente. El mensaje pasa de `logLineMsg string` a
`taggedLogLineMsg{source, line}`.

```go
type taggedLogLineMsg struct {
    source string // "general", "serve", "api"
    line   string
}
```

**`internal/app/tui.go`** — Crear 3 LogSinks en `RunTUI()`:
- `generalSink` — para build, update-content, doctor, init
- `serveSink` — para RunServe (static y con API embebida)
- `apiSink` — para RunAPI standalone

Cada Action closure recibe el sink apropiado.

**`internal/tui/model.go`** — El Model almacena mensajes separados:
- `messages []Message` — general (viewport principal)
- `serveMessages []Message` — logs del dev server
- `apiMessages []Message` — logs de la API standalone
- 3 canales de log (`generalCh`, `serveCh`, `apiCh`)

**Ficheros**: `logsink.go`, `model.go`, `update.go`, `app/tui.go`
**Tests**: `logsink_test.go` actualizado, tests multi-canal

## Fase B: Gestion de serve + API

### Nuevos comandos y teclas

| Slash command | Tecla | Accion |
|---------------|-------|--------|
| `/serve` | F5 | Toggle serve (solo static files) |
| `/serve --api` | — | Lanza serve con API embebida |
| `/api` | F6 | Toggle API standalone |
| `/stop serve` | — | Para el dev server |
| `/stop api` | — | Para la API standalone |
| `/logs` | F7 | Toggle panel de logs |
| `/config` | F8 | Abre editor de config |

### Cambios al Model

Nuevos campos en `Model`:
```go
apiRunning   bool
apiCancel    context.CancelFunc
serveMode    string // "" | "static" | "api" (serve+api embebida)
```

Nuevas Actions:
```go
ServeWithAPI func(ctx context.Context) error
RunAPI       func(ctx context.Context) error
```

### Cambios a update.go

- `toggleServe()` acepta modo (static/api)
- Nuevo `toggleAPI()` para API standalone
- `finishTask` maneja `taskServe` y `taskAPI`
- F5/F6 delegados a los toggles

### Sidebar

Seccion "Services" reemplaza la actual "Serve":
```
── Services ──────────────
serve   ● running :1313
          mode: static+api
api     ○ stopped
```

### Header

Badges separados con colores distintos:
```
OSG · My Blog          [SERVE :1313] [API :8090] ⠋
```

**Ficheros**: `model.go`, `commands.go`, `keys.go`, `update.go`, `sidebar.go`,
`header.go`, `app/tui.go`
**Tests**: Nuevos comandos, toggle API, estados combinados

## Fase C: Panel de logs

### Componente LogPanel

Nuevo fichero `internal/tui/logpanel.go`:

- Viewport propio con scroll independiente
- Tabs: `[Serve]` `[API]` `[All]`
- Tab seleccionable con `1`/`2`/`3` o shift+left/right
- Altura: 1/3 de la terminal
- Borde superior con titulo, tab activo, indicador de scroll

### Layout

Sin log panel:
```
┌─────────────────────────────┐
│ header                      │
├──────────────────┬──────────┤
│ viewport         │ sidebar  │
│                  │          │
├──────────────────┴──────────┤
│ > input                     │
│ hints                       │
└─────────────────────────────┘
```

Con log panel:
```
┌─────────────────────────────┐
│ header                      │
├──────────────────┬──────────┤
│ viewport         │ sidebar  │
│                  │          │
├──────────────────┴──────────┤
│ ── Logs [Serve] API  All ── │
│ 14:32:01 GET /index.html    │
│ 14:32:02 200 OK 12ms        │
├─────────────────────────────┤
│ > input                     │
│ hints                       │
└─────────────────────────────┘
```

`recalcLayout()` redistribuye alturas. El viewport principal se reduce para
acomodar el panel.

### Foco

- up/down: scroll viewport principal (por defecto)
- Shift+up/down: scroll log panel (cuando visible)
- O bien Tab cambia foco entre viewport y log panel

**Ficheros**: `logpanel.go` (nuevo), `model.go`, `view.go`, `update.go`, `styles.go`
**Tests**: `logpanel_test.go`

## Fase D: Config — Infraestructura

### Schema registry

Nuevo fichero `internal/config/schema.go`:

```go
type FieldType int
const (
    FieldString FieldType = iota
    FieldBool
    FieldInt
    FieldStringList    // []string
    FieldIntList       // []int
    FieldStringMap     // map[string]string
    FieldDropdown      // string con opciones limitadas
    FieldStructList    // []struct (taxonomies, providers, languages)
)

type ConfigField struct {
    Key         string     // YAML path: "ai.provider"
    Label       string     // Display: "AI Provider"
    Description string     // Help text
    Type        FieldType
    Options     []string   // Para FieldDropdown
    Default     any        // Valor por defecto
}

type ConfigSection struct {
    Name        string
    Description string
    Fields      []ConfigField
}

func ConfigSchema() []ConfigSection
```

`ConfigSchema()` devuelve ~15 secciones con todos los campos, tipos,
descripciones y defaults. Es la fuente de verdad para el editor TUI.

Secciones:
1. Site Identity (base_url, site_title, site_description, default_language)
2. Theme & Appearance (theme, color_scheme, logo, favicon, nav_taxonomy)
3. Content (vault_path, content_dir, content_layout, include_drafts)
4. Output (public_dir, clean_public, minify)
5. Summaries (summary_strategy)
6. AI (ai.provider, ai.model, ai.api_key, ai.base_url, ai.system_prompt, ai.timeout, ai.concurrency)
7. Feeds (site_feed, site_feed_limit)
8. Templates & Static (templates_dir, static_dir)
9. Plugins (plugins_dir, plugins_enabled, plugin_timeout)
10. Sass (sass_dir, compile_sass)
11. Images (image_optimization, image_quality, image_widths, lightbox)
12. Sharing & Social (sharing, social)
13. Dev Server (serve_watch, serve_live_reload, serve_debounce_ms)
14. Logging (logging.level, logging.format)
15. Taxonomies (taxonomies[])
16. Interactions (interactions.enabled, api_url, listen, db_path, cors_origins, view_dedup_hours)
17. Comments (interactions.comments.enabled, db_path, auth_session_days, auth_callback_url, providers[])
18. Deploy (deploy.provider, deploy.cloudflare, deploy.rsync, deploy.s3)

### YAML Node helpers

Nuevo fichero `internal/config/yamlnode.go`:

```go
func LoadNode(path string) (*yaml.Node, error)
func GetNodeValue(root *yaml.Node, keyPath string) (*yaml.Node, bool)
func SetNodeValue(root *yaml.Node, keyPath string, value any) error
func DeleteNodeKey(root *yaml.Node, keyPath string) error
func SaveNode(path string, root *yaml.Node) error
```

Estos helpers usan `yaml.Node` de `gopkg.in/yaml.v3` para editar config.yaml
preservando comentarios, orden de claves y formato. No se necesitan dependencias
nuevas (ya es dependencia directa).

### Refactor de UpdatePluginsEnabled

El actual `UpdatePluginsEnabled()` en `update.go` destruye comentarios al hacer
round-trip por `map[string]any`. Se refactoriza para usar los helpers de
`yamlnode.go`. Esto arregla un bug existente.

**Ficheros**: `schema.go` (nuevo), `yamlnode.go` (nuevo), `update.go`
**Tests**: Schema cobertura completa, yamlnode round-trip con comentarios

## Fase E: Config — Pantalla modal

### Layout

```
┌─────────────────────────────────────────────────────────┐
│ Configuration                              [● modified] │
├────────────────┬────────────────────────────────────────┤
│ ▸ Site Identity│ Site Identity                          │
│   Theme        │                                        │
│   Content      │ base_url         https://mysite.com    │
│   Output       │   Base URL for the site. Used for      │
│   Summaries    │   absolute URLs, feeds, sitemap.       │
│   AI           │                                        │
│   Feeds        │ site_title       My Blog               │
│   Templates    │   Title shown in header and <title>.   │
│   Plugins      │                                        │
│   Sass         │ site_description A personal blog       │
│   Images       │   Short description for meta tags.     │
│   Sharing      │                                        │
│   Dev Server   │ default_language es                    │
│   Logging      │   ISO 639-1 code. Default language     │
│   Taxonomies   │   for content and UI.                  │
│   Interactions  │                                        │
│   Comments     │                                        │
│   Deploy       │                                        │
├────────────────┴────────────────────────────────────────┤
│ Ctrl+S save │ Esc back │ Enter edit │ Tab panel │ ↑↓ nav│
└─────────────────────────────────────────────────────────┘
```

### Componentes de edicion por tipo

| Tipo | Interaccion |
|------|------------|
| String | Enter abre input inline, Enter confirma, Esc cancela |
| Bool | Space toggle |
| Int | Enter abre input con validacion numerica |
| Dropdown | Enter abre lista, up/down selecciona, Enter confirma |
| StringList | Lista con items. `a` anade, `d` borra, Enter edita |
| IntList | Igual que StringList con validacion numerica |
| StringMap | Lista key=value. `a` anade par, `d` borra, Enter edita |
| StructList | Lista expandible. Enter expande/colapsa. Campos inline |

### Dirty state

- `dirtyFields map[string]bool` — campos modificados
- Indicador `[● modified]` en el header de la pantalla
- Ctrl+S: escribe al config.yaml via `yamlnode.SaveNode()`, limpia dirty state
- Esc con dirty state: dialogo "Unsaved changes. Save? (y/n/Esc)"

### Model

```go
type ConfigScreenModel struct {
    sections      []config.ConfigSection
    sectionIdx    int        // seccion seleccionada
    fieldIdx      int        // campo seleccionado dentro de la seccion
    focusPanel    string     // "sections" | "fields"
    editing       bool       // campo en modo edicion
    values        map[string]any    // valores actuales (cargados de config)
    dirtyFields   map[string]bool   // campos modificados
    configNode    *yaml.Node        // arbol YAML para guardar
    configPath    string
    // sub-components
    textInput     textinput.Model
    listEditor    *ListEditorModel  // para StringList/IntList
    mapEditor     *MapEditorModel   // para StringMap
    structEditor  *StructEditorModel // para StructList
}
```

**Ficheros**: `configscreen.go` (nuevo), `configfields.go` (nuevo), `model.go`,
`commands.go`, `update.go`, `view.go`
**Tests**: Navegacion, edicion por tipo, dirty state, save/cancel

## Fase F: Integracion y pulido

1. **Status bar contextual**: hints cambian segun modo (normal/config/log panel)
2. **Config reload**: al guardar desde el editor, recargar config para que el
   sidebar refleje los nuevos valores
3. **Docs**: DESIGN.md seccion TUI, AGENTS.md modulos nuevos
4. **ROADMAP/TASKS**: marcar como done

## Orden de ejecucion

| Fase | Dependencias | Tamano |
|------|-------------|--------|
| A — Multi-canal logs | ninguna | mediano |
| B — Serve + API mgmt | A | mediano |
| C — Log panel | A, B | mediano |
| D — Config infra | ninguna (paralela a A-C) | mediano |
| E — Config modal | D | grande |
| F — Integracion | A-E | pequeno |

Las fases A-C (servicios) y D (config infra) son independientes.
