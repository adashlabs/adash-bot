FROM node:24-bookworm-slim AS dependencies
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev

FROM node:24-bookworm-slim
WORKDIR /app
ENV NODE_ENV=production
ENV ADASH_DB_PATH=/app/data/adash.db
RUN apt-get update \
  && apt-get install -y --no-install-recommends gosu tini \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p /app/data \
  && chown -R node:node /app
COPY --from=dependencies --chown=node:node /app/node_modules ./node_modules
COPY --chown=node:node package.json package-lock.json ./
COPY --chown=node:node src ./src
COPY docker-entrypoint.sh /usr/local/bin/adash-entrypoint
RUN chmod 755 /usr/local/bin/adash-entrypoint
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/adash-entrypoint"]
CMD ["node", "src/sharder.js"]
