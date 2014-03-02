
run:
	go run bake.go > bakedfiles.go
	go run cmd/gofill/main.go
