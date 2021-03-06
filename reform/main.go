// Package reform implements reform command.
package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xaionaro/reform"
	"github.com/xaionaro/reform/internal"
	"github.com/xaionaro/reform/parse"
)

var (
	logger *internal.Logger

	debugF = flag.Bool("debug", false, "Enable debug logging")
	gofmtF = flag.Bool("gofmt", true, "Format with gofmt")
)

func processFile(path, file, pack string) error {
	logger.Debugf("processFile: path=%q file=%q pack=%q", path, file, pack)

	srcFilePath := filepath.Join(path, file)

	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	outFileName := base + "_reform" + ext

	outFileInfo, outFileInfoErr := os.Stat(outFileName)
	if outFileInfoErr == nil {
		srcFileInfo, srcFileInfoErr := os.Stat(srcFilePath)
		if srcFileInfoErr != nil {
			return srcFileInfoErr
		}
		if outFileInfo.ModTime().UnixNano() > srcFileInfo.ModTime().UnixNano() {
			logger.Debugf("source file \"%v\" is not modified, skipping.", file)
		}
	}

	structs, err := parse.File(srcFilePath)
	if err != nil {
		return err
	}

	logger.Debugf("%#v", structs)
	if len(structs) == 0 {
		return nil
	}

	f, err := os.Create(filepath.Join(path, outFileName))
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString("package " + pack + "\n"); err != nil {
		return err
	}
	if err = prologTemplate.Execute(f, nil); err != nil {
		return err
	}

	sds := make([]StructData, 0, len(structs))
	for _, str := range structs {
		// decide about view/table suffix
		t := strings.ToLower(str.Type[0:1]) + str.Type[1:]
		v := str.Type
		s := str.Type
		if str.IsTable() {
			t += "TableType"
			v += "Table"
		} else {
			t += "ViewType"
			v += "View"
		}
		t += "Type"
		s += "Scope"

		capitalizedModelName := strings.ToUpper(str.Type[0:1]) + str.Type[1:]
		isPrivateStruct := str.Type[0:1] == strings.ToLower(str.Type[0:1])

		var querierVar string
		if isPrivateStruct {
			querierVar = strings.ToUpper(str.Type[0:1]) + str.Type[1:]
		} else {
			querierVar = str.Type + "SQL"
		}

		sd := StructData{
			LogType:             str.Type + "LogRow",
			StructInfo:          str,
			TableType:           t,
			LogTableType:        t + "_log",
			ScopeType:           s,
			FilterType:          capitalizedModelName + "Filter",
			FilterPublicType:    capitalizedModelName + "Type",
			FilterShorthandType: capitalizedModelName + "F",
			TableVar:            v,
			LogTableVar:         v + "LogRow",
			IsPrivateStruct:     isPrivateStruct,
			QuerierVar:          querierVar,
			ImitateGorm:         str.ImitateGorm,
			SkipMethodOrder:     str.SkipMethodOrder,
		}
		sds = append(sds, sd)

		if err = structTemplate.Execute(f, &sd); err != nil {
			return err
		}
	}

	return initTemplate.Execute(f, sds)
}

func gofmt(path string) {
	if *gofmtF {
		cmd := exec.Command("gofmt", "-s", "-w", path)
		logger.Debugf(strings.Join(cmd.Args, " "))
		b, err := cmd.CombinedOutput()
		if err != nil {
			logger.Fatalf("gofmt error: %s", err)
		}
		logger.Debugf("gofmt output: %s", b)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "reform - a better ORM generator. %s.\n\n", reform.Version)
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "  %s [flags] [packages or directories]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  go generate [flags] [packages or files] (with '//go:generate reform' in files)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	logger = internal.NewLogger("reform: ", *debugF)

	logger.Debugf("Environment:")
	for _, pair := range os.Environ() {
		if strings.HasPrefix(pair, "GO") {
			logger.Debugf("\t%s", pair)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		logger.Fatalf("%s", err)
	}
	logger.Debugf("wd: %s", wd)
	logger.Debugf("args: %v", flag.Args())

	// process arguments
	for _, arg := range flag.Args() {
		// import arg as directory or package path
		var pack *build.Package
		s, err := os.Stat(arg)
		if err == nil && s.IsDir() {
			pack, err = build.ImportDir(arg, 0)
		}
		if os.IsNotExist(err) {
			err = nil
		}
		if pack == nil && err == nil {
			pack, err = build.Import(arg, wd, 0)
		}
		if err != nil {
			logger.Fatalf("%s: %s", arg, err)
		}

		logger.Debugf("%#v", pack)

		var changed bool
		for _, f := range pack.GoFiles {
			err = processFile(pack.Dir, f, pack.Name)
			if err != nil {
				logger.Fatalf("%s %s: %s", arg, f, err)
			}
			changed = true
		}

		if changed {
			gofmt(pack.Dir)
		}
	}

	// process go generate environment
	file := os.Getenv("GOFILE")
	pack := os.Getenv("GOPACKAGE")
	if file != "" && pack != "" {
		err := processFile(wd, file, pack)
		if err != nil {
			logger.Fatalf("%s", err)
		}
		gofmt(wd)
	}
}
