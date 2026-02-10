# Release Notes

## v0.2.0 - 2026-02-10

### Novedades Brillantes
- **TUI por defecto:** Al ejecutar `osg` se abre la interfaz visual con acciones rápidas y estado en tiempo real.
- **Plugins WASM listos para usar:** CLI/TUI para instalar, habilitar y crear plugins, con ejemplos de RSS y búsqueda.
- **Quickstart y sample site:** Un sitio de ejemplo y guía rápida para arrancar sin depender de un vault real.

### Ajustes y Mejoras
- **Build incremental con cache:** Evita renders innecesarios y acelera iteraciones.
- **Live reload y watch:** Previsualización fluida con rebuild automático.
- **Theme starter kit:** Comando para generar un tema base y empezar a personalizar rápido.
- **TUI más clara y ergonómica:** Paneles, atajos y badges de estado para seguimiento visual.

### Bichos Aplastados
- **Plugins Rust:** Correcciones en targets WASM y strings crudas en el ejemplo de búsqueda.
- **CLI theme init:** Mejora en el manejo de comandos anidados.

### Bajo el Capó
- **`osg doctor`:** Validación de config y entorno con warnings útiles.
- **Limpieza de `public/`:** Evita archivos stale cuando se eliminan contenidos.
