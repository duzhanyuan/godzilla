package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/jingweno/godzilla/ast"
	"github.com/jingweno/godzilla/compiler"
	"github.com/jingweno/godzilla/source"
	"github.com/spf13/cobra"
)

var (
	parserPath          string
	buildJavaScriptFile string
	buildGoOutFile      string
)

func main() {
	rootCmd := &cobra.Command{Use: "godzilla"}
	cmdBuild := &cobra.Command{
		Use:   "build",
		Short: "compile JavaScript program",
		RunE:  runBuild,
	}
	cmdBuild.PersistentFlags().StringVarP(&buildJavaScriptFile, "js", "", "", "path to JavaScript file")
	cmdBuild.PersistentFlags().StringVarP(&buildGoOutFile, "output", "o", "", "output file")
	cmdRun := &cobra.Command{
		Use:   "run",
		Short: "compile and run JavaScript program",
		RunE:  runRun,
	}
	rootCmd.AddCommand(cmdBuild)
	rootCmd.AddCommand(cmdRun)
	rootCmd.PersistentFlags().StringVarP(&parserPath, "parser-path", "p", filepath.Join(filepath.Dir(os.Args[0]), "godzilla-parser"), "path to godzilla-parser")
	rootCmd.Execute()
}

func runBuild(cmd *cobra.Command, args []string) error {
	mainFile, err := compileMain()
	if err != nil {
		return err
	}

	return goBuild(mainFile, buildGoOutFile)
}

func runRun(cmd *cobra.Command, args []string) error {
	mainFile, err := compileMain()
	if err != nil {
		return err
	}

	return goRun(mainFile)
}

func compileMain() (string, error) {
	source, err := compileSource()
	if err != nil {
		return "", err
	}

	return writeMainFile(source)
}

func compileSource() (*source.Code, error) {
	c := exec.Command(parserPath)
	c.Stdin = os.Stdin
	stdoutStderr, err := c.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error parsing JavaScript %s: %s", err, stdoutStderr)
	}

	m := make(map[string]interface{})
	if err := json.NewDecoder(bytes.NewBuffer(stdoutStderr)).Decode(&m); err != nil {
		return nil, err
	}

	f := &ast.File{}
	f.UnmarshalMap(m)

	return compiler.Compile(f), nil
}

func writeMainFile(code *source.Code) (string, error) {
	mainDir, err := ioutil.TempDir("", "main")
	if err != nil {
		return "", err
	}

	mainFile, err := os.OpenFile(filepath.Join(mainDir, "main.go"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}

	if err := code.WriteTo(mainFile); err != nil {
		return "", err
	}

	if err := mainFile.Close(); err != nil {
		return "", err
	}

	return mainFile.Name(), nil
}

func goBuild(mainFile, outFile string) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return err
	}

	return syscall.Exec(goBin, []string{"go", "build", "-o", outFile, mainFile}, os.Environ())
}

func goRun(mainFile string) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return err
	}

	return syscall.Exec(goBin, []string{"go", "run", mainFile}, os.Environ())
}
