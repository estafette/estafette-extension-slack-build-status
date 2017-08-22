FROM scratch

LABEL maintainer="estafette.io" \
      description="The estafette-extension-slack-build-status component is an Estafette extension to send build status updates to Slack for builds handled by Estafette CI"

COPY ca-certificates.crt /etc/ssl/certs/
COPY estafette-extension-slack-build-status /

ENTRYPOINT ["/estafette-extension-slack-build-status"]