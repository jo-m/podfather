FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/podfather /podfather
EXPOSE 8080
ENTRYPOINT ["/podfather"]
