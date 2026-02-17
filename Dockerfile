FROM gcr.io/distroless/static-debian12:nonroot
COPY podview /podview
EXPOSE 8080
ENTRYPOINT ["/podview"]
