class Wee < Formula
  desc "Wee - Control center and wrapper for Claude Code"
  homepage "https://github.com/schlunsen/wee-editor"
  version "1.15.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/schlunsen/wee-editor/releases/download/v1.15.0/wee-darwin-arm64.tar.gz"
      sha256 "a067fbf7d294fc158e9d420adb3f6423fa1291dc7021dbc963a9f12a1e8abea2"
    else
      url "https://github.com/schlunsen/wee-editor/releases/download/v1.15.0/wee-darwin-amd64.tar.gz"
      sha256 "3a860ec051f63214019e223c7695004ae694f7584da1f2ef629e370384bf2c70"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/schlunsen/wee-editor/releases/download/v1.15.0/wee-linux-arm64.tar.gz"
      sha256 "eda4d2df398ba0fd4a3aa128fbe1d4a3f9e6f7b03e5bc8879d74b3dd785a2cf6"
    else
      url "https://github.com/schlunsen/wee-editor/releases/download/v1.15.0/wee-linux-amd64.tar.gz"
      sha256 "0d26a3251bb0407c692aa365ce91e4191fdc5bfabd20fb3ae4b9ead9ebb94782"
    end
  end

  def install
    # Every release tarball ships bin/wee plus its runtime libraries in lib/.
    # The binary's rpath is <exe>/../lib on both platforms, so keep that layout.
    bin.install "bin/wee"
    lib.install Dir["lib/*"]
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/wee --version 2>&1", 0)
  end
end
