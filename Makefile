# SLG Server — 全局 Makefile
#
# 工作区根目录，slg_server 是主要项目。
# 所有 go 命令使用 -C 切换到项目目录执行，确保 go.mod 正确关联。

GO      ?= go
SLG     := slg_server

.PHONY: all build vet marchdos cores clean proto

all: vet build

# ---- Build ----

build:
	$(GO) -C $(SLG) build ./...

# ---- Vet ----

vet:
	$(GO) -C $(SLG) vet ./...

vet-marchdos:
	$(GO) -C $(SLG) vet ./services/internal/cores/marchdos/...

vet-cores:
	$(GO) -C $(SLG) vet ./services/internal/cores/...

# ---- Proto ----

proto:
	pwsh -File $(SLG)/scripts/sync_proto_client.ps1

# ---- Clean ----

clean:
	$(GO) -C $(SLG) clean ./...
