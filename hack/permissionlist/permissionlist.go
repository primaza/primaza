/*
Copyright 2023 The Primaza Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This is a stand-alone program to generate a Go program with permission list struct.
// The same program is used for both application and service workers.
// This is program is invoked through go:generate (see main.go)

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Role struct {
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Rules []struct {
		APIGroups     []string `yaml:"apiGroups"`
		Resources     []string `yaml:"resources"`
		Verbs         []string `yaml:"verbs"`
		ResourceNames []string `yaml:"resourceNames"`
	} `yaml:"rules"`
}

func main() {
	role := Role{}
	agentType := os.Args[1]
	goProgramDir := os.Args[2]
	groupsTmpl := template.Must(template.New("").Parse(`
APIGroups: []string{{ "{" }}{{range $i, $v := .}}{{if $i}}, {{end}}"{{$v}}"{{end}}{{ "}" }},`))
	resourcesTmpl := template.Must(template.New("").Parse(`
Resources: []string{{ "{" }}{{range $i, $v := .}}{{if $i}}, {{end}}"{{$v}}"{{end}}{{ "}" }},`))
	namespacesTmpl := template.Must(template.New("").Parse(`Namespace: "{{ . }}",`))
	nameTmpl := template.Must(template.New("").Parse(`Name: "{{ . }}",`))
	verbsTmpl := template.Must(template.New("").Parse(`
Verbs: []string{{ "{" }}{{range $i, $v := .}}{{if $i}}, {{end}}"{{$v}}"{{end}}{{ "}" }},`))
	resourceNamesTmpl := template.Must(template.New("").Parse(`
ResourceNames: []string{{ "{" }}{{range $i, $v := .}}{{if $i}}, {{end}}"{{$v}}"{{end}}{{ "}" }},`))

	var out strings.Builder

	out.WriteString(fmt.Sprintf(`package authz

var %sPermissionList = []Permission{
`, agentType))
	for _, ymlFile := range os.Args[3:] {
		ymlBytes, _ := os.ReadFile(filepath.Clean(ymlFile))
		_ = yaml.Unmarshal(ymlBytes, &role)
		for _, r := range role.Rules {
			var err error
			out.WriteString("{")
			err = groupsTmpl.Execute(&out, r.APIGroups)
			if err != nil {
				log.Println(err)
			}
			err = resourcesTmpl.Execute(&out, r.Resources)
			if err != nil {
				log.Println(err)
			}
			err = resourceNamesTmpl.Execute(&out, r.ResourceNames)
			if err != nil {
				log.Println(err)
			}
			out.WriteString("\n")
			err = namespacesTmpl.Execute(&out, role.Metadata.Namespace)
			if err != nil {
				log.Println(err)
			}
			out.WriteString("\n")
			err = nameTmpl.Execute(&out, role.Metadata.Name)
			if err != nil {
				log.Println(err)
			}
			err = verbsTmpl.Execute(&out, r.Verbs)
			if err != nil {
				log.Println(err)
			}
			out.WriteString("\n},\n")
		}
	}

	out.WriteString("}\n")
	goProgram := filepath.Join(goProgramDir, fmt.Sprintf("permission_list_%s.go", strings.ToLower(agentType)))
	err := os.WriteFile(goProgram, []byte(out.String()), 0600)
	if err != nil {
		log.Fatal(err)
	}
}
