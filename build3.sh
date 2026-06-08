#!/usr/bin/env bash
set -e

appName="iNoi"
outDir="build"
zipName="${appName}-windows-386.zip"

builtAt="$(date +'%F %T %z')"
gitCommit=$(git rev-parse --short HEAD || echo unknown)
gitAuthor="The iNoi Projects Contributors <inoi@peifeng.li>"
frontendRepo="${FRONTEND_REPO:-NecroticGlow/iNoi-Web}"
localFrontendDir="${INOI_WEB_DIR:-../iNoi-Web}"
webPackage="${INOI_WEB_DIST_TAR:-../iNoi-Web/compress/dist.tar.gz}"
webPackageUrl="${INOI_WEB_DIST_URL:-https://github.com/user-attachments/files/28699218/dist.tar.gz}"
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

version="windows-386"
webVersion=$(GetWebVersion)

echo "== Build Windows 386 =="
echo "commit: $gitCommit"

ldflags="\
-w -s \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.BuiltAt=$builtAt' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitAuthor=$gitAuthor' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitCommit=$gitCommit' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.Version=$version' \
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.WebVersion=$webVersion' \
"

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
  rm -rf dist.tar.gz
}

EnsureGoModules() {
  go get github.com/OpenListTeam/OpenList/v4/drivers/local
  go get github.com/andybalholm/cascadia@v1.3.3
  go get github.com/OpenListTeam/OpenList/v4/internal/archive/zip
  go get github.com/quic-go/quic-go/http3@v0.54.1
  go mod download
}

rm -rf "$outDir"
mkdir -p "$outDir/tmp"

FetchWebRelease
EnsureGoModules

export GOOS=windows
export GOARCH=386
export CGO_ENABLED=1
export CC=i686-w64-mingw32-gcc
export CXX=i686-w64-mingw32-g++

go build -o "$outDir/tmp/$appName.exe" -ldflags="$ldflags" -tags=jsoniter .

mkdir -p "$outDir/tmp/public"
cp -R public/dist "$outDir/tmp/public/"

cd "$outDir/tmp"
zip -r "../$zipName" "$appName.exe" "public/dist"
cd ../..

rm -rf "$outDir/tmp"

echo "== Done =="
echo "Output: $outDir/$zipName"
