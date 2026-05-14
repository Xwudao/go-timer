package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

const defaultInstallDir = "/usr/local/bin"

var (
	installDir string

	errAlreadyInstalled = errors.New("binary already installed at destination")
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Move the current binary into a system bin directory",
	Long:  "Moves the currently running timerd binary into a system bin directory so it can be invoked from PATH.",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		sourcePath, err := currentBinaryPath()
		if err != nil {
			return err
		}

		targetPath := filepath.Join(installDir, rootCmd.Name())

		ui.Info("Installing current binary")
		ui.Dim("  source : %s", sourcePath)
		ui.Dim("  target : %s", targetPath)

		if flagDryRun {
			ui.DryRunNotice()
			return nil
		}

		err = installBinary(sourcePath, targetPath)
		switch {
		case err == nil:
			ui.Success("installed to %s", targetPath)
			ui.Info("You can now run: %s version", rootCmd.Name())
			return nil
		case errors.Is(err, errAlreadyInstalled):
			ui.Info("already installed at %s", targetPath)
			return nil
		case errors.Is(err, os.ErrPermission):
			return fmt.Errorf("installing to %s: permission denied; re-run with sudo or use --dir", targetPath)
		default:
			return err
		}
	},
}

func init() {
	installCmd.Flags().StringVar(&installDir, "dir", defaultInstallDir, "Destination directory for the timerd binary")
	rootCmd.AddCommand(installCmd)
}

func currentBinaryPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving current executable: %w", err)
	}

	resolvedPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = resolvedPath
	}

	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return "", fmt.Errorf("normalising executable path %s: %w", exePath, err)
	}

	return absPath, nil
}

func installBinary(sourcePath, targetPath string) error {
	if samePath(sourcePath, targetPath) {
		return errAlreadyInstalled
	}

	if sameFile(sourcePath, targetPath) {
		return errAlreadyInstalled
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source binary %s: %w", sourcePath, err)
	}
	if sourceInfo.IsDir() {
		return fmt.Errorf("source path %s is a directory", sourcePath)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("creating install dir %s: %w", filepath.Dir(targetPath), err)
	}

	if err := moveFile(sourcePath, targetPath, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("moving binary to %s: %w", targetPath, err)
	}

	return nil
}

func moveFile(sourcePath, targetPath string, sourceMode os.FileMode) error {
	if err := os.Rename(sourcePath, targetPath); err == nil {
		return os.Chmod(targetPath, sourceMode|0o755)
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	if err := copyFile(sourcePath, targetPath, sourceMode|0o755); err != nil {
		return err
	}

	if err := os.Remove(sourcePath); err != nil {
		return err
	}

	return nil
}

func copyFile(sourcePath, targetPath string, mode os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	tempFile, err := os.CreateTemp(filepath.Dir(targetPath), filepath.Base(targetPath)+".*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(tempFile, sourceFile); err != nil {
		return err
	}

	if err := tempFile.Chmod(mode); err != nil {
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		return err
	}

	return nil
}

func samePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func sameFile(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr != nil || rightErr != nil {
		return false
	}

	return os.SameFile(leftInfo, rightInfo)
}
