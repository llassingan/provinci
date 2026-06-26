package service

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

var forbiddenModules = []string{"shell", "command", "raw", "script", "expect", "synchronize"}

func ValidatePlaybookYAML(yamlStr string) error {
	var docs []map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &docs); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	for _, doc := range docs {
		if err := validateTasks(doc); err != nil {
			return err
		}
	}

	return nil
}

func validateTasks(doc map[string]interface{}) error {
	tasks, ok := doc["tasks"]
	if !ok {
		return nil
	}

	taskList, ok := tasks.([]interface{})
	if !ok {
		return fmt.Errorf("tasks must be a list")
	}

	for i, task := range taskList {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			return fmt.Errorf("task %d is not a valid mapping", i)
		}
		for _, module := range forbiddenModules {
			if _, exists := taskMap[module]; exists {
				return fmt.Errorf("task %d uses forbidden module %q", i, module)
			}
		}
		if err := validateBlocks(taskMap); err != nil {
			return err
		}
	}

	return nil
}

func validateBlocks(task map[string]interface{}) error {
	block, ok := task["block"]
	if !ok {
		return nil
	}
	blockList, ok := block.([]interface{})
	if !ok {
		return nil
	}
	for i, item := range blockList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for _, module := range forbiddenModules {
			if _, exists := itemMap[module]; exists {
				return fmt.Errorf("block task %d uses forbidden module %q", i, module)
			}
		}
	}
	return nil
}
