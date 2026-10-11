TINYGO ?= tinygo
TAGS = -tags fat_noexfat

test:
	go vet ./...
	go test ./...

smoketest:
	$(TINYGO) build -o test.bin -size=short -target=xiao-esp32c3 $(TAGS) ./devices/xiao-expansion
	$(TINYGO) build -o test.uf2 -size=short -target=xiao-rp2350 $(TAGS) ./devices/xiao-expansion
	$(TINYGO) build -o test.uf2 -size=short -target=pico2 $(TAGS) ./devices/pico2-ssd1306
	$(TINYGO) build -o test.uf2 -size=short -target=gopher-badge $(TAGS) ./devices/gopher-badge
	@rm -f test.bin test.uf2

.PHONY: test smoketest
