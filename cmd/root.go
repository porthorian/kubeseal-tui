package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/porthorian/kubeseal-tui/internal/app"
)

const (
	defaultSecretType          = "Opaque"
	defaultControllerNamespace = "sealed-secrets"
	defaultControllerName      = "sealed-secrets"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type rootOptions struct {
	secretName          string
	secretType          string
	controllerNamespace string
	controllerName      string
	force               bool
	dataValues          []string
	dataFiles           []string
	targets             []string
}

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	opts := rootOptions{
		secretType:          defaultSecretType,
		controllerNamespace: defaultControllerNamespace,
		controllerName:      defaultControllerName,
	}

	rootCmd := &cobra.Command{
		Use:           "kubeseal-tui",
		Short:         "Generate SealedSecret manifests with an interactive prompt or flags",
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unexpected argument: %s", args[0])
			}
			return run(cmd.Context(), cmd, opts)
		},
	}

	rootCmd.Flags().StringVar(&opts.secretName, "name", "", "Secret metadata.name (required in non-interactive mode)")
	rootCmd.Flags().StringVar(&opts.secretType, "type", defaultSecretType, "Secret type")
	rootCmd.Flags().StringArrayVar(&opts.dataValues, "data", nil, "Add plaintext data entry as KEY=VALUE (repeatable)")
	rootCmd.Flags().StringArrayVar(&opts.dataFiles, "data-file", nil, "Add data entry from file contents as KEY=FILEPATH (repeatable)")
	rootCmd.Flags().StringArrayVar(&opts.targets, "target", nil, "Output target as NAMESPACE=OUTPUT_DIR (repeatable)")
	rootCmd.Flags().StringVar(&opts.controllerNamespace, "controller-namespace", defaultControllerNamespace, "Sealed Secrets controller namespace")
	rootCmd.Flags().StringVar(&opts.controllerName, "controller-name", defaultControllerName, "Sealed Secrets controller name")
	rootCmd.Flags().BoolVar(&opts.force, "force", false, "Overwrite existing outputs in non-interactive mode")

	return rootCmd
}

func run(ctx context.Context, cmd *cobra.Command, opts rootOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	runner := app.ExecKubesealRunner{Path: "kubeseal"}
	if err := runner.CheckAvailable(); err != nil {
		return err
	}

	nonInteractive := len(opts.dataValues) > 0 || len(opts.dataFiles) > 0 || len(opts.targets) > 0
	cfg := app.Config{
		SecretName:          opts.secretName,
		SecretType:          opts.secretType,
		ControllerNamespace: opts.controllerNamespace,
		ControllerName:      opts.controllerName,
		Force:               opts.force,
		CWD:                 cwd,
	}

	for _, raw := range opts.dataValues {
		if err := app.ParseDataFlag(&cfg, raw, app.DataSourceValue); err != nil {
			return err
		}
	}
	for _, raw := range opts.dataFiles {
		if err := app.ParseDataFlag(&cfg, raw, app.DataSourceFile); err != nil {
			return err
		}
	}
	for _, raw := range opts.targets {
		if err := app.ParseTargetFlag(&cfg, raw); err != nil {
			return err
		}
	}

	if nonInteractive {
		if app.TrimSpace(cfg.SecretName) == "" {
			return errors.New("--name is required in non-interactive mode")
		}
		if err := app.ValidateAndPrepare(&cfg); err != nil {
			return err
		}
		if !cfg.Force {
			if existing := app.ExistingOutputFiles(cfg.Targets); len(existing) > 0 {
				return fmt.Errorf("output file already exists: %s (use --force to overwrite)", existing[0])
			}
		}
		return app.SealAndWriteAll(ctx, cfg, runner, cmd.OutOrStdout())
	}

	if cfg.Force {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: --force is only used in non-interactive mode and will be ignored.")
	}

	prompt := app.NewPrompt(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), cwd)
	if err := prompt.Run(&cfg); err != nil {
		return err
	}
	return app.SealAndWriteAll(ctx, cfg, runner, cmd.OutOrStdout())
}
