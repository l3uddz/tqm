package config

type FilterConfiguration struct {
	Ignore []string
	Remove []string
	Label  []struct {
		Name   string
		Update []string
	}
}
