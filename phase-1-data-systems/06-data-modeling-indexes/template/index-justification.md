# Worksheet — justificación de índices

Completá una entrada por cada índice que agregaste en `migrations/001_schema.sql`. El
objetivo es que puedas defender cada índice ante alguien que te pregunte "¿por qué este y no
otro?", incluyendo el costo que le agrega a cada escritura sobre esa tabla.

## Plantilla por índice

Copiá este bloque por cada índice (vas a necesitar al menos tres: uno para
`ListRecentOrders`, uno para `StalePendingPayments`, uno para `LowStockSKUs`).

### Índice: `<nombre_del_indice>`

- **Tabla y columnas**: `<tabla> (<columnas>)` — ¿es un índice simple, compuesto, o
  parcial (`WHERE ...`)?
- **Qué consulta sirve**: ¿cuál de las tres funciones de `queries.go` lo necesita, y qué
  parte de esa consulta cubre (el filtro, el orden, ambos)?
- **Por qué el orden de columnas elegido** (si es compuesto): ¿por qué esa columna va
  primero y no la otra?
- **Costo en escrituras**: cada `INSERT`/`UPDATE`/`DELETE` sobre esta tabla también
  actualiza este índice. ¿Qué tan seguido se escribe esta tabla en el dominio real
  (`orders`, `payments`, `inventory`), y qué tan caro es mantener este índice en ese
  contexto?
- **Alternativa descartada**: ¿qué otro índice (u otra combinación de columnas)
  consideraste y por qué este resultó mejor?

## Preguntas generales

- Si tuvieras que elegir un solo índice para eliminar por presión de espacio en disco,
  ¿cuál sería y qué consulta se degradaría?
- `StalePendingPayments` filtra por `status = 'pending'`, pero la mayoría de los pagos no
  están en ese estado. ¿Qué ventaja tiene (si la usaste) un índice parcial (`WHERE status =
  'pending'`) sobre un índice compuesto `(status, created_at)` que cubre todos los estados?
- `EXPLAIN (FORMAT JSON)` te muestra el plan elegido para los datos actuales. ¿Qué pasaría
  con ese plan si la tabla `orders` tuviera 10 filas en vez de 100.000? ¿Por qué el
  optimizador podría preferir un sequential scan en ese caso?
