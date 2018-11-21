FROM redocly/redoc AS redoc

# Install packages
RUN apk add --update --no-cache bash curl \
    && rm -rf /var/cache/apk/*

COPY tequilapi.json /usr/share/nginx/html/tequilapi.json

ENV SPEC_URL='/tequilapi.json'
ENV PORT=8081
EXPOSE 8081

RUN chmod 755 /usr/local/bin/docker-run.sh

ENTRYPOINT ["/usr/local/bin/docker-run.sh"]
