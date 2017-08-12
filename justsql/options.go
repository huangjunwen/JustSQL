package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	"os"
	"path/filepath"
	"strings"
)

// MutipleValues represents a list of options: "-op x -op y -op z ..."
type MutipleValues []string

// String implements flag/Value interface.
func (m *MutipleValues) String() string {
	if m == nil {
		return ""
	}
	return strings.Join([]string(*m), ";")
}

// Set implements flag/Value interface.
func (m *MutipleValues) Set(val string) error {
	*m = append(*m, val)
	return nil
}

// MarshalJSON implements json/Marshaler interface.
func (m *MutipleValues) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(*m))
}

// MarshalJSON implements json/Marshaler interface.
func (m *MutipleValues) UnmarshalJSON(data []byte) error {
	res := []string{}
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

type Options struct {
	OutputDir string        `json:"output_dir"` // Output directory.
	LogLevel  string        `json:"log_level"`  // Log level (fatal/error/warn/info/debug).
	DDL       MutipleValues `json:"ddl"`        // DDL files.
	DML       MutipleValues `json:"dml"`        // DML files.
	NoFormat  bool          `json:"no_format"`  // Do not go format output files.
	Template  MutipleValues `json:"template"`   // Custom template directory.
}

func ParseOptions() *Options {

	printUsageAndExit := func(withErr error) {
		if withErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n\nUsage:\n", withErr)
		}
		flag.PrintDefaults()
		os.Exit(255)
	}

	checkDir := func(path string) string {
		absPath, err := filepath.Abs(path)
		if err != nil {
			printUsageAndExit(err)
		}
		fi, err := os.Stat(absPath)
		if err != nil {
			printUsageAndExit(err)
		}
		if !fi.IsDir() {
			printUsageAndExit(fmt.Errorf("%+q is not a directory.", absPath))
		}
		return absPath
	}

	// Parse options in command line.
	var configFile string
	var help bool
	options := &Options{}
	flag.StringVar(&configFile, "conf", "", "Configure file in JSON format. If omitted, justsql will try to find 'justsql.json' in current dir.")
	flag.BoolVar(&help, "h", false, "Print help.")
	flag.StringVar(&options.OutputDir, "output_dir", "", "Output directory for generated files.")
	flag.StringVar(&options.LogLevel, "log_level", "", "Log level: fatal/error/warn/info/debug")
	flag.Var(&options.DDL, "ddl", "Glob of DDL files (file containing DDL SQL). Multiple \"-ddl\" is allowed.")
	flag.Var(&options.DML, "dml", "Glob of DML files (file containing DML SQL). Multiple \"-ddl\" is allowed.")
	flag.BoolVar(&options.NoFormat, "no_format", false, "Do not go format output files.")
	flag.Var(&options.Template, "template", "Custom templates directory.")
	flag.Parse()

	if help {
		printUsageAndExit(nil)
	}

	// Try to parse options in configure file (default to ./justsql.json).
	explicitConfigFile := false
	if configFile != "" {
		explicitConfigFile = true
	} else {
		configFile = "./justsql.json"
	}
	configFile, err := filepath.Abs(configFile)
	if err != nil {
		printUsageAndExit(err)
	}
	if err = os.Chdir(filepath.Dir(configFile)); err != nil {
		printUsageAndExit(err)
	}
	f, err := os.Open(filepath.Base(configFile))
	if err == nil {
		defer f.Close()
		configOptions := &Options{}
		err := json.NewDecoder(f).Decode(configOptions)
		if err != nil {
			printUsageAndExit(err)
		}
		// Merge two options. Command line override config file.
		if options.OutputDir == "" && configOptions.OutputDir != "" {
			options.OutputDir = configOptions.OutputDir
		}
		if options.LogLevel == "" && configOptions.LogLevel != "" {
			options.LogLevel = configOptions.LogLevel
		}
		options.DDL = append(configOptions.DDL, options.DDL...)
		options.DML = append(configOptions.DML, options.DML...)
		if configOptions.NoFormat {
			options.NoFormat = true
		}
		options.Template = append(configOptions.Template, options.Template...)
	} else {
		// Yield error only when config file is explicit.
		if explicitConfigFile {
			printUsageAndExit(err)
		}
	}

	// Checks and set default value.
	switch options.LogLevel {
	case "fatal", "error", "warn", "info", "debug":
	case "":
		options.LogLevel = "error"
	default:
		printUsageAndExit(fmt.Errorf("Unknown log level %+q", options.LogLevel))
	}

	if options.OutputDir == "" {
		printUsageAndExit(fmt.Errorf("Missing -output_dir"))
	}
	absOutputDir := checkDir(options.OutputDir)
	base := filepath.Base(absOutputDir)
	if !utils.IsIdent(base) {
		printUsageAndExit(fmt.Errorf("%+q is not a good package name.", base))
	}
	options.OutputDir = absOutputDir

	absTempateDirs := []string{}
	for _, templateDir := range options.Template {
		if templateDir == "" {
			continue
		}
		absTempateDirs = append(absTempateDirs, checkDir(templateDir))
	}
	options.Template = MutipleValues(absTempateDirs)

	return options
}
