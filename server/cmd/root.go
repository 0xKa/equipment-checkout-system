package cmd

import "github.com/spf13/cobra"

func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "equipment-checkout",
		Short:         "Equipment Checkout API",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newInspectTokenCommand(),
		newServeCommand(),
	)
	return root
}
