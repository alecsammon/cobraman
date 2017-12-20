// Copyright 2015 Red Hat Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package man

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const defaultManTemplate = `.TH "{{.ProgramName}}" "{{ .Section }}" "{{.CenterFooter}}" "{{.LeftFooter}}" "{{.CenterHeader}}" 
.nh
.ad l
.SH NAME
.PP
zap\-publish \- Publish into MQTT
.SH SYNOPSIS
.PP
.B {{ .CommandPath }}
{{ .SynFlags }}
.SH DESCRIPTION
.PP
{{ .Description }}{{ if .HasFlags }}
.SH OPTIONS
{{ .Flags }}{{ end }}{{ if .HasInheritedFlags }}
.SH OPTIONS INHERITED FROM PARENT COMMANDS
{{ .InheritedFlags }}{{ end }}{{ if .HasEnvironment }}
.SH Environment
.PP
{{ .Environment }}{{ end }}{{ if .HasFiles }}
.SH FILES
.PP
{{ .Files }}{{ end }}{{ if .HasBugs }}
.SH BUGS
.PP
{{ .Bugs }}{{ end }}{{ if .HasExamples }}
.SH EXAMPLES
.PP
{{ .Examples }}{{ end }}{{if .HasAuthor }}
.SH AUTHOR
.PP
{{.Author}}{{end}}{{if .HasSeeAlso }}
.SH SEE ALSO
{{ .SeeAlsos }}{{ end }}
." This file auto-generated by github.com/rjohnson/cobra-man
`

// GenerateManOptions is used configure how GenerateManPages will
// do its job.
type GenerateManOptions struct {
	// ProgramName is used in the man page header across all pages
	ProgramName string

	// What section to generate the pages 4 (1 is the default if not set)
	Section string

	// CenterFooter used across all pages (defaults to current month and year)
	// If you just want to set the date used in the center footer use Date
	CenterFooter string

	// If you just want to set the date used in the center footer use Date
	Date *time.Time

	// LeftFooter used across all pages
	LeftFooter string

	// CenterHeader used across all pages
	CenterHeader string

	// Files if set with content will create a FILES section for all
	// pages.  If you want this section only for a single command add
	// it as an annotation: cmd.Annotations["man-files-section"]
	// The field will be sanitized for troff output. However, if
	// it starts with a '.' we assume it is valid troff and pass it through.
	Files string

	// Bugs if set with content will create a BUGS section for all
	// pages.  If you want this section only for a single command add
	// it as an annotation: cmd.Annotations["man-files-section"]
	// The field will be sanitized for troff output. However, if
	// it starts with a '.' we assume it is valid troff and pass it through.
	Bugs string

	// Environment if set with content will create a ENVIRONMENT section for all
	// pages.  If you want this section only for a single command add
	// it as an annotation: cmd.Annotations["man-environment-section"]
	// The field will be sanitized for troff output. However, if
	// it starts with a '.' we assume it is valid troff and pass it through.
	Environment string

	// Author if set will create a Author section with this content.
	Author string

	// Directory location for where to generate the man pages
	Directory string

	// CommandSperator defines what character to use to separate the
	// sub commands in the man page file name.  The '-' char is the default.
	CommandSeparator string

	// GenSeprateInheiratedFlags will generate a separate section for
	// inherited flags.  By default they will all be in the same OPZTIONS
	// section.
	GenSeprateInheritedFlags bool

	// UseTemplate allows you to override the default go template used to
	// generate the man pages with your own version.
	UseTemplate string
}

// GenerateManPages - build man pages for the passed in cobra.Command
// and all of its children
func GenerateManPages(cmd *cobra.Command, opts *GenerateManOptions) error {
	if opts.ProgramName == "" {
		opts.ProgramName = cmd.CommandPath() // TODO: this can't be right default
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenerateManPages(c, opts); err != nil {
			return err
		}
	}
	section := "1"
	if opts.Section != "" {
		section = opts.Section
	}

	separator := "-"
	if opts.CommandSeparator != "" {
		separator = opts.CommandSeparator
	}
	basename := strings.Replace(cmd.CommandPath(), " ", separator, -1)
	filename := filepath.Join(opts.Directory, basename+"."+section)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return generateManPage(cmd, opts, f)
}

