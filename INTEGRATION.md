# ============================================================
# AÑADIR AL docker-compose.yml DEL REPO PRINCIPAL
# ============================================================
#
# OPCIÓN A: flashcards como subcarpeta del repo principal
#   Estructura: ~/nas/flashcards/  (git submodule o carpeta simple)
#   En el compose principal añade:
#
#   services:
#     flashcards:
#       build:
#         context: ./flashcards        # ruta relativa al repo de flashcards
#         dockerfile: Dockerfile
#       container_name: flashcards
#       restart: unless-stopped
#       volumes:
#         - ./data/flashcards:/data
#       networks:
#         - proxy
#       labels:
#         - "com.centurylinklabs.watchtower.enable=false"
#
# OPCIÓN B: flashcards como git submodule (recomendado para aislar)
#   git submodule add https://github.com/TU_USER/flashcards.git flashcards
#   El context del build sigue siendo ./flashcards
#
# ============================================================
# BLOQUE COMPLETO LISTO PARA PEGAR:
# ============================================================

  flashcards:
    build:
      context: ./flashcards
      dockerfile: Dockerfile
    container_name: flashcards
    restart: unless-stopped
    volumes:
      - ./data/flashcards:/data
    networks:
      - proxy
    labels:
      - "com.centurylinklabs.watchtower.enable=false"
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/cards/stats"]
      interval: 30s
      timeout: 5s
      retries: 3

# ============================================================
# BLOQUE NGINX — añadir dentro del bloque http {} del nginx.conf
# ============================================================

#    server {
#        listen 443 ssl;
#        http2 on;
#        server_name flashcards.villaseca.duckdns.org;
#        include /etc/nginx/authelia_location.conf;
#        location / {
#            include /etc/nginx/authelia_authrequest.conf;
#            proxy_pass http://flashcards:8080;
#            proxy_set_header Host $host;
#            proxy_set_header X-Real-IP $remote_addr;
#            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
#            proxy_set_header X-Forwarded-Proto $scheme;
#        }
#    }
