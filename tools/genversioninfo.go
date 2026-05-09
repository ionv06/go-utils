//go:build ignore

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func git(args ...string) string {

	out, err := exec.Command("git", args...).Output()

	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func atoi(s string) int {

	n, _ := strconv.Atoi(s)

	return n
}

func Genverioninfo() {

	ver := git("describe", "--tags", "--abbrev=0")
	commit_id := git("rev-parse", "--short", "HEAD")

	if ver == "" {
		ver = "v0.0.0"
	}
	if commit_id == "" {
		commit_id = "abcd"
	}
	ver = strings.TrimPrefix(ver, "v")

	parts := strings.Split(ver, ".")

	major := 0
	minor := 0
	patch := 0

	if len(parts) > 0 {
		major = atoi(parts[0])
	}

	if len(parts) > 1 {
		minor = atoi(parts[1])
	}

	if len(parts) > 2 {
		patch = atoi(parts[2])
	}

	data := map[string]interface{}{
		"FixedFileInfo": map[string]interface{}{
			"FileVersion": map[string]interface{}{
				"Major": major,
				"Minor": minor,
				"Patch": patch,
				"Build": 0,
			},
			"ProductVersion": map[string]interface{}{
				"Major": major,
				"Minor": minor,
				"Patch": patch,
				"Build": 0,
			},
		},
		"StringFileInfo": map[string]interface{}{
			"Comments":         commit_id,
			"CompanyName":      "Cometa SRL",
			"FileDescription":  "ContabSQL Tools",
			"FileVersion":      ver,
			"InternalName":     "contab_tools",
			"OriginalFilename": "saft-xml2xlsx.exe",
			"ProductName":      "ERP Tools",
			"ProductVersion":   ver,
		},
	}

	f, err := os.Create("versioninfo.json")

	if err != nil {
		panic(err)
	}

	defer f.Close()

	enc := json.NewEncoder(f)

	enc.SetIndent("", "  ")

	err = enc.Encode(data)

	if err != nil {
		panic(err)
	}
}
