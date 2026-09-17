package app

import (
	"fmt"

	"mkrab/internal/data"

	"github.com/spf13/cobra"
)

// NewRootCommand 创建统一的 Cobra 根命令。
func NewRootCommand() *cobra.Command {
	command := &cobra.Command{
		Use:           "mkrab",
		Short:         "MQTT broker with SQLite-backed authentication and auditing",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	command.AddCommand(newServeCommand(), newGenerateCommand())
	return command
}

func newServeCommand() *cobra.Command {
	var configPath string
	command := &cobra.Command{
		Use:   "serve",
		Short: "Start the MQTT broker",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			application, cleanup, err := Initialize(configPath)
			if err != nil {
				return err
			}
			defer cleanup()
			defer application.Close()
			return application.Run()
		},
	}
	command.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "broker YAML configuration")
	return command
}

func newGenerateCommand() *cobra.Command {
	var output string
	command := &cobra.Command{
		Use:   "gen",
		Short: "Generate type-safe GORM query code",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return data.Generate(output)
		},
	}
	command.Flags().StringVarP(&output, "output", "o", "internal/data/query", "generated query output directory")
	return command
}

func Execute() error {
	if err := NewRootCommand().Execute(); err != nil {
		return fmt.Errorf("execute command: %w", err)
	}
	return nil
}
