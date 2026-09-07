#!/usr/bin/env bash
set -euo pipefail

: "${RELEASE_OS:?}" "${RELEASE_ARCH:?}" "${RELEASE_TRIPLE:?}"
package_dir="$(mktemp -d)"
trap 'rm -rf "$package_dir"' EXIT
mkdir -p "$package_dir/bin" "$package_dir/lib" "$package_dir/licenses" dist
GOOS="$RELEASE_OS" GOARCH="$RELEASE_ARCH" go build -trimpath -ldflags='-s -w' -o "$package_dir/bin/wee" ./cmd/wee
cp LICENSE "$package_dir/"
cp licenses/*.txt licenses/THIRD_PARTY_NOTICES.md "$package_dir/licenses/"
# Include license/notice files from downloaded Go modules, including transitive dependencies.
python3 - "$package_dir/licenses" <<'PY'
import json, pathlib, subprocess, sys
out = pathlib.Path(sys.argv[1]) / 'go-modules'
raw = subprocess.check_output(['go', 'list', '-m', '-json', 'all'], text=True)
decoder = json.JSONDecoder()
while raw.strip():
    obj, end = decoder.raw_decode(raw.lstrip())
    raw = raw.lstrip()[end:]
    if obj.get('Main') or not obj.get('Dir'):
        continue
    root = pathlib.Path(obj['Dir'])
    for source in root.iterdir():
        if source.is_file() and source.name.lower().startswith(('license', 'licence', 'copying', 'notice', 'copyright')):
            dest = out / (obj['Path'] + '@' + obj.get('Version', 'unknown')) / source.name
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_bytes(source.read_bytes())
PY
if [[ "$RELEASE_OS" == linux ]]; then
  module_dir="$(go list -m -f '{{.Dir}}' github.com/k2-fsa/sherpa-onnx-go-linux)"
  cp "$module_dir/lib/$RELEASE_TRIPLE/"*.so "$package_dir/lib/"
  chmod u+w "$package_dir/lib/"*
  # Preserve upstream SONAMEs as local aliases (e.g. libonnxruntime.so.1).
  for lib in "$package_dir/lib/"*.so; do
    soname="$(patchelf --print-soname "$lib")"
    if [[ -n "$soname" && "$soname" != "$(basename "$lib")" ]]; then
      ln -sf "$(basename "$lib")" "$package_dir/lib/$soname"
    fi
    patchelf --set-rpath '$ORIGIN' "$lib"
  done
  patchelf --set-rpath '$ORIGIN/../lib' "$package_dir/bin/wee"
else
  module_dir="$(go list -m -f '{{.Dir}}' github.com/k2-fsa/sherpa-onnx-go-macos)"
  cp "$module_dir/lib/$RELEASE_TRIPLE/"*.dylib "$package_dir/lib/"
  chmod u+w "$package_dir/lib/"*
  install_name_tool -rpath "$module_dir/lib/$RELEASE_TRIPLE" '@executable_path/../lib' "$package_dir/bin/wee"
  for lib in "$package_dir/lib/"*.dylib; do
    codesign --force --sign - "$lib"
  done
  codesign --force --sign - "$package_dir/bin/wee"
fi
# Remove access to module-cache libraries during the smoke test.
lib_source="$module_dir/lib/$RELEASE_TRIPLE"
chmod u+w "$(dirname "$lib_source")"
mv "$lib_source" "$lib_source.release-test"
restore() {
  mv "$lib_source.release-test" "$lib_source"
  rm -rf "$package_dir"
}
trap restore EXIT
"$package_dir/bin/wee" --version
mkdir "$package_dir/extracted"
tar -czf "dist/wee-$RELEASE_OS-$RELEASE_ARCH.tar.gz" -C "$package_dir" bin lib licenses LICENSE
tar -xzf "dist/wee-$RELEASE_OS-$RELEASE_ARCH.tar.gz" -C "$package_dir/extracted"
"$package_dir/extracted/bin/wee" --version
