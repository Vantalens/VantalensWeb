package config

import (
    "bufio"
    "os"
    "strings"
)

// LoadEnvFiles loads .env style files without overriding existing environment values.
func LoadEnvFiles(paths ...string) {
    for _, path := range paths {
        loadEnvFile(path)
    }
}

func loadEnvFile(path string) {
    file, err := os.Open(path)
    if err != nil {
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        idx := strings.Index(line, "=")
        if idx <= 0 {
            continue
        }

        key := strings.TrimSpace(line[:idx])
        value := strings.TrimSpace(line[idx+1:])
        value = strings.Trim(value, `"`)
        if key == "" {
            continue
        }

        if os.Getenv(key) == "" {
            _ = os.Setenv(key, value)
        }
    }
}

// GetEnvAny returns the first non-empty environment value from keys.
func GetEnvAny(keys []string, def string) string {
    for _, key := range keys {
        if v := strings.TrimSpace(os.Getenv(key)); v != "" {
            return v
        }
    }
    return def
}
