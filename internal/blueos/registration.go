package blueos

import (
	"errors"
	"fmt"
	"strings"
)

type Registration struct {
	Name                 string            `json:"name"`
	Description          string            `json:"description"`
	Icon                 string            `json:"icon"`
	Company              string            `json:"company"`
	Version              string            `json:"version"`
	Webpage              string            `json:"webpage"`
	API                  string            `json:"api"`
	NewPage              bool              `json:"new_page"`
	AvoidIframes         bool              `json:"avoid_iframes"`
	WorksInRelativePaths bool              `json:"works_in_relative_paths"`
	Extras               map[string]string `json:"extras,omitempty"`
}

func DefaultRegistration() Registration {
	return Registration{
		Name:                 "SemLink Companion",
		Description:          "MAVLink companion and mesh node for boat-class vehicles.",
		Icon:                 "mdi-access-point-network",
		Company:              "C360 Studio",
		Version:              "0.1.0",
		Webpage:              "https://github.com/C360Studio/semlink",
		API:                  "https://github.com/C360Studio/semlink/blob/main/docs/blueos-extension.md",
		NewPage:              false,
		AvoidIframes:         false,
		WorksInRelativePaths: true,
		Extras: map[string]string{
			"runtime":  "semlink-companion",
			"mavlink":  "udp-ingress",
			"commands": "blocked-on-hardware",
		},
	}
}

func (r Registration) Validate() error {
	var missing []string
	if strings.TrimSpace(r.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(r.Description) == "" {
		missing = append(missing, "description")
	}
	if strings.TrimSpace(r.Icon) == "" {
		missing = append(missing, "icon")
	}
	if strings.TrimSpace(r.Company) == "" {
		missing = append(missing, "company")
	}
	if strings.TrimSpace(r.Version) == "" {
		missing = append(missing, "version")
	}
	if strings.TrimSpace(r.Webpage) == "" {
		missing = append(missing, "webpage")
	}
	if strings.TrimSpace(r.API) == "" {
		missing = append(missing, "api")
	}
	if len(missing) > 0 {
		return fmt.Errorf("blueos registration missing %s", strings.Join(missing, ", "))
	}
	if !r.WorksInRelativePaths {
		return errors.New("blueos registration must work in relative extension paths")
	}
	return nil
}
