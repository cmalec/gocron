package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	yaml "go.yaml.in/yaml/v3"
)

var writeMu sync.Mutex

// JobWriteInput carries the editable fields of a job for config write-back.
type JobWriteInput struct {
	Name        string   `json:"name"`
	Cron        string   `json:"cron"`
	DisableCron bool     `json:"disable_cron"`
	Commands    []string `json:"commands"`
}

func loadConfigDoc() (*yaml.Node, error) {
	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &doc, nil
}

func mappingFor(doc *yaml.Node, key string) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	if doc.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == key {
			return doc.Content[i+1]
		}
	}
	return nil
}

func jobsSequence(doc *yaml.Node) *yaml.Node {
	jobs := mappingFor(doc, "jobs")
	if jobs == nil || jobs.Kind != yaml.SequenceNode {
		return nil
	}
	return jobs
}

func findJobNode(seq *yaml.Node, name string) *yaml.Node {
	for _, item := range seq.Content {
		nameNode := mappingFor(item, "name")
		if nameNode != nil && strings.EqualFold(nameNode.Value, name) {
			return item
		}
	}
	return nil
}

func setScalarField(job *yaml.Node, key, value string) {
	node := mappingFor(job, key)
	if node != nil {
		node.Value = value
		node.Tag = "!!str"
		node.Kind = yaml.ScalarNode
		return
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	job.Content = append(job.Content, keyNode, valNode)
}

func setBoolField(job *yaml.Node, key string, value bool) {
	str := "false"
	if value {
		str = "true"
	}
	node := mappingFor(job, key)
	if node != nil {
		node.Value = str
		node.Tag = "!!bool"
		node.Kind = yaml.ScalarNode
		return
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: str}
	job.Content = append(job.Content, keyNode, valNode)
}

func setCommandsField(job *yaml.Node, commands []string) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, cmd := range commands {
		seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: cmd})
	}
	node := mappingFor(job, "commands")
	if node != nil {
		*node = *seq
		return
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "commands"}
	job.Content = append(job.Content, keyNode, seq)
}

func removeField(job *yaml.Node, key string) {
	for i := 0; i+1 < len(job.Content); i += 2 {
		if job.Content[i].Value == key {
			job.Content = append(job.Content[:i], job.Content[i+2:]...)
			return
		}
	}
}

func writeConfigDoc(doc *yaml.Node) error {
	var sb strings.Builder
	enc := yaml.NewEncoder(&sb)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(GetConfigFilePath(), []byte(sb.String()), 0o644)
}

// UpdateJobInConfig rewrites a job's editable fields in the YAML config file.
func UpdateJobInConfig(name string, input JobWriteInput) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	doc, err := loadConfigDoc()
	if err != nil {
		return err
	}
	seq := jobsSequence(doc)
	if seq == nil {
		return fmt.Errorf("no jobs section in config")
	}
	jobNode := findJobNode(seq, name)
	if jobNode == nil {
		return fmt.Errorf("job %q not found in config", name)
	}

	if input.Cron == "" || input.DisableCron {
		removeField(jobNode, "cron")
	} else {
		setScalarField(jobNode, "cron", input.Cron)
	}
	setBoolField(jobNode, "disable_cron", input.DisableCron)
	if len(input.Commands) > 0 {
		setCommandsField(jobNode, input.Commands)
	}

	return writeConfigDoc(doc)
}

// AddJobToConfig appends a new job to the YAML config file.
func AddJobToConfig(input JobWriteInput) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	doc, err := loadConfigDoc()
	if err != nil {
		return err
	}
	seq := jobsSequence(doc)
	if seq == nil {
		return fmt.Errorf("no jobs section in config")
	}
	if findJobNode(seq, input.Name) != nil {
		return fmt.Errorf("job %q already exists", input.Name)
	}

	jobNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setScalarField(jobNode, "name", input.Name)
	if input.DisableCron || input.Cron == "" {
		setBoolField(jobNode, "disable_cron", true)
	} else {
		setScalarField(jobNode, "cron", input.Cron)
	}
	setCommandsField(jobNode, input.Commands)

	seq.Content = append(seq.Content, jobNode)
	return writeConfigDoc(doc)
}

// DeleteJobFromConfig removes a job from the YAML config file.
func DeleteJobFromConfig(name string) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	doc, err := loadConfigDoc()
	if err != nil {
		return err
	}
	seq := jobsSequence(doc)
	if seq == nil {
		return fmt.Errorf("no jobs section in config")
	}
	for i, item := range seq.Content {
		nameNode := mappingFor(item, "name")
		if nameNode != nil && strings.EqualFold(nameNode.Value, name) {
			seq.Content = append(seq.Content[:i], seq.Content[i+1:]...)
			return writeConfigDoc(doc)
		}
	}
	return fmt.Errorf("job %q not found in config", name)
}
