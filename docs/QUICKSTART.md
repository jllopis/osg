# Quickstart

## 1) Crear un sitio
```bash
osg init
```

## 2) Importar contenido desde Obsidian
```bash
osg update-content --vault-path /ruta/a/mi-vault
```

## 3) Build HTML
```bash
osg build
```

## 4) Previsualizar
```bash
osg serve
```

## 5) TUI
```bash
osg tui
```

Por defecto, `osg` sin comando lanza la TUI.

## Ejemplo rapido
En `examples/sample-site/` hay un sitio minimo con `config.yaml` y un contenido de ejemplo.
```bash
cd examples/sample-site
osg build
osg serve
```
