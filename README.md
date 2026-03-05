# flashcards

Aplicación de flashcards para memorización por montones (Leitner simplificado).

**Backend:** Go + SQLite (~5-10MB RAM)  
**Frontend:** HTML/JS vanilla (sin build step)  
**Imagen final:** ~15MB Alpine

## Uso con el NAS (repo principal)

Este repo se referencia desde el docker-compose principal. Ver sección de integración más abajo.

## Estructura

```
flashcards/
├── main.go          # Backend Go completo
├── go.mod           # Dependencias (solo modernc.org/sqlite)
├── Dockerfile       # Multistage build
└── static/
    └── index.html   # Frontend completo
```

## Añadir tarjetas

Accede a `https://flashcards.villaseca.duckdns.org` → pestaña "➕ Añadir".

Categorías disponibles: **Leyes** / **Tecnología**

## Atajos de teclado (modo test)

| Tecla | Acción |
|-------|--------|
| `Espacio` | Voltear tarjeta |
| `1` | Mover a ❌ Falladas |
| `2` | Mover a 🟡 A medias |
| `3` | Mover a ✅ Me lo sé |

## Exportar / Importar

Gestionar → Exportar JSON/CSV — Importar JSON

Formato de importación:
```json
[
  {"question": "Estándar de AES", "answer": "FIPS 197", "category": "Tecnología"},
  {"question": "Artículo 18 CE", "answer": "Derecho a la intimidad", "category": "Leyes"}
]
```
