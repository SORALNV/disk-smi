class DiskSmi < Formula
  desc "macOS SSD status viewer for the terminal"
  homepage "https://github.com/OWNER/disk-smi"
  url "https://github.com/OWNER/disk-smi/archive/refs/tags/v0.0.0.tar.gz"
  sha256 "CHANGE_ME"
  license "MIT"

  depends_on "go" => :build
  depends_on "smartmontools"

  def install
    ldflags = %W[
      -s -w
      -X disk-smi/internal/version.Version=#{version}
      -X disk-smi/internal/version.Commit=homebrew
    ]
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/disk-smi"
  end

  test do
    (testpath/"nvme-good.json").write <<~JSON
      {
        "device": {"name": "/dev/disk0", "type": "nvme", "protocol": "NVMe"},
        "model_name": "APPLE SSD AP1024Z",
        "serial_number": "SYNTHETIC9K2A",
        "firmware_version": "874.120.9",
        "nvme_total_capacity": 1000204886016,
        "smart_status": {"passed": true},
        "nvme_smart_health_information_log": {
          "critical_warning": 0,
          "temperature": 36,
          "available_spare": 100,
          "available_spare_threshold": 10,
          "percentage_used": 30,
          "data_units_read": 6835938,
          "data_units_written": 8203125,
          "host_read_commands": 82019334,
          "host_write_commands": 61274004,
          "power_on_hours": 1500,
          "power_cycles": 428,
          "unsafe_shutdowns": 7,
          "media_errors": 0
        }
      }
    JSON
    assert_match "disk-smi", shell_output("#{bin}/disk-smi --version")
    assert_match "APPLE SSD AP1024Z", shell_output("#{bin}/disk-smi --input #{testpath}/nvme-good.json")
  end
end