type manStruct struct {
	ProgramName  string
	Section      string
	CenterFooter string
	LeftFooter   string
	CenterHeader string

	Name        string
	UseLine     string
	CommandPath string
	Description string
	SynFlags    string

	HasFlags          bool
	Flags             string
	HasInheritedFlags bool
	InheritedFlags    string

	HasSeeAlso bool
	SeeAlsos   string

	HasAuthor      bool
	Author         string
	HasEnvironment bool
	Environment    string
	HasFiles       bool
	Files          string
	HasBugs        bool
	Bugs           string
	HasExamples    bool
	Examples       string
}

func generateManPage(cmd *cobra.Command, opts *GenerateManOptions, w io.Writer) error {
	var flags *pflag.FlagSet
	values := manStruct{}

	// Header fields
	values.ProgramName = opts.ProgramName
	values.LeftFooter = opts.LeftFooter
	values.CenterHeader = opts.CenterHeader

	values.Section = opts.Section
	if values.Section == "" {
		values.Section = "1"
	}

	date := opts.Date
	if opts.Date == nil {
		now := time.Now()
		date = &now
	}
	if opts.CenterFooter == "" {
		values.CenterFooter = date.Format("Jan 2006")
	} else {
		values.CenterFooter = opts.CenterFooter
	}

	// NAME
	dashCommandName := strings.Replace(cmd.CommandPath(), " ", "-", -1)
	values.Name = fmt.Sprintf("%s \\- %s\n", dashCommandName, backslashify(cmd.Short))
	flags = cmd.Flags()
	if flags.HasFlags() {
		buf := new(bytes.Buffer)
		printSynFlags(buf, flags)
		values.SynFlags = buf.String()
	}

	// SYNOPSIS
	values.UseLine = cmd.UseLine()
	values.CommandPath = cmd.CommandPath()

	// DESCRIPTION
	description := cmd.Long
	if len(description) == 0 {
		description = cmd.Short
	}
	values.Description = description

	// Options
	if opts.GenSeprateInheritedFlags {
		flags = cmd.NonInheritedFlags()
	} else {
		flags = cmd.Flags()
	}
	if flags.HasFlags() {
		values.HasFlags = true
		buf := new(bytes.Buffer)
		printFlags(buf, flags)
		values.Flags = buf.String()
	}
	if opts.GenSeprateInheritedFlags {
		flags = cmd.NonInheritedFlags()
		values.HasInheritedFlags = true
		buf := new(bytes.Buffer)
		printFlags(buf, flags)
		values.InheritedFlags = buf.String()
	}

	// ENVIRONMENT section
	if opts.Environment != "" || cmd.Annotations["man-environment-section"] != "" {
		values.HasEnvironment = true
		if cmd.Annotations["man-environment-section"] != "" {
			values.Environment = simpleToTroff(cmd.Annotations["man-environment-section"])
		} else {
			values.Environment = simpleToTroff(opts.Environment)
		}
	}

	// FILES section
	if opts.Files != "" || cmd.Annotations["man-files-section"] != "" {
		values.HasFiles = true
		if cmd.Annotations["man-files-section"] != "" {
			values.Files = simpleToTroff(cmd.Annotations["man-files-section"])
		} else {
			values.Files = simpleToTroff(opts.Files)
		}
	}

	// BUGS section
	if opts.Bugs != "" || cmd.Annotations["man-bugs-section"] != "" {
		values.HasBugs = true
		if cmd.Annotations["man-bugs-section"] != "" {
			values.Bugs = simpleToTroff(cmd.Annotations["man-bugs-section"])
		} else {
			values.Bugs = simpleToTroff(opts.Bugs)
		}
	}

	// EXAMPLES section
	if cmd.Example != "" || cmd.Annotations["man-examples-section"] != "" {
		values.HasExamples = true
		if cmd.Annotations["man-examples-section"] != "" {
			values.Bugs = simpleToTroff(cmd.Annotations["man-examples-section"])
		} else {
			values.Bugs = simpleToTroff(cmd.Example)
		}
	}

	// AUTHOR section
	if opts.Author != "" {
		values.HasAuthor = true
		values.Author = opts.Author + "\n.PP\n.SM Page auto-generated by rjohnson/cobra-man and spf13/cobra"
	}

	// SEE ALSO section
	values.HasSeeAlso, values.SeeAlsos = generateSeeAlso(cmd, values.Section)

	// Build the template and generate the man page
	manTemplateStr := defaultManTemplate
	if opts.UseTemplate != "" {
		manTemplateStr = opts.UseTemplate
	}
	parsedTemplate, err := template.New("man").Parse(manTemplateStr)
	if err != nil {
		return err
	}
	err = parsedTemplate.Execute(w, values)
	if err != nil {
		return err
	}
	return nil
}

