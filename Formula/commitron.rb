class Commitron < Formula
  desc "AI-driven CLI tool that generates optimal, context-aware commit messages"
  homepage "https://github.com/stiliajohny/commitron"
  url "https://github.com/stiliajohny/commitron/archive/refs/tags/v0.1.4.tar.gz"
  # SHA256 checksum for v0.1.4 release
  sha256 "41199a4e1d2003036aa1a6116df7da9f0ab015858570e07a4dcaf24056041583"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"commitron", "./cmd/commitron"
  end

  test do
    system "#{bin}/commitron", "--version"
  end
end
