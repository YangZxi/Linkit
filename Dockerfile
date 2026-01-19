FROM debian:bookworm-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY ./dist/ ./
RUN ln -s /app/linkit /usr/local/bin/linkit

EXPOSE 3301
ENV PORT=3301 \
    FRONTEND_ORIGIN=*

CMD ["/app/linkit"]