func printSynFlags(buf *bytes.Buffer, flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		if len(flag.Deprecated) > 0 || flag.Hidden {
			return
		}
		if len(flag.Shorthand) > 0 && len(flag.ShorthandDeprecated) == 0 {
			buf.WriteString(fmt.Sprintf(".RB [ \\-%s ]\n", flag.Shorthand))
		} else {
			buf.WriteString(fmt.Sprintf(".RB [ \\-\\-%s ]\n", backslashify(flag.Name)))
		}
	})
}

func printFlags(buf *bytes.Buffer, flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		if len(flag.Deprecated) > 0 || flag.Hidden {
			return
		}
		format := ".TP\n"
		if len(flag.Shorthand) > 0 && len(flag.ShorthandDeprecated) == 0 {
			format += fmt.Sprintf("\\fB\\-%s\\fP, \\fB\\-\\-%s\\fP", flag.Shorthand, backslashify(flag.Name))
		} else {
			format += fmt.Sprintf("\\fB\\-\\-%s\\fP", backslashify(flag.Name))
		}
		if len(flag.NoOptDefVal) > 0 {
			format += "["
		}
		format += "=\\fI%s\\fR"
		if len(flag.NoOptDefVal) > 0 {
			format += "]"
		}
		format += "\n%s\n"
		str := fmt.Sprintf(format, backslashify(flag.DefValue), backslashify(flag.Usage))
		buf.WriteString(strings.TrimRight(str, " \n"))
	})
}

func generateSeeAlso(cmd *cobra.Command, section string) (bool, string) {
	var hasSeeAlso bool

	seealsos := make([]string, 0)
	if cmd.HasParent() {
		hasSeeAlso = true
		parentPath := cmd.Parent().CommandPath()
		dashParentPath := strings.Replace(parentPath, " ", "\\-", -1)
		seealso := fmt.Sprintf(".BR %s (%s)", dashParentPath, section)
		seealsos = append(seealsos, seealso)
		// TODO: may want to control if siblings are shown or not
		siblings := cmd.Parent().Commands()
		sort.Sort(byName(siblings))
		for _, c := range siblings {
			if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() || c.Name() == cmd.Name() {
				continue
			}
			seealso := fmt.Sprintf(".BR %s\\-%s (%s)", dashParentPath, c.Name(), section)
			seealsos = append(seealsos, seealso)
		}
	}
	commandPath := cmd.CommandPath()
	dashCommandName := strings.Replace(commandPath, " ", "\\-", -1)
	children := cmd.Commands()
	sort.Sort(byName(children))
	for _, c := range children {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		hasSeeAlso = true
		seealso := fmt.Sprintf(".BR %s\\-%s (%s)", dashCommandName, c.Name(), section)
		seealsos = append(seealsos, seealso)
	}

	return hasSeeAlso, strings.Join(seealsos, ",\n")
}
