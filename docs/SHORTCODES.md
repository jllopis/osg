# Shortcodes

Los shortcodes permiten insertar contenido enriquecido directamente en tus notas
Markdown de Obsidian. Se expanden **antes** de la conversion Markdown (antes de
Goldmark), por lo que puedes mezclarlos libremente con sintaxis Markdown normal.

## Sintaxis

Hay dos tipos de shortcodes:

**Block** (pares, con contenido entre apertura y cierre):

```
{{< nombre [argumentos] >}}
contenido Markdown...
{{< /nombre >}}
```

**Inline** (auto-cerrados, sin contenido):

```
{{< nombre argumentos />}}
```

### Argumentos

Los shortcodes aceptan argumentos en varios formatos:

| Formato | Ejemplo |
|---------|---------|
| `key="value"` | `src="foto.jpg"` |
| `key='value'` | `caption='Mi foto'` |
| `key=value` | `height=400` |
| Posicional (bare) | `"Titulo del bloque"` |

Cuando solo hay un argumento posicional (sin `key=`), se interpreta segun el
shortcode. Por ejemplo, en `note` es el titulo; en `youtube` es el ID del video.

---

## Admoniciones: note, warning, tip

Bloques destacados con colores de la paleta Nord. Utiles para llamar la atencion
sobre informacion importante.

