package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raphael/goa-swagger/app"
)

// SpecController implements the spec resource.
type SpecController struct{}

// NewSpecController creates a spec controller.
func NewSpecController() *SpecController {
	return &SpecController{}
}

// Show runs the show action.
func (c *SpecController) Show(ctx *app.ShowSpecContext) error {
	tmpGoPath, err := ioutil.TempDir("", "goa-swagger-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpGoPath)
	packagePath := strings.TrimPrefix(ctx.PackagePath, "/")
	getCmd := exec.Command("go", "get", "-d", packagePath)
	getCmd.Env = []string{"GOPATH=" + tmpGoPath, "PATH=" + os.Getenv("PATH")}
	out, err := getCmd.CombinedOutput()
	if err != nil {
		return ctx.UnprocessableEntity(out)
	}
	sha := extractSHA(filepath.Join(tmpGoPath, "src", packagePath))
	if sha != "" {
		if b, ok := Load(packagePath, sha); ok {
			return ctx.OK(b)
		}
	}
	genCmd := exec.Command("goagen", "-o", tmpGoPath, "swagger", "-d", packagePath)
	genCmd.Env = []string{
		fmt.Sprintf("GOPATH=%s:%s", tmpGoPath, os.Getenv("GOPATH")),
		"PATH=" + os.Getenv("PATH"),
	}
	out, err = genCmd.CombinedOutput()
	if err != nil {
		return ctx.UnprocessableEntity(out)
	}
	b, err := ioutil.ReadFile(filepath.Join(tmpGoPath, "swagger", "swagger.json"))
	if err != nil {
		return ctx.UnprocessableEntity([]byte(err.Error()))
	}
	if sha != "" {
		Save(b, packagePath, sha)
	}
	return ctx.OK(b)
}

func extractSHA(vcsDir string) string {
	gitSHA := filepath.Join(vcsDir, ".git/refs/heads/go1")
	if _, err := os.Stat(gitSHA); err == nil {
		if b, err := ioutil.ReadFile(gitSHA); err == nil {
			return string(b)
		}
	}
	gitSHA = filepath.Join(vcsDir, ".git/refs/heads/master")
	if _, err := os.Stat(gitSHA); err == nil {
		if b, err := ioutil.ReadFile(gitSHA); err == nil {
			return string(b)
		}
	}
	// TBD: handle other vcs
	return ""
}
