FROM gcr.io/distroless/static-debian12:nonroot
COPY podfather /podfather
EXPOSE 8080
ENTRYPOINT ["/podfather"]
