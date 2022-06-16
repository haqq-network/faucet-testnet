FROM node:lts-alpine as frontend

WORKDIR /frontend-build

COPY ./web/package*.json ./
RUN npm install

COPY ./web .
RUN npm run build

FROM golang AS builder

WORKDIR /src
# Download dependencies
COPY go.mod go.sum /
RUN go mod download

# Add source code
COPY . .
COPY --from=frontend /frontend-build/public ./web/public

RUN CGO_ENABLED=0 go build -o main .

# Multi-Stage production build
FROM alpine AS production
RUN apk --no-cache add ca-certificates

WORKDIR /app
# Retrieve the binary from the previous stage
COPY --from=builder /src/main .
# Expose port
EXPOSE 8080
# Set the binary as the entrypoint of the container
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# ENTRYPOINT ["/entrypoint.sh"]