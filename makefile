all:
	docker buildx build --platform linux/amd64,linux/arm64 -t epracsupply/estock_backend:v1.0.0 . --push