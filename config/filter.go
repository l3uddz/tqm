package config

type FilterConfiguration struct {
	Ignore []string
	Remove []string
	Label  map[string]struct {
		Ignore []string
		Move   []string
	}
}
