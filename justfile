AIR_CONFIG := ".air.toml"

api:
    @echo "Starting server with Air hot reload..."
    air --build.cmd "go build -o ./tmp/api ./cmd/api" \
        --build.bin "./tmp/api" \
        --build.exclude_dir "tmp,vendor,docs,bin,tests" \
        --build.include_ext "go,yaml,yml" \
        --build.delay "100" \
        --log.main_only "false" \
        -c {{ AIR_CONFIG }} || air \
        --root . \
        --build.cmd "go build -o ./tmp/api ./cmd/api" \
        --build.bin "./tmp/api" \
        --build.exclude_dir "tmp,vendor,docs,bin,tests" \
        --build.include_ext "go,yaml,yml" \
        --build.delay "100"

migup:
    @echo "🚀 Launching database migrations..."
    @go run ./cmd/api migrate up

migdown:
    @echo "🚀 Launching database migrations..."
    @go run ./cmd/api migrate down

migstatus:
    @echo "🚀 Launching database migrations..."
    go run ./cmd/api migrate status
