# ADR-001: unsafe.Pointer en el SDK WASM de plugins

**Estado:** Aceptado
**Fecha:** 2025-07-15
**Contexto:** internal/plugin/sdk/sdk.go

## Contexto

El SDK Go para plugins WASM (`internal/plugin/sdk/`) necesita leer y
escribir datos en la memoria lineal de WebAssembly. Go no permite crear
slices a partir de punteros arbitrarios sin usar `unsafe.Pointer`.

## Decision

Usamos `unsafe.Pointer` y `unsafe.Slice` en dos funciones del SDK:

1. **`BytesToWasm`** (linea ~251): copia un `[]byte` de Go a la
   memoria lineal WASM usando `unsafe.Slice((*byte)(unsafe.Pointer(...)), len)`.

2. **`SliceFromPtr`** (linea ~271): crea un `[]byte` a partir de un
   puntero y longitud crudos recibidos del host WASM.

Estas conversiones son las mismas que usa el runtime de wazero y son el
patron estandar en el ecosistema WASM de Go.

## Consecuencias

- `go vet` reporta "possible misuse of unsafe.Pointer" porque no puede
  verificar que los punteros apuntan a memoria valida. Esto es un falso
  positivo en contexto WASM donde el host (wazero) garantiza que la
  memoria lineal esta mapeada.

- Se suprime el warning con `//nolint:govet` en cada linea afectada.

- El job `vet` en CI excluye explicitamente el paquete `plugin/sdk`:
  ```yaml
  go vet $(go list ./... | grep -v /plugin/sdk)
  ```

- Los tests del SDK (`sdk_test.go`, 17 tests) verifican el
  comportamiento correcto de ambas funciones incluyendo edge cases
  (nil, vacio, datos grandes).

## Alternativas consideradas

1. **Usar `reflect.SliceHeader`**: Deprecated desde Go 1.21 y mas
   fragil que `unsafe.Slice`.

2. **Evitar unsafe completamente**: Imposible sin un ABI diferente.
   El ABI de WASM requiere punteros crudos para compartir memoria.

3. **CGo bridge**: Anade dependencia de CGo (el proyecto es pure Go
   con `CGO_ENABLED=0`). Descartado.
