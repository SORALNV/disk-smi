# frozen_string_literal: true

require "fileutils"
require "minitest/autorun"
require "tmpdir"

require_relative "update_formula"

class UpdateFormulaTest < Minitest::Test
  SHA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

  def test_updates_owner_version_and_sha
    Dir.mktmpdir do |dir|
      formula = File.join(dir, "disk-smi.rb")
      FileUtils.cp(File.expand_path("../Formula/disk-smi.rb", __dir__), formula)

      code = UpdateFormula.run([
        "--formula", formula,
        "--owner", "example",
        "--version", "v1.2.3",
        "--sha256", SHA
      ])

      assert_equal 0, code
      content = File.read(formula)
      assert_includes content, %(homepage "https://github.com/example/disk-smi")
      assert_includes content, %(url "https://github.com/example/disk-smi/archive/refs/tags/v1.2.3.tar.gz")
      assert_includes content, %(sha256 "#{SHA}")
    end
  end

  def test_owner_tap_uses_repo_owner_for_source_url
    Dir.mktmpdir do |dir|
      formula = File.join(dir, "disk-smi.rb")
      FileUtils.cp(File.expand_path("../Formula/disk-smi.rb", __dir__), formula)

      code = UpdateFormula.run([
        "--formula", formula,
        "--owner", "example/homebrew-tap",
        "--version", "1.2.3",
        "--sha256", SHA
      ])

      assert_equal 0, code
      assert_includes File.read(formula), %(url "https://github.com/example/disk-smi/archive/refs/tags/v1.2.3.tar.gz")
    end
  end

  def test_rejects_invalid_sha
    Dir.mktmpdir do |dir|
      formula = File.join(dir, "disk-smi.rb")
      FileUtils.cp(File.expand_path("../Formula/disk-smi.rb", __dir__), formula)

      code = UpdateFormula.run([
        "--formula", formula,
        "--owner", "example",
        "--version", "1.2.3",
        "--sha256", "not-a-sha"
      ])

      assert_equal 2, code
    end
  end
end
