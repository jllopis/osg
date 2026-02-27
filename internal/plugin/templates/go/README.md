# {{name}} (WASM plugin)

OSG plugin written in Go, compiled with TinyGo to WASM.

## Prerequisites

- [TinyGo](https://tinygo.org/getting-started/install/) 0.35+

## Build

```bash
chmod +x build.sh
./build.sh
```

## Install

Copy the compiled `.wasm` file to your site's `plugins/` directory:

```bash
cp {{name}}.wasm <your-site>/plugins/
```

Then enable it in `config.yaml`:

```yaml
plugins_enabled:
  - {{name}}
```

## Available Hooks

| Hook | Description |
|------|-------------|
| `config.validate` | Validate config before build (errors abort) |
| `build.started` | Build pipeline starting |
| `content.transform` | Modify Markdown before rendering |
| `page.render` | Override page template context |
| `section.render` | Override section template context |
| `taxonomy.list.render` | Override taxonomy list context |
| `taxonomy.term.render` | Override taxonomy term context |
| `image.process` | Process images via WASI filesystem |
| `build.finished` | All rendering complete |
| `after.build` | Post-build (deploy, notifications) |
