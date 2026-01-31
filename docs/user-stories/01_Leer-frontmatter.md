# 01_Leer-frontmatter

## Historia
Como autor, quiero que OSG lea el frontmatter YAML de mis notas y mantenga el body intacto, para poder publicar sin perder informacion.

## Criterios de aceptacion
- El parser detecta correctamente el bloque frontmatter entre `---`.
- El body se conserva sin cambios (byte a byte) tras la separacion.
- Si no hay frontmatter, el body se conserva y se usa frontmatter vacio.
- Errores de parseo generan warning y el archivo se omite (no hard fail).

## Notas
- Prioridad de campos para fecha y slug se define en REQ/Design.
