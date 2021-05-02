package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey"
	"gopkg.in/alecthomas/kingpin.v2"
)

var version string

type wizard struct {
	OutputFilename string
	Values         map[string]string
}

var (
	app = kingpin.New("texlate", "Generate an interactive wizard from text/template")
	createCommand = app.Command("create", "Template from scratch")
	updateCommand = app.Command("update", "Update a document from an existing values file")

	templateFile = createCommand.Arg("template", "source template file").Required().ExistingFile()
	valuesFile = updateCommand.Arg("values", "previous answers").Required().ExistingFile()
)

func main() {

	app.Version(version).VersionFlag.Short('V')
	app.HelpFlag.Short('h')
	app.UsageTemplate(kingpin.SeparateOptionalFlagsUsageTemplate)
	w := wizard{}
	w.Values = make(map[string]string)
	w.OutputFilename = ""
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case createCommand.FullCommand():
		w.Values["_template"] = *templateFile
	case updateCommand.FullCommand():
		data, _ := ioutil.ReadFile(*valuesFile)
		_ = json.Unmarshal(data, &w.Values)
		t := w.Values["_template"]
		templateFile = &t
	}

	tmpl, err := template.New(path.Base(*templateFile)).Delims("\\begin{template}", "\\end{template}").ParseFiles(*templateFile)

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, &w)
	if err != nil {
		log.Println(err)
	}

	if w.OutputFilename == "" {
		_, err := io.Copy(os.Stdout, &output)
		if err != nil {
			os.Exit(1)
		}
	} else {
		outputDir, _ := filepath.Abs(path.Join(".", filepath.Dir(w.OutputFilename)))
		_, err := os.Stat(outputDir)
		if os.IsNotExist(err) {
			os.MkdirAll(outputDir, 0755)
		}

		texFileAbsolutePath, _ := filepath.Abs(w.OutputFilename + ".tex")
		valuesFileAbsolutePath, _ := filepath.Abs(w.OutputFilename + ".json")
		outputFile, err := os.OpenFile(texFileAbsolutePath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		_, err = io.Copy(outputFile, &output)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		valueBytes, err := json.Marshal(w.Values)
		if err != nil {
			log.Println(err)
		}

		err = ioutil.WriteFile(valuesFileAbsolutePath, valueBytes, 0644)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}

		// Run pdflatex twice
		templateAbsolutePath, err := filepath.Abs(*templateFile)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		templateDir := filepath.Dir(templateAbsolutePath)

		cmd := exec.Command("pdflatex", "-halt-on-error", texFileAbsolutePath)
		cmd.Dir = outputDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("TEXINPUTS=$TEXMFDOTDIR:$TEXMF/tex/{$progname,generic,latex,}//:%s", templateDir))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		err = cmd.Run()
	}
}

func (w *wizard) SetOutputFilename(filename string) string {

	w.OutputFilename = filename
	return ""

}

//PromptBool asks the user a yes/no question and returns the answer as a bool
func (w *wizard) PromptBool(mapKey string, question string) bool {

	yesno, _ := strconv.ParseBool(w.Values[mapKey])
	prompt := &survey.Confirm{
		Message: question,
		Default: yesno,
	}

	err := survey.AskOne(prompt, &yesno, nil)
	if err != nil {
		panic(err)
	}
	w.Values[mapKey] = strconv.FormatBool(yesno)
	return yesno

}

//PromptString asks the user a question and returns the string the user enters
func (w *wizard) PromptString(mapKey string, question string) string {

	answer := w.Values[mapKey]
	prompt := &survey.Input{
		Message: question,
		Default: answer,
	}
	err := survey.AskOne(prompt, &answer, nil)
	if err != nil {
		panic(err)
	}
	w.Values[mapKey] = answer
	return TexEscape(answer)

}

//PromptSelect asks the user a question and gives a list of options for the user to select and answer
func (w *wizard) PromptSelect(mapKey string, question string, options ...string) string {

	answer := w.Values[mapKey]
	prompt := &survey.Select{
		Message: question,
		Options: options,
		Default: answer,
	}

	err := survey.AskOne(prompt, &answer, nil)
	if err != nil {
		panic(err)
	}
	w.Values[mapKey] = answer
	return TexEscape(answer)

}

// FormatDate formats the current time based on a provided date format
func (w wizard) FormatDate(format string) string {

	return time.Now().Format(format)

}

func TexEscape(input string) string {

	input = strings.ReplaceAll(input, "&", "\\&")
	input = strings.ReplaceAll(input, "%", "\\%")
	input = strings.ReplaceAll(input, "$", "\\$")
	input = strings.ReplaceAll(input, "#", "\\#")
	input = strings.ReplaceAll(input, "_", "\\_")
	input = strings.ReplaceAll(input, "{", "\\{")
	input = strings.ReplaceAll(input, "}", "\\}")
	input = strings.ReplaceAll(input, "~ ", "\\textasciitilde\\space ")
	input = strings.ReplaceAll(input, "~", "\\textasciitilde ")
	input = strings.ReplaceAll(input, "^ ", "\\textasciicircum\\space ")
	input = strings.ReplaceAll(input, "^", "\\textasciicircum ")
	input = strings.ReplaceAll(input, "\\ ", "\\textbackslash\\space ")
	input = strings.ReplaceAll(input, "\\", "\\textbackslash ")

	return input

}
