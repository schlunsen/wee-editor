# Third-party notices

Wee is distributed under the MIT License (see `LICENSE` at the repository root).
That license covers Wee's own source code only. The components listed below are
copyright their respective owners and are distributed under their own terms.

## Bundled with release archives

The Linux and macOS release archives (`wee-<os>-<arch>.tar.gz`) ship prebuilt shared
libraries alongside the `wee` binary under `lib/`. Their licenses are reproduced
in this directory and are included in the archives under `licenses/`.

| Component | Version | Files | License | Text |
| --- | --- | --- | --- | --- |
| ONNX Runtime (Microsoft) | 1.23.2 | `lib/libonnxruntime.1.23.2.dylib`, `lib/libonnxruntime.dylib` | MIT | [`onnxruntime-LICENSE.txt`](onnxruntime-LICENSE.txt) |
| sherpa-onnx (k2-fsa) | 1.12.30 | `lib/libsherpa-onnx-c-api.dylib`, `lib/libsherpa-onnx-cxx-api.dylib` | Apache-2.0 | [`sherpa-onnx-LICENSE.txt`](sherpa-onnx-LICENSE.txt) |

Linux archives include the corresponding `.so` libraries from
`github.com/k2-fsa/sherpa-onnx-go-linux` under the same upstream licenses.

The dylibs are obtained from the `github.com/k2-fsa/sherpa-onnx-go-macos`
Go module, which redistributes upstream ONNX Runtime and sherpa-onnx builds.

Sources:

- ONNX Runtime — <https://github.com/microsoft/onnxruntime/blob/v1.23.2/LICENSE>
- sherpa-onnx — <https://github.com/k2-fsa/sherpa-onnx/blob/v1.12.30/LICENSE>

## Statically linked Go dependencies

The `wee` binary links the Go modules declared in `go.mod`. Release packages include available module license and notice files under
`licenses/go-modules/`. Generate a current dependency manifest with:

```bash
go install github.com/google/go-licenses@latest
go-licenses report ./cmd/wee
```

## Media in this repository

See [`docs/MEDIA_PROVENANCE.md`](../docs/MEDIA_PROVENANCE.md) for the origin and
terms of images and audio tracked in this repository. The root MIT license is not
asserted over third-party media.
