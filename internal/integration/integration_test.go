package integration

import (
	"os"
	"testing"

	"disk-smi/internal/app"
	"disk-smi/internal/render"
)

func TestRealDiscoveryOptIn(t *testing.T) {
	if os.Getenv("DISK_SMI_INTEGRATION") != "1" {
		t.Skip("set DISK_SMI_INTEGRATION=1 to run real disk discovery")
	}
	output, err := app.Run("", "", render.Options{
		Width:   100,
		Locale:  render.LocaleEnglish,
		Summary: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" {
		t.Fatal("empty integration output")
	}
}
