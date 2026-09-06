class Wee < Formula
  desc "Wee - Control center and wrapper for Claude Code"
  homepage "https://github.com/schlunsen/wee"
  version "1.8.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/schlunsen/wee/releases/download/v1.8.0/wee-darwin-arm64.tar.gz"
      sha256 "4d6eb3cdf927f05e6842842b714e295a9dd8db641b28b1d586ef5e975ee8070f"
    else
      url "https://github.com/schlunsen/wee/releases/download/v1.8.0/wee-darwin-amd64.tar.gz"
      sha256 "3fccbf96f315a4ab1ce2b0a238f9caf198c77f128eca379376db22ac16815ada"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/schlunsen/wee/releases/download/v1.8.0/wee-linux-arm64"
      sha256 "b3bcb7a55665fab6ebf97dbf9c64d15e5519d2da80acbfa60d6a9fe381d9d8d5"
    else
      url "https://github.com/schlunsen/wee/releases/download/v1.8.0/wee-linux-amd64"
      sha256 "62930d2ddb71fdb43cbb6cc2a7fe90987acb4926bfd03ad9f2a9911b89cbcd92"
    end
  end

  def install
    if OS.mac?
      bin.install "bin/wee"
      (lib/"wee").install Dir["lib/*.dylib"]
    else
      bin.install Dir["wee-*"].first => "wee"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/wee --version 2>&1", 0)
  end
end
