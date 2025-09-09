// +build ignore

package main

import (
	"context"
	_flag "flag"
	_fmt "fmt"
	_ioutil "io/ioutil"
	_log "log"
	"os"
	"os/signal"
	_filepath "path/filepath"
	_sort "sort"
	"strconv"
	_strings "strings"
	"syscall"
	_tabwriter "text/tabwriter"
	"time"
	
)

func main() {
	// Use local types and functions in order to avoid name conflicts with additional magefiles.
	type arguments struct {
		Verbose       bool          // print out log statements
		List          bool          // print out a list of targets
		Help          bool          // print out help for a specific target
		Timeout       time.Duration // set a timeout to running the targets
		Args          []string      // args contain the non-flag command-line arguments
	}

	parseBool := func(env string) bool {
		val := os.Getenv(env)
		if val == "" {
			return false
		}		
		b, err := strconv.ParseBool(val)
		if err != nil {
			_log.Printf("warning: environment variable %s is not a valid bool value: %v", env, val)
			return false
		}
		return b
	}

	parseDuration := func(env string) time.Duration {
		val := os.Getenv(env)
		if val == "" {
			return 0
		}		
		d, err := time.ParseDuration(val)
		if err != nil {
			_log.Printf("warning: environment variable %s is not a valid duration value: %v", env, val)
			return 0
		}
		return d
	}
	args := arguments{}
	fs := _flag.FlagSet{}
	fs.SetOutput(os.Stdout)

	// default flag set with ExitOnError and auto generated PrintDefaults should be sufficient
	fs.BoolVar(&args.Verbose, "v", parseBool("MAGEFILE_VERBOSE"), "show verbose output when running targets")
	fs.BoolVar(&args.List, "l", parseBool("MAGEFILE_LIST"), "list targets for this binary")
	fs.BoolVar(&args.Help, "h", parseBool("MAGEFILE_HELP"), "print out help for a specific target")
	fs.DurationVar(&args.Timeout, "t", parseDuration("MAGEFILE_TIMEOUT"), "timeout in duration parsable format (e.g. 5m30s)")
	fs.Usage = func() {
		_fmt.Fprintf(os.Stdout, `
%s [options] [target]

Commands:
  -l    list targets in this binary
  -h    show this help

Options:
  -h    show description of a target
  -t <string>
        timeout in duration parsable format (e.g. 5m30s)
  -v    show verbose output when running targets
 `[1:], _filepath.Base(os.Args[0]))
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		// flag will have printed out an error already.
		return
	}
	args.Args = fs.Args()
	if args.Help && len(args.Args) == 0 {
		fs.Usage()
		return
	}
		
	// color is ANSI color type
	type color int

	// If you add/change/remove any items in this constant,
	// you will need to run "stringer -type=color" in this directory again.
	// NOTE: Please keep the list in an alphabetical order.
	const (
		black color = iota
		red
		green
		yellow
		blue
		magenta
		cyan
		white
		brightblack
		brightred
		brightgreen
		brightyellow
		brightblue
		brightmagenta
		brightcyan
		brightwhite
	)

	// AnsiColor are ANSI color codes for supported terminal colors.
	var ansiColor = map[color]string{
		black:         "\u001b[30m",
		red:           "\u001b[31m",
		green:         "\u001b[32m",
		yellow:        "\u001b[33m",
		blue:          "\u001b[34m",
		magenta:       "\u001b[35m",
		cyan:          "\u001b[36m",
		white:         "\u001b[37m",
		brightblack:   "\u001b[30;1m",
		brightred:     "\u001b[31;1m",
		brightgreen:   "\u001b[32;1m",
		brightyellow:  "\u001b[33;1m",
		brightblue:    "\u001b[34;1m",
		brightmagenta: "\u001b[35;1m",
		brightcyan:    "\u001b[36;1m",
		brightwhite:   "\u001b[37;1m",
	}
	
	const _color_name = "blackredgreenyellowbluemagentacyanwhitebrightblackbrightredbrightgreenbrightyellowbrightbluebrightmagentabrightcyanbrightwhite"

	var _color_index = [...]uint8{0, 5, 8, 13, 19, 23, 30, 34, 39, 50, 59, 70, 82, 92, 105, 115, 126}

	colorToLowerString := func (i color) string {
		if i < 0 || i >= color(len(_color_index)-1) {
			return "color(" + strconv.FormatInt(int64(i), 10) + ")"
		}
		return _color_name[_color_index[i]:_color_index[i+1]]
	}

	// ansiColorReset is an ANSI color code to reset the terminal color.
	const ansiColorReset = "\033[0m"

	// defaultTargetAnsiColor is a default ANSI color for colorizing targets.
	// It is set to Cyan as an arbitrary color, because it has a neutral meaning
	var defaultTargetAnsiColor = ansiColor[cyan]

	getAnsiColor := func(color string) (string, bool) {
		colorLower := _strings.ToLower(color)
		for k, v := range ansiColor {
			colorConstLower := colorToLowerString(k)
			if colorConstLower == colorLower {
				return v, true
			}
		}
		return "", false
	}

	// Terminals which  don't support color:
	// 	TERM=vt100
	// 	TERM=cygwin
	// 	TERM=xterm-mono
    var noColorTerms = map[string]bool{
		"vt100":      false,
		"cygwin":     false,
		"xterm-mono": false,
	}

	// terminalSupportsColor checks if the current console supports color output
	//
	// Supported:
	// 	linux, mac, or windows's ConEmu, Cmder, putty, git-bash.exe, pwsh.exe
	// Not supported:
	// 	windows cmd.exe, powerShell.exe
	terminalSupportsColor := func() bool {
		envTerm := os.Getenv("TERM")
		if _, ok := noColorTerms[envTerm]; ok {
			return false
		}
		return true
	}

	// enableColor reports whether the user has requested to enable a color output.
	enableColor := func() bool {
		b, _ := strconv.ParseBool(os.Getenv("MAGEFILE_ENABLE_COLOR"))
		return b
	}

	// targetColor returns the ANSI color which should be used to colorize targets.
	targetColor := func() string {
		s, exists := os.LookupEnv("MAGEFILE_TARGET_COLOR")
		if exists == true {
			if c, ok := getAnsiColor(s); ok == true {
				return c
			}
		}
		return defaultTargetAnsiColor
	}

	// store the color terminal variables, so that the detection isn't repeated for each target
	var enableColorValue = enableColor() && terminalSupportsColor()
	var targetColorValue = targetColor()

	printName := func(str string) string {
		if enableColorValue {
			return _fmt.Sprintf("%s%s%s", targetColorValue, str, ansiColorReset)
		} else {
			return str
		}
	}

	list := func() error {
		
		targets := map[string]string{
			"allProtobuf": "Generate code for all languages (Go, Java, C#, JS, TS) from protobuf files.",
			"genCSharp": "Generate C# code from protobuf files.",
			"genDocs": "",
			"genGo": "Generate Go code from protobuf files.",
			"genHarmonyTS": "Generate Harmony JavaScript code from protobuf files.",
			"genJava": "Generate Java code from protobuf files.",
			"genJavaScript": "",
			"genKotlin": "Generate Kotlin code from protobuf files.",
			"genSwift": "Generate Swift code from protobuf files.",
			"genTypeScript": "Need to install `ts-proto`.",
			"installDepend*": "install proto plugin",
			"meeting:genCSharp": "Generate C# code from protobuf files.",
			"meeting:genGo": "",
			"meeting:genJava": "",
			"meeting:genJavaScript": "",
			"meeting:genKotlin": "",
			"meeting:genSwift": "Generate Swift code from protobuf files.",
			"meeting:genTypeScript": "Generate TypeScript code from protobuf files.",
		}

		keys := make([]string, 0, len(targets))
		for name := range targets {
			keys = append(keys, name)
		}
		_sort.Strings(keys)

		_fmt.Println("Targets:")
		w := _tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
		for _, name := range keys {
			_fmt.Fprintf(w, "  %v\t%v\n", printName(name), targets[name])
		}
		err := w.Flush()
			if err == nil {
				_fmt.Println("\n* default target")
			}
		return err
	}

	var ctx context.Context
	ctxCancel := func(){}

	// by deferring in a closure, we let the cancel function get replaced
	// by the getContext function.
	defer func() {
		ctxCancel()
	}()

	getContext := func() (context.Context, func()) {
		if ctx == nil {
			if args.Timeout != 0 {
				ctx, ctxCancel = context.WithTimeout(context.Background(), args.Timeout)
			} else {
				ctx, ctxCancel = context.WithCancel(context.Background())
			}
		}

		return ctx, ctxCancel
	}

	runTarget := func(logger *_log.Logger, fn func(context.Context) error) interface{} {
		var err interface{}
		ctx, cancel := getContext()
		d := make(chan interface{})
		go func() {
			defer func() {
				err := recover()
				d <- err
			}()
			err := fn(ctx)
			d <- err
		}()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT)
		select {
		case <-sigCh:
			logger.Println("cancelling mage targets, waiting up to 5 seconds for cleanup...")
			cancel()
			cleanupCh := time.After(5 * time.Second)

			select {
			// target exited by itself
			case err = <-d:
				return err
			// cleanup timeout exceeded
			case <-cleanupCh:
				return _fmt.Errorf("cleanup timeout exceeded")
			// second SIGINT received
			case <-sigCh:
				logger.Println("exiting mage")
				return _fmt.Errorf("exit forced")
			}
		case <-ctx.Done():
			cancel()
			e := ctx.Err()
			_fmt.Printf("ctx err: %v\n", e)
			return e
		case err = <-d:
			// we intentionally don't cancel the context here, because
			// the next target will need to run with the same context.
			return err
		}
	}
	// This is necessary in case there aren't any targets, to avoid an unused
	// variable error.
	_ = runTarget

	handleError := func(logger *_log.Logger, err interface{}) {
		if err != nil {
			logger.Printf("Error: %+v\n", err)
			type code interface {
				ExitStatus() int
			}
			if c, ok := err.(code); ok {
				os.Exit(c.ExitStatus())
			}
			os.Exit(1)
		}
	}
	_ = handleError

	// Set MAGEFILE_VERBOSE so mg.Verbose() reflects the flag value.
	if args.Verbose {
		os.Setenv("MAGEFILE_VERBOSE", "1")
	} else {
		os.Setenv("MAGEFILE_VERBOSE", "0")
	}

	_log.SetFlags(0)
	if !args.Verbose {
		_log.SetOutput(_ioutil.Discard)
	}
	logger := _log.New(os.Stderr, "", 0)
	if args.List {
		if err := list(); err != nil {
			_log.Println(err)
			os.Exit(1)
		}
		return
	}

	if args.Help {
		if len(args.Args) < 1 {
			logger.Println("no target specified")
			os.Exit(2)
		}
		switch _strings.ToLower(args.Args[0]) {
			case "allprotobuf":
				_fmt.Println("Generate code for all languages (Go, Java, C#, JS, TS) from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage allprotobuf\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "gencsharp":
				_fmt.Println("Generate C# code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage gencsharp\n\n")
				var aliases []string
				aliases = append(aliases, "csharp")
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "gendocs":
				
				_fmt.Print("Usage:\n\n\tmage gendocs\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "gengo":
				_fmt.Println("Generate Go code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage gengo\n\n")
				var aliases []string
				
				
				aliases = append(aliases, "go")
				
				
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "genharmonyts":
				_fmt.Println("Generate Harmony JavaScript code from protobuf files. Note: please install pbjs and pbts command first Reference Link: https://ohpm.openharmony.cn/#/cn/detail/@ohos%2Fprotobufjs")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage genharmonyts\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "genjava":
				_fmt.Println("Generate Java code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage genjava\n\n")
				var aliases []string
				
				
				
				aliases = append(aliases, "java")
				
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "genjavascript":
				
				_fmt.Print("Usage:\n\n\tmage genjavascript\n\n")
				var aliases []string
				
				
				
				
				aliases = append(aliases, "js")
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "genkotlin":
				_fmt.Println("Generate Kotlin code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage genkotlin\n\n")
				var aliases []string
				
				
				
				
				
				aliases = append(aliases, "kotlin")
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "genswift":
				_fmt.Println("Generate Swift code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage genswift\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				
				
				
				aliases = append(aliases, "swift")
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "gentypescript":
				_fmt.Println("Need to install `ts-proto`. Generate TypeScript code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage gentypescript\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				aliases = append(aliases, "ts")
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "installdepend":
				_fmt.Println("install proto plugin")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage installdepend\n\n")
				var aliases []string
				
				aliases = append(aliases, "dep")
				
				
				
				
				
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "meeting:gencsharp":
				_fmt.Println("Generate C# code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage meeting:gencsharp\n\n")
				var aliases []string
				
				
				
				
				
				
				aliases = append(aliases, "m:csharp")
				
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "meeting:gengo":
				
				_fmt.Print("Usage:\n\n\tmage meeting:gengo\n\n")
				var aliases []string
				
				
				
				
				
				
				
				aliases = append(aliases, "m:go")
				
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "meeting:genjava":
				
				_fmt.Print("Usage:\n\n\tmage meeting:genjava\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				aliases = append(aliases, "m:java")
				
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "meeting:genjavascript":
				
				_fmt.Print("Usage:\n\n\tmage meeting:genjavascript\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				aliases = append(aliases, "m:js")
				
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "meeting:genkotlin":
				
				_fmt.Print("Usage:\n\n\tmage meeting:genkotlin\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				aliases = append(aliases, "m:kotlin")
				
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "meeting:genswift":
				_fmt.Println("Generate Swift code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage meeting:genswift\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				
				aliases = append(aliases, "m:swift")
				
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			case "meeting:gentypescript":
				_fmt.Println("Generate TypeScript code from protobuf files.")
				_fmt.Println()
				
				_fmt.Print("Usage:\n\n\tmage meeting:gentypescript\n\n")
				var aliases []string
				
				
				
				
				
				
				
				
				
				
				
				
				aliases = append(aliases, "m:ts")
				
				
				if len(aliases) > 0 {
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			default:
				logger.Printf("Unknown target: %q\n", args.Args[0])
				os.Exit(2)
		}
	}
	if len(args.Args) < 1 {
		ignoreDefault, _ := strconv.ParseBool(os.Getenv("MAGEFILE_IGNOREDEFAULT"))
		if ignoreDefault {
			if err := list(); err != nil {
				logger.Println("Error:", err)
				os.Exit(1)
			}
			return
		}
		
				wrapFn := func(ctx context.Context) error {
					return InstallDepend()
				}
				ret := runTarget(logger, wrapFn)
		handleError(logger, ret)
		return
	}
	for x := 0; x < len(args.Args); {
		target := args.Args[x]
		x++

		// resolve aliases
		switch _strings.ToLower(target) {
		
			case "csharp":
				target = "GenCSharp"
			case "dep":
				target = "InstallDepend"
			case "go":
				target = "GenGo"
			case "java":
				target = "GenJava"
			case "js":
				target = "GenJavaScript"
			case "kotlin":
				target = "GenKotlin"
			case "m:csharp":
				target = "Meeting:GenCSharp"
			case "m:go":
				target = "Meeting:GenGo"
			case "m:java":
				target = "Meeting:GenJava"
			case "m:js":
				target = "Meeting:GenJavaScript"
			case "m:kotlin":
				target = "Meeting:GenKotlin"
			case "m:swift":
				target = "Meeting:GenSwift"
			case "m:ts":
				target = "Meeting:GenTypeScript"
			case "swift":
				target = "GenSwift"
			case "ts":
				target = "GenTypeScript"
		}

		switch _strings.ToLower(target) {
		
			case "allprotobuf":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"AllProtobuf\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "AllProtobuf")
				}
				
				wrapFn := func(ctx context.Context) error {
					return AllProtobuf()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "gencsharp":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenCSharp\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenCSharp")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenCSharp()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "gendocs":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenDocs\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenDocs")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenDocs()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "gengo":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenGo\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenGo")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenGo()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "genharmonyts":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenHarmonyTS\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenHarmonyTS")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenHarmonyTS()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "genjava":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenJava\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenJava")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenJava()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "genjavascript":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenJavaScript\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenJavaScript")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenJavaScript()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "genkotlin":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenKotlin\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenKotlin")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenKotlin()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "genswift":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenSwift\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenSwift")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenSwift()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "gentypescript":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"GenTypeScript\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "GenTypeScript")
				}
				
				wrapFn := func(ctx context.Context) error {
					return GenTypeScript()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "installdepend":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"InstallDepend\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "InstallDepend")
				}
				
				wrapFn := func(ctx context.Context) error {
					return InstallDepend()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "meeting:gencsharp":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"Meeting:GenCSharp\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "Meeting:GenCSharp")
				}
				
				wrapFn := func(ctx context.Context) error {
					return Meeting{}.GenCSharp()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "meeting:gengo":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"Meeting:GenGo\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "Meeting:GenGo")
				}
				
				wrapFn := func(ctx context.Context) error {
					return Meeting{}.GenGo()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "meeting:genjava":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"Meeting:GenJava\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "Meeting:GenJava")
				}
				
				wrapFn := func(ctx context.Context) error {
					return Meeting{}.GenJava()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "meeting:genjavascript":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"Meeting:GenJavaScript\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "Meeting:GenJavaScript")
				}
				
				wrapFn := func(ctx context.Context) error {
					return Meeting{}.GenJavaScript()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "meeting:genkotlin":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"Meeting:GenKotlin\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "Meeting:GenKotlin")
				}
				
				wrapFn := func(ctx context.Context) error {
					return Meeting{}.GenKotlin()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "meeting:genswift":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"Meeting:GenSwift\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "Meeting:GenSwift")
				}
				
				wrapFn := func(ctx context.Context) error {
					return Meeting{}.GenSwift()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
			case "meeting:gentypescript":
				expected := x + 0
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"Meeting:GenTypeScript\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "Meeting:GenTypeScript")
				}
				
				wrapFn := func(ctx context.Context) error {
					return Meeting{}.GenTypeScript()
				}
				ret := runTarget(logger, wrapFn)
				handleError(logger, ret)
		
		default:
			logger.Printf("Unknown target specified: %q\n", target)
			os.Exit(2)
		}
	}
}




