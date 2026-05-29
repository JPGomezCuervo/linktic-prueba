package dotenv

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func Load(path string) (map[string]string, error) {
	r := regexp.MustCompile(`([\w_]+)\s*=\s*(.*)$`)
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	env := make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := r.FindStringSubmatch(line)

		if len(matches) != 3 {
			continue
		}

		key := matches[1]
		value := matches[2]
		env[key] = value

	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return nil, fmt.Errorf("failed to set environment variable %q: %w", k, err)
		}
	}

	return env, nil
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
