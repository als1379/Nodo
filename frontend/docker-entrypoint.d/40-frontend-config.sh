#!/bin/sh
set -eu

envsubst '${NODO_API_URL}' \
  < /usr/share/nginx/html/config.template.js \
  > /usr/share/nginx/html/config.js
