FROM node:24-alpine AS build

WORKDIR /src
COPY apps/admin-frontend/package.json apps/admin-frontend/package-lock.json ./
RUN npm ci
COPY apps/admin-frontend .
RUN npm run build

FROM nginx:1.29-alpine

COPY --from=build /src/dist /usr/share/nginx/html
COPY deploy/nginx-admin-frontend.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
