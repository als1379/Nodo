FROM nginx:1.27-alpine

COPY index.html styles.css app.js config.js config.template.js /usr/share/nginx/html/
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY docker-entrypoint.d/40-frontend-config.sh /docker-entrypoint.d/40-frontend-config.sh

RUN chmod +x /docker-entrypoint.d/40-frontend-config.sh

EXPOSE 80
