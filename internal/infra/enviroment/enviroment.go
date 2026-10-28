package enviroment

import "os"

type EnvVariables struct {
	DLQ_URL           string
	EVENTS_TABLE_NAME string
	SCHEMA_TABLE_NAME string
}

var EnvironmentVariables = Load()

func Load() EnvVariables {
	return EnvVariables{
		DLQ_URL:           os.Getenv("DLQ_URL"),
		SCHEMA_TABLE_NAME: os.Getenv("SCHEMA_TABLE_NAME"),
		EVENTS_TABLE_NAME: os.Getenv("EVENTS_TABLE_NAME"),
	}
}