| Shortcode | Color | Titulo por defecto |
|-----------|-------|--------------------|
| `note` | Azul (Nord #5e81ac) | info |
| `warning` | Naranja (Nord #d08770) | warning |
| `tip` | Verde (Nord #a3be8c) | tip |

### Uso basico

```markdown
{{< note >}}
Este es un aviso informativo.
{{< /note >}}
```

### Con titulo personalizado

```markdown
{{< note "Importante" >}}
Recuerda configurar tu `vault_path` en `config.yaml`.
{{< /note >}}

{{< warning "Cuidado" >}}
Esta operacion borra el directorio `public/`.
{{< /warning >}}

{{< tip "Consejo" >}}
Usa `osg serve --watch` para recargar automaticamente.
{{< /tip >}}
```

El contenido interior se procesa como Markdown, asi que puedes usar **negritas**,
`codigo`, listas, etc.

---

## Details (bloque colapsable)

Genera un elemento HTML `<details>/<summary>` nativo. El bloque aparece
colapsado por defecto y el usuario puede expandirlo haciendo clic.

### Uso basico

```markdown
{{< details >}}
Contenido oculto que se muestra al hacer clic.
{{< /details >}}
```

### Con titulo personalizado

```markdown
{{< details "Ver codigo fuente" >}}
```go
func main() {
    fmt.Println("hello")
}
```
{{< /details >}}
```

Sin titulo, el texto del `<summary>` es "Details".

---

## Quote (cita con atribucion)

Genera un `<blockquote>` estilizado con atribucion opcional (autor y fuente).
El contenido interior se procesa como Markdown.

### Argumentos

| Argumento | Descripcion | Obligatorio |
|-----------|-------------|-------------|
| `author` | Nombre del autor (posicional o `author="..."`) | No |
| `source` | Titulo de la obra o fuente | No |

### Ejemplos

Sin atribucion:

```markdown
{{< quote >}}
La imaginacion es mas importante que el conocimiento.
{{< /quote >}}
```

Con autor (posicional):

```markdown
{{< quote "Albert Einstein" >}}
La imaginacion es mas importante que el conocimiento.
{{< /quote >}}
```

Con autor y fuente:

```markdown
{{< quote author="Albert Einstein" source="Sobre la ciencia cosmica" >}}
La imaginacion es mas importante que el conocimiento.
{{< /quote >}}
```

Solo fuente, sin autor:

```markdown
{{< quote source="Refranero popular" >}}
Mas vale pajaro en mano que cien volando.
{{< /quote >}}
```

---

## Figure (imagen con caption)

Genera un elemento `<figure>` con imagen, caption opcional, atributos de clase,
ancho y enlace. Es una alternativa mas rica que la sintaxis `![alt](src)` de
Markdown.

### Argumentos

| Argumento | Descripcion | Obligatorio |
|-----------|-------------|-------------|
| `src` | Ruta de la imagen | Si |
| `caption` | Texto del `<figcaption>` | No |
| `alt` | Texto alt de la imagen (si no se da, usa `caption`) | No |
| `class` | Clase CSS adicional en el `<figure>` | No |
| `width` | Ancho de la imagen (px o %) | No |
| `link` | URL de enlace alrededor de la imagen | No |

### Ejemplos

```markdown
{{< figure src="/img/paisaje.jpg" caption="Atardecer en la costa" alt="Foto de un atardecer" >}}
{{< /figure >}}
```

Con ancho y enlace:

```markdown
{{< figure src="/img/diagrama.png" width="600" link="https://example.com" class="centered" >}}
{{< /figure >}}
```

Forma corta (solo ruta como argumento posicional):

```markdown
{{< figure "/img/foto.jpg" >}}
{{< /figure >}}
```

Si pones contenido Markdown entre las etiquetas, se usa como `<figcaption>` en
lugar de `caption`:

```markdown
{{< figure src="/img/foto.jpg" >}}
**Foto tomada** en enero de 2025.
{{< /figure >}}
```

---

## Tabs (pestanas)

Sistema de pestanas con cambio por JavaScript, navegacion por teclado y
accesibilidad (ARIA). Requiere un contenedor `tabs` con uno o mas bloques `tab`
dentro.

### Uso

```markdown
{{< tabs >}}

{{< tab "JavaScript" >}}
```js
console.log("hello");
```
{{< /tab >}}

{{< tab "Python" >}}
```python
print("hello")
```
{{< /tab >}}

{{< tab "Go" >}}
```go
fmt.Println("hello")
```
{{< /tab >}}

{{< /tabs >}}
```

La primera pestana se muestra activa por defecto. El argumento de `tab` es el
titulo que aparece en la pestana (por defecto "Tab").

Las teclas `←` y `→` navegan entre pestanas. El script `tabs.js` se carga
automaticamente y no tiene dependencias.

---

## YouTube

Inserta un video de YouTube con embed responsive 16:9. Usa
`youtube-nocookie.com` para respetar la privacidad del visitante.

### Formatos aceptados

```markdown
{{< youtube "dQw4w9WgXcQ" />}}

{{< youtube "https://www.youtube.com/watch?v=dQw4w9WgXcQ" />}}

{{< youtube "https://youtu.be/dQw4w9WgXcQ" />}}
```

Los tres formatos producen el mismo resultado. El shortcode extrae
automaticamente el ID del video de URLs completas.

---

## Twitter / X

Inserta un tweet embebido via oEmbed con el widget oficial de Twitter.
Las URLs de `x.com` se normalizan automaticamente a `twitter.com` para
compatibilidad con el sistema de embeds.

### Uso

```markdown
{{< twitter "https://twitter.com/usuario/status/1234567890" />}}

{{< twitter "https://x.com/usuario/status/1234567890" />}}
```

Ambas formas funcionan identicamente.

---

## CodePen

Inserta un embed de CodePen como iframe. Parsea automaticamente la URL para
extraer el usuario y el ID del pen.

### Argumentos

| Argumento | Descripcion | Default |
|-----------|-------------|---------|
| URL (posicional o `url=`) | URL del pen en CodePen | (obligatorio) |
| `height` | Altura del iframe en px | `400` |
| `theme` | Tema del embed (`dark` / `light`) | `dark` |
| `tab` | Pestana activa (`result`, `html`, `css`, `js`) | `result` |

### Ejemplos

```markdown
{{< codepen "https://codepen.io/usuario/pen/AbCdEf" />}}
```

Con argumentos personalizados:

```markdown
{{< codepen url="https://codepen.io/usuario/pen/AbCdEf" height=600 theme=light tab=css />}}
```

Si la URL no es valida, se genera un enlace de fallback en lugar del iframe.

---

## Referencia rapida

| Shortcode | Tipo | Ejemplo minimo |
|-----------|------|----------------|
| `note` | Block | `{{< note >}}Texto{{< /note >}}` |
| `warning` | Block | `{{< warning >}}Texto{{< /warning >}}` |
| `tip` | Block | `{{< tip >}}Texto{{< /tip >}}` |
| `details` | Block | `{{< details >}}Texto{{< /details >}}` |
| `quote` | Block | `{{< quote "Autor" >}}Texto{{< /quote >}}` |
| `figure` | Block | `{{< figure src="img.jpg" >}}{{< /figure >}}` |
| `tabs`+`tab` | Block | `{{< tabs >}}{{< tab "T" >}}...{{< /tab >}}{{< /tabs >}}` |
| `youtube` | Inline | `{{< youtube "ID" />}}` |
| `twitter` | Inline | `{{< twitter "URL" />}}` |
| `codepen` | Inline | `{{< codepen "URL" />}}` |

## Notas tecnicas

- Los shortcodes se expanden **antes** de Goldmark, en la fase
  `ExpandShortcodes()` del pipeline Markdown.
- El contenido interior de los block shortcodes se procesa como Markdown
  (pasa por Goldmark despues de la expansion).
- Los shortcodes desconocidos se dejan intactos en el output.
- El estilo visual esta definido en `style.css` del tema default (secciones
  ADMONITIONS, DETAILS, QUOTE, FIGURE, EMBEDS, TABS).
