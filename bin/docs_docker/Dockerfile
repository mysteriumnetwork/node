FROM redocly/redoc AS redoc

# Install packages
RUN apk add --update --no-cache bash curl \
    && rm -rf /var/cache/apk/*

COPY tequilapi.json /usr/share/nginx/html/tequilapi.json

ENV SPEC_URL='/tequilapi.json'

RUN chmod 755 /usr/local/bin/docker-run.sh

ENTRYPOINT ["/usr/local/bin/docker-run.sh"]
