package config

type ConfigFile interface {
	Read(path string) error
	Write(path string) error
	Set(key string, value interface{})
}
