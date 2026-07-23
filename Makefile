IMAGE  := poker
PORT   := 7878

.PHONY: dev prod build stop logs

dev:
	docker build -f Dockerfile.dev -t $(IMAGE)-dev .
	docker run --rm -it \
		-p $(PORT):$(PORT) \
		-v $(PWD):/app \
		--name $(IMAGE)-dev \
		$(IMAGE)-dev

prod:
	docker build -t $(IMAGE) .
	docker run --rm -d \
		-p $(PORT):$(PORT) \
		--name $(IMAGE) \
		$(IMAGE)

build:
	docker build -t $(IMAGE) .

stop:
	-docker stop $(IMAGE) 2>/dev/null
	-docker stop $(IMAGE)-dev 2>/dev/null

logs:
	docker logs -f $(IMAGE)
