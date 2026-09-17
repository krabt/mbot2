package data

import (
	"fmt"
	"os"
	"path/filepath"

	"mkrab/internal/data/model"

	"gorm.io/gen"
)

// Generate 生成类型安全的 GORM Query。相对输出路径始终以项目根目录为基准。
func Generate(output string) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, output)
	}
	generator := gen.NewGenerator(gen.Config{
		OutPath:      output,
		ModelPkgPath: "mkrab/internal/data/model",
		Mode:         gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	generator.ApplyBasic(
		model.User{},
		model.Role{},
		model.ACLRule{},
		model.ReceivedMessage{},
		model.Topic{},
		model.ClientConnection{},
		model.UserRole{},
	)
	generator.Execute()
	return nil
}

func moduleRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("go.mod not found from working directory")
		}
		directory = parent
	}
}
