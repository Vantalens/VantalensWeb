package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ArticleMetadata struct {
	Title      string   `json:"title" yaml:"title"`
	Date       string   `json:"date" yaml:"date"`
	Draft      bool     `json:"draft" yaml:"draft"`
	Pinned     bool     `json:"pinned" yaml:"pinned"`
	Categories []string `json:"categories" yaml:"categories"`
}

type ArticleDocument struct {
	Content  string          `json:"content"`
	Body     string          `json:"body"`
	Metadata ArticleMetadata `json:"metadata"`
	Revision string          `json:"revision"`
}

func parseArticleDocument(content string) (ArticleDocument, error) {
	frontmatter, body := splitFrontmatter(content)
	root, err := parseYAMLMapping(frontmatter)
	if err != nil {
		return ArticleDocument{}, fmt.Errorf("invalid front matter: %w", err)
	}
	metadata := ArticleMetadata{}
	if data, err := yaml.Marshal(root); err == nil {
		_ = yaml.Unmarshal(data, &metadata)
	}
	return ArticleDocument{Content: content, Body: body, Metadata: metadata, Revision: articleRevision(content)}, nil
}

func mergeArticleDocument(existing, body string, metadata ArticleMetadata) (string, error) {
	frontmatter, _ := splitFrontmatter(existing)
	root, err := parseYAMLMapping(frontmatter)
	if err != nil {
		return "", fmt.Errorf("invalid front matter: %w", err)
	}
	mapping := root.Content[0]
	setYAMLValue(mapping, "title", metadata.Title)
	setYAMLValue(mapping, "date", metadata.Date)
	setYAMLValue(mapping, "draft", metadata.Draft)
	setYAMLValue(mapping, "pinned", metadata.Pinned)
	setYAMLValue(mapping, "categories", metadata.Categories)
	encoded, err := yaml.Marshal(mapping)
	if err != nil {
		return "", err
	}
	return "---\n" + strings.TrimSpace(string(encoded)) + "\n---\n" + strings.TrimLeft(body, "\r\n"), nil
}

func splitFrontmatter(content string) (string, string) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return "", normalized
	}
	rest := normalized[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", normalized
	}
	after := rest[end+4:]
	if strings.HasPrefix(after, "\n") {
		after = after[1:]
	}
	return rest[:end], after
}

func parseYAMLMapping(frontmatter string) (*yaml.Node, error) {
	root := &yaml.Node{Kind: yaml.DocumentNode}
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root.Content = []*yaml.Node{mapping}
	if strings.TrimSpace(frontmatter) == "" {
		return root, nil
	}
	if err := yaml.Unmarshal([]byte(frontmatter), root); err != nil {
		return nil, err
	}
	if len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("front matter must be a YAML mapping")
	}
	return root, nil
}

func setYAMLValue(mapping *yaml.Node, key string, value interface{}) {
	valueNode := &yaml.Node{}
	_ = valueNode.Encode(value)
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = valueNode
			return
		}
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, valueNode)
}

func articleRevision(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".talentwriter-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}