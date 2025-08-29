class Commitron < Formula
  desc "AI-driven CLI tool that generates optimal, context-aware commit messages"
  homepage "https://github.com/stiliajohny/commitron"
  url "https://github.com/stiliajohny/commitron/archive/refs/tags/v0.1.4.tar.gz"
  # SHA256 checksum for v0.1.4 release
  sha256 "41199a4e1d2003036aa1a6116df7da9f0ab015858570e07a4dcaf24056041583"

  depends_on "go" => :build

  def install
    # Extract version from the URL
    version = url.match(/v(\d+\.\d+\.\d+)/)[1]
    commit_sha = `git rev-parse HEAD`.strip
    build_date = Time.now.utc.strftime("%Y-%m-%d %H:%M:%S UTC")
    
    system "go", "build", 
           "-ldflags", "-X github.com/johnstilia/commitron/pkg/version.Version=#{version} -X github.com/johnstilia/commitron/pkg/version.CommitSHA=#{commit_sha} -X github.com/johnstilia/commitron/pkg/version.BuildDate='#{build_date}'",
           "-o", bin/"commitron", "./cmd/commitron"
  end

  test do
    system "#{bin}/commitron", "--version"
  end
end
