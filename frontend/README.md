# Frontend (generado por ia)

Frontend estático (HTML + CSS + JS, sin build) que consume la API de `backend/`.
Se despliega junto al backend desde la raíz del repo con `docker compose up --build`;
el backend no sabe nada de él.

```
frontend/
  public/            <- lo único que se sirve al navegador
    index.html
    app.js
    style.css
    manifest.json
    icon.svg
  Caddyfile          <- sirve public/ y hace de proxy hacia el backend
  docker-compose.yml
```

## Por qué hay un proxy delante

El backend no tiene middleware CORS. Si el navegador carga el frontend desde un
origen (`http://servidor:8081`) y llama a la API en otro (`http://servidor:8080`),
el navegador bloquea las respuestas.

Caddy resuelve esto sin tocar el backend: sirve `public/` y reenvía `/api/*` y
`/health` al backend, de modo que para el navegador **todo es el mismo origen**.

Caddy llega al backend por la red interna de Docker, usando el nombre del
servicio (`BACKEND_UPSTREAM`, por defecto `api:8080`). Por eso el frontend solo
funciona arrancado desde el `docker-compose.yml` de la raíz, junto al backend.

---

# Qué hace la v1

- **Diario → Hoy**: escribir texto y adjuntar fotos, ver las entradas del día,
  cambiar de fecha con el selector nativo, y el chip *Otros años* (`ignore_year`)
  para ver ese mismo día en años anteriores.
- **Diario → Buscar**: búsqueda por texto con paginación; al tocar un resultado
  saltas a su día.
- **Notas**: entradas atemporales (sin fecha) como lista aparte, en su propia
  pestaña del menú inferior.
- Editar y borrar en cualquiera de las dos listas.
- Toque en una miniatura para ver la foto a tamaño completo.

Sigue la paleta *Salvia* del diseño (`#eef0ea` / `#5f7a63`), Manrope + Space Mono,
tabs superiores y nav inferior.

## Simplificaciones respecto al diseño

- Selector de fecha nativo del móvil en vez del calendario dibujado a mano.
- Sin secciones Dinero, Listas ni Jornada.
- Sin galería ni "En este día" (el chip *Otros años* cubre esa idea).
- Sin hora en cada entrada: `diary.Entry` no expone `created_at`.

---

# Cosas del backend que te vas a encontrar

Ninguna está tocada, son notas de lo que vi al cablear:

1. **No se puede pasar una entrada con fecha a atemporal.** `postgres.DiaryRepository.Edit`
   usa `date = COALESCE($1, date)`, así que mandar `date` vacío no la borra.
2. **Editar adjuntos es todo o nada.** Si `Edit` recibe `keys != nil`, borra
   todos los adjuntos y mete los nuevos. Por eso la v1 solo edita el texto.
3. **Timeout de 5 s en `Insert`.** Subiendo varias fotos grandes desde el móvil
   con 4G se puede quedar corto y devolver 504.
4. **`date` vacío rompe el parseo.** `domain.Date.UnmarshalText("")` da error, así
   que el frontend directamente no manda el campo `date` para las atemporales
   (es lo que hace que se guarden como `NULL`).
5. **`limit` obligatorio entre 1 y 100** en `/search`, `/atemporal` y
   `/attachments`; sin él da 400.
6. **Si algún día quitas el proxy** y sirves el frontend desde otro origen,
   tendrás que añadir `github.com/gofiber/fiber/v2/middleware/cors` en
   `controller.CreateApp`.
