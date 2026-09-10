# Frontend — v1 de cableado

Frontend estático (HTML + CSS + JS, sin build) que consume la API de `backend/`.
Se despliega por separado: el backend no sabe nada de él.

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
Backend y frontend siguen siendo dos despliegues independientes.

---

# Lo que tienes que hacer tú

## 1. Levantar el backend (sin cambios)

```powershell
cd backend
docker compose up -d          # Postgres
.\scripts\run-env.ps1         # API en :8080
```

## 2. Levantar el frontend

```powershell
cd frontend
docker compose up -d          # Caddy en :8081
```

Abre `http://localhost:8081`. Si el backend no corre en la misma máquina que
Docker, pon su dirección en `BACKEND_UPSTREAM` antes de levantar:

```powershell
$env:BACKEND_UPSTREAM = "192.168.1.50:8080"; docker compose up -d
```

**Sin Docker** (Caddy instalado nativamente, útil en el servidor de casa):

```bash
cd frontend
BACKEND_UPSTREAM=localhost:8080 SITE_ROOT=./public caddy run
```

## 3. Acceso desde el móvil con Tailscale

1. Crea cuenta en <https://tailscale.com> (plan Personal, gratis).
2. En el servidor de casa: instala Tailscale y `sudo tailscale up`.
   - Activa MagicDNS en el panel de admin para poder usar el nombre del equipo.
3. En el móvil: instala la app de Tailscale y entra con la misma cuenta.
4. Desde el móvil, abre `http://<nombre-del-servidor>:8081`.

No hay que abrir ningún puerto del router: Tailscale monta una VPN mesh cifrada
con conexiones salientes.

**Opcional — HTTPS y URL más corta.** En el servidor:

```bash
tailscale serve --bg 8081
```

Te da `https://<nombre>.<tu-tailnet>.ts.net` con certificado válido. Merece la
pena porque HTTPS es requisito para que Chrome ofrezca instalar la PWA.

## 4. Instalarla en el móvil

Con la página abierta: menú del navegador → **Añadir a pantalla de inicio**.
Arranca a pantalla completa, sin barra de direcciones.

> El icono es un SVG. iOS quiere PNG para el icono de pantalla de inicio, así que
> allí saldrá una captura en vez del icono. Si te molesta, exporta `icon.svg` a
> `icon-180.png` y añade `<link rel="apple-touch-icon" href="icon-180.png">`.

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
- Sin hora en cada entrada: `core.DiaryEntry` no expone `created_at`.

---

# Cosas del backend que te vas a encontrar

Ninguna está tocada, son notas de lo que vi al cablear:

1. **No se puede pasar una entrada con fecha a atemporal.** `PostgresDiary.Edit`
   usa `date = COALESCE($1, date)`, así que mandar `date` vacío no la borra.
2. **Editar adjuntos es todo o nada.** Si `Edit` recibe `keys != nil`, borra
   todos los adjuntos y mete los nuevos. Por eso la v1 solo edita el texto.
3. **Timeout de 5 s en `Insert`.** Subiendo varias fotos grandes desde el móvil
   con 4G se puede quedar corto y devolver 504.
4. **`date` vacío rompe el parseo.** `core.Date.UnmarshalText("")` da error, así
   que el frontend directamente no manda el campo `date` para las atemporales
   (es lo que hace que se guarden como `NULL`).
5. **`limit` obligatorio entre 1 y 100** en `/search`, `/atemporal` y
   `/attachments`; sin él da 400.
6. **Si algún día quitas el proxy** y sirves el frontend desde otro origen,
   tendrás que añadir `github.com/gofiber/fiber/v2/middleware/cors` en
   `api.CreateApp`.
