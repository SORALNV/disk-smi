#!/usr/bin/env ruby
# frozen_string_literal: true

require "optparse"

module UpdateFormula
  module_function

  SHA256_RE = /\A[0-9a-f]{64}\z/
  OWNER_RE = %r{\A[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)?\z}

  def run(argv)
    options = {
      formula: File.expand_path("../Formula/disk-smi.rb", __dir__)
    }
    parser = OptionParser.new do |opts|
      opts.banner = "Usage: update_formula.rb --owner OWNER --version VERSION --sha256 SHA256 [--formula PATH]"
      opts.on("--owner OWNER", "GitHub owner or owner/tap") { |value| options[:owner] = value }
      opts.on("--version VERSION", "Release version, with or without leading v") { |value| options[:version] = value }
      opts.on("--sha256 SHA256", "Source archive SHA256") { |value| options[:sha256] = value }
      opts.on("--formula PATH", "Formula path") { |value| options[:formula] = value }
    end
    parser.parse!(argv)

    update(
      formula: options.fetch(:formula),
      owner: options.fetch(:owner),
      version: options.fetch(:version),
      sha256: options.fetch(:sha256)
    )
  rescue KeyError => e
    warn "missing required option: #{e.key}"
    warn parser
    2
  rescue ArgumentError => e
    warn e.message
    2
  end

  def update(formula:, owner:, version:, sha256:)
    validate_owner(owner)
    version = normalize_version(version)
    validate_sha256(sha256)

    content = File.read(formula)
    content = replace_line(content, /^  homepage ".*"$/, %(  homepage "https://github.com/#{repo_owner(owner)}/disk-smi"))
    content = replace_line(content, /^  url ".*"$/, %(  url "https://github.com/#{repo_owner(owner)}/disk-smi/archive/refs/tags/v#{version}.tar.gz"))
    content = replace_line(content, /^  sha256 ".*"$/, %(  sha256 "#{sha256}"))
    File.write(formula, content)
    0
  end

  def validate_owner(owner)
    raise ArgumentError, "invalid owner: #{owner.inspect}" unless owner&.match?(OWNER_RE)
  end

  def normalize_version(version)
    value = version.to_s.sub(/\Av/, "")
    raise ArgumentError, "invalid version: #{version.inspect}" unless value.match?(/\A\d+\.\d+\.\d+([.-][A-Za-z0-9_.-]+)?\z/)

    value
  end

  def validate_sha256(sha256)
    raise ArgumentError, "invalid sha256" unless sha256&.match?(SHA256_RE)
  end

  def repo_owner(owner)
    owner.split("/", 2).first
  end

  def replace_line(content, pattern, replacement)
    raise ArgumentError, "formula line not found for #{pattern.inspect}" unless content.match?(pattern)

    content.sub(pattern, replacement)
  end
end

exit(UpdateFormula.run(ARGV)) if $PROGRAM_NAME == __FILE__
