package config

import (
	"os"

	"github.com/joho/godotenv"
)

func loadDotEnvCandidates(paths []string) error {
	for _, path := range paths {
		if err := godotenv.Load(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
	}
	return nil
}
