#!/usr/bin/env bash
set -e

# 基本信息
appName="iNoi"
builtAt="$(date +'%F %T %z')"
gitAuthor="The iNoi Projects Contributors <inoi@peifeng.li>"
gitCommit=$(git log --pretty=format:"%h" -1 || echo "0000000")
frontendRepo="${FRONTEND_REPO:-NecroticGlow/iNoi-Web}"
localFrontendDir="${INOI_WEB_DIR:-../iNoi-Web}"
webPackage="${INOI_WEB_DIST_TAR:-../iNoi-Web/compress/dist.tar.gz}"
webPackageUrl="${INOI_WEB_DIST_URL:-https://github.com/user-attachments/files/28699218/dist.tar.gz}"

# GitHub Token HTTP Header（可选）
githubAuthArgs=""
if [ -n "$GITHUB_TOKEN" ]; then
  githubAuthArgs="--header \"Authorization: Bearer $GITHUB_TOKEN\""
fi

GetWebVersion() {
  if [ -d "$localFrontendDir/.git" ]; then
    git -C "$localFrontendDir" rev-parse --short HEAD 2>/dev/null && return
  fi

  web_tag=$(eval "curl -fsSL --max-time 2 $githubAuthArgs \"https://api.github.com/repos/${frontendRepo}/releases/latest\"" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g' || true)
  if [ -n "$web_tag" ]; then
    echo "$web_tag"
  else
    echo "unknown"
  fi
}

# 版本信息
if [ "$1" = "dev" ]; then
  version="dev"
  webVersion=$(GetWebVersion)
else
  # 如果没有 tag，用默认 main
  version=$(git describe --abbrev=0 --tags 2>/dev/null || echo "0.0.0-main")
  webVersion=$(GetWebVersion)
fi

echo "backend version: $version"
echo "frontend version: $webVersion"

# Go 链接参数
ldflags="\
-w -s \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.BuiltAt=$builtAt' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitAuthor=$gitAuthor' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitCommit=$gitCommit' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.Version=$version' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.WebVersion=$webVersion' \
"

# ----------------------------
# 前端资源下载
# ----------------------------
FetchWebRelease() {
  if [ -f "$webPackage" ]; then
    echo "using local frontend package: $webPackage"
    cp "$webPackage" dist.tar.gz
  else
    echo "downloading frontend package from ${webPackageUrl}"
    curl -fL "$webPackageUrl" -o dist.tar.gz
  fi

  rm -rf dist
  tar -zxvf dist.tar.gz
  rm -rf public/dist
  mv -f dist public
  rm -f dist.tar.gz
}

EnsureGoModules() {
  go get github.com/OpenListTeam/OpenList/v4/drivers/local
  go get github.com/andybalholm/cascadia@v1.3.3
  go get github.com/OpenListTeam/OpenList/v4/internal/archive/zip
  go get github.com/quic-go/quic-go/http3@v0.54.1
  go mod download
}

# ----------------------------
# Docker 多平台构建
# ----------------------------
PrepareBuildDockerMusl() {
  mkdir -p build/musl-libs
  BASE="https://github.com/OpenListTeam/musl-compilers/releases/latest/download/"
  FILES=(x86_64-linux-musl-cross aarch64-linux-musl-cross i486-linux-musl-cross s390x-linux-musl-cross armv6-linux-musleabihf-cross armv7l-linux-musleabihf-cross riscv64-linux-musl-cross powerpc64le-linux-musl-cross)
  for i in "${FILES[@]}"; do
    url="${BASE}${i}.tgz"
    lib_tgz="build/${i}.tgz"
    curl -fsSL -o "${lib_tgz}" "${url}"
    tar xf "${lib_tgz}" --strip-components 1 -C build/musl-libs
    rm -f "${lib_tgz}"
  done
}

BuildDockerMultiplatform() {
  go mod download
  export PATH=$PATH:$PWD/build/musl-libs/bin
  export CGO_ENABLED=1
  docker_lflags="--extldflags '-static -fpic' $ldflags"

  # 标准 Linux 平台
  OS_ARCHES=(linux-amd64 linux-arm64 linux-386 linux-s390x linux-riscv64 linux-ppc64le)
  CGO_ARGS=(x86_64-linux-musl-gcc aarch64-linux-musl-gcc i486-linux-musl-gcc s390x-linux-musl-gcc riscv64-linux-musl-gcc powerpc64le-linux-musl-gcc)
  for i in "${!OS_ARCHES[@]}"; do
    os_arch=${OS_ARCHES[$i]}
    cgo_cc=${CGO_ARGS[$i]}
    os=${os_arch%%-*}
    arch=${os_arch##*-}
    export GOOS=$os
    export GOARCH=$arch
    export CC=${cgo_cc}
    echo "building for $os_arch"
    go build -o build/$os/$arch/"$appName" -ldflags="$docker_lflags" -tags=jsoniter .
  done

  # ARM 平台
  DOCKER_ARM_ARCHES=(linux-arm/v6 linux-arm/v7)
  CGO_ARGS=(armv6-linux-musleabihf-gcc armv7l-linux-musleabihf-gcc)
  GO_ARM=(6 7)
  export GOOS=linux
  export GOARCH=arm
  for i in "${!DOCKER_ARM_ARCHES[@]}"; do
    docker_arch=${DOCKER_ARM_ARCHES[$i]}
    cgo_cc=${CGO_ARGS[$i]}
    export GOARM=${GO_ARM[$i]}
    export CC=${cgo_cc}
    echo "building for $docker_arch"
    go build -o build/${docker_arch%%-*}/${docker_arch##*-}/"$appName" -ldflags="$docker_lflags" -tags=jsoniter .
  done
}

# ----------------------------
# 主流程
# ----------------------------
if [ "$1" = "dev" ]; then
  echo "== Building Dev Version =="
  FetchWebRelease
  EnsureGoModules
  BuildDockerMultiplatform
elif [ "$1" = "release" ]; then
  echo "== Building Release Version =="
  FetchWebRelease
  EnsureGoModules
  PrepareBuildDockerMusl
  BuildDockerMultiplatform
else
  echo "Usage: $0 [dev|release]"
  exit 1
fi

echo "Build complete!"
