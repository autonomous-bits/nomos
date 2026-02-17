package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/autonomous-bits/nomos/libs/compiler/pkg/encryption"
	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage encryption keys",
	Long:  `Generate and manage keys for encrypting secrets in Nomos configurations.`,
}

var keysGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new encryption key",
	Long:  `Generate a new random AES-256 key and output it in base64 format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := encryption.GenerateKey()
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		encoded := base64.StdEncoding.EncodeToString(key)

		if keysOutput != "" {
			if err := os.WriteFile(keysOutput, []byte(encoded), 0600); err != nil {
				return fmt.Errorf("failed to write key file: %w", err)
			}
			if !globalFlags.quiet {
				fmt.Printf("Generated key saved to %s\n", keysOutput)
			}
			return nil
		}

		fmt.Println(encoded)
		return nil
	},
}

var keysOutput string

func init() {
	keysGenerateCmd.Flags().StringVarP(&keysOutput, "out", "o", "", "Output file path (default stdout)")
	keysCmd.AddCommand(keysGenerateCmd)
}
