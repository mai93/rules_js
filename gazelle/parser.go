package gazelle

import (
	"io/ioutil"
	"log"

	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
)

// Scanner reads a file into a string.
type Scanner struct {
}

// Scan reads a file named name in directory dir into a string.
// The contents of the file are stored in fileInfo.Content.
func (s *Scanner) Scan(dir string, name string) (error, string) {
	fpath := filepath.Join(dir, name)
	content, err := ioutil.ReadFile(fpath)
	return err, string(content)
}

func NewScanner() *Scanner {
	return &Scanner{}
}

type FileImportInfo struct {
	// The path being imported.
	Path string `json:"path"`
	// The source line number of the import.
	LineNumber uint32 `json:"lineno"`
}

type Parser struct {
}

func NewParser() *Parser {
	p := &Parser{}
	return p
}

// filenameToLoader takes in a filename, e.g. myFile.ts,
// and returns the appropriate esbuild loader for that file.
func filenameToLoader(filename string) api.Loader {
	ext := filepath.Ext(filename)
	switch ext {
	case ".ts":
		return api.LoaderTS
	case ".tsx":
		return api.LoaderTSX
	case ".js":
		return api.LoaderJSX
	case ".jsx":
		return api.LoaderJSX
	default:
		return api.LoaderTS
	}
}

// ParseImports returns all the imports from a file
// after parsing it.
func (p *Parser) ParseImports(filePath, source string) []FileImportInfo {
	imports := []FileImportInfo{}

	// Construct an esbuild plugin that pulls out all the imports.
	plugin := api.Plugin{
		Name: "GetImports",
		Setup: func(pluginBuild api.PluginBuild) {
			// callback is a handler for esbuild resolutions. This is how
			// we'll get access to every import in the file.
			callback := func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				// Add the imported string to our list of imports.
				imports = append(imports, FileImportInfo{
					Path:       args.Path,
					LineNumber: 0,
				})
				return api.OnResolveResult{
					// Mark the import as external so esbuild doesn't complain
					// about not being able to find the import.
					External: true,
				}, nil
			}

			// pluginBuild.OnResolve sets the plugin's onResolve callback to our custom callback.
			// Make sure to set Filter: ".*" so that our plugin runs on all imports.
			pluginBuild.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: ""}, callback)
		},
	}
	options := api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   source,
			Sourcefile: filePath,
			// The Loader determines how esbuild will parse the file.
			// We want to parse .ts files as typescript, .tsx files as .tsx, etc.
			Loader: filenameToLoader(filePath),
		},
		Plugins: []api.Plugin{
			plugin,
		},
		// Must set bundle to true so that esbuild actually does resolutions.
		Bundle: true,
	}
	result := api.Build(options)
	if len(result.Errors) > 0 {
		// Inform users that some files couldn't be fully parsed.
		// No need to crash the program though.
		log.Printf("Encountered errors parsing source %v: %v\n", filePath, result.Errors)
	}

	return imports
}
