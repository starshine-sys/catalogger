package common

import "runtime/debug"

// Version returns the version
func Version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "UNKNOWN"
	}

	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}

	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" {
			if len(s.Value) > 12 {
				s.Value = s.Value[:12]
			}
			return s.Value
		}
	}

	if bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "UNKNOWN"
}
