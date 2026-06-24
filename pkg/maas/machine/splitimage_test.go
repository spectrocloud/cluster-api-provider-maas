package machine

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSplitImage(t *testing.T) {
	tests := []struct {
		name         string
		image        string
		wantOSSystem string
		wantDistro   string
	}{
		{
			name:         "prefixed rhel image",
			image:        "rhel/rocky-92-0-k-1285-0",
			wantOSSystem: "rhel",
			wantDistro:   "rocky-92-0-k-1285-0",
		},
		{
			name:         "prefixed suse image",
			image:        "suse/sles-15-0-k-1304-0",
			wantOSSystem: "suse",
			wantDistro:   "sles-15-0-k-1304-0",
		},
		{
			name:         "prefixed ubuntu image",
			image:        "ubuntu/u-2204-0-k-1329-0",
			wantOSSystem: "ubuntu",
			wantDistro:   "u-2204-0-k-1329-0",
		},
		{
			name:         "legacy non-prefixed image stays custom",
			image:        "u-2204-0-k-1329-0",
			wantOSSystem: "custom",
			wantDistro:   "u-2204-0-k-1329-0",
		},
		{
			name:         "empty image stays custom",
			image:        "",
			wantOSSystem: "custom",
			wantDistro:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			osystem, distro := splitImage(tt.image)
			g.Expect(osystem).To(Equal(tt.wantOSSystem))
			g.Expect(distro).To(Equal(tt.wantDistro))
		})
	}
}
